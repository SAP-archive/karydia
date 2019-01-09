package cmd

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/ghodss/yaml"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	karydiaadmission "github.com/kinvolk/karydia/pkg/admission/karydia"
	kspadmission "github.com/kinvolk/karydia/pkg/admission/karydiasecuritypolicy"
	opaadmission "github.com/kinvolk/karydia/pkg/admission/opa"
	"github.com/kinvolk/karydia/pkg/controller"
	"github.com/kinvolk/karydia/pkg/k8sutil"
	"github.com/kinvolk/karydia/pkg/server"
	"github.com/kinvolk/karydia/pkg/util/tls"
	"github.com/kinvolk/karydia/pkg/webhook"
)

var runserverCmd = &cobra.Command{
	Use:   "runserver",
	Short: "Run the karydia server",
	Run:   runserverFunc,
}

func init() {
	rootCmd.AddCommand(runserverCmd)

	runserverCmd.Flags().String("addr", "0.0.0.0:33333", "Address to listen on")

	runserverCmd.Flags().Bool("enable-opa-admission", false, "Enable the OPA admission plugin")
	runserverCmd.Flags().Bool("enable-ksp-admission", false, "Enable the KarydiaSecurityPolicy admission plugin")
	runserverCmd.Flags().Bool("enable-karydia-admission", false, "Enable the Karydia admission plugin")

	// TODO(schu): the '/v1' currently is required since the OPA package
	// from kubernetes-policy-controller that we use does not include that
	// in the URL when sending requests.
	// IMHO it should since the package should set the API version
	// it's written for.
	runserverCmd.Flags().String("opa-api-endpoint", "http://127.0.0.1:8181/v1", "Open Policy Agent API endpoint")

	runserverCmd.Flags().String("tls-cert", "cert.pem", "Path to TLS certificate file")
	runserverCmd.Flags().String("tls-key", "key.pem", "Path to TLS private key file")

	runserverCmd.Flags().String("kubeconfig", "", "Path to the kubeconfig file")
	runserverCmd.Flags().String("server", "", "The address and port of the Kubernetes API server")

	runserverCmd.Flags().Bool("enable-default-network-policy", false, "Whether to install a default network policy in namespaces")
	runserverCmd.Flags().StringSlice("default-network-policy-excludes", []string{"kube-system"}, "List of namespaces where the default network policy should not be installed")
	runserverCmd.Flags().String("default-network-policy-configmap", "kube-system:karydia-default-network-policy", "Configmap where to load the default network policy from, in the format <namespace>:<name>")

	runserverCmd.Flags().Bool("karydia-admission-disable-automount-service-account-token", true, "Whether to set `automountServiceaAccountToken` to `false` by default (can be overwritten on a per-namespace basis)")
}

func runserverFunc(cmd *cobra.Command, args []string) {
	var (
		enableController           bool
		enableDefaultNetworkPolicy = viper.GetBool("enable-default-network-policy")
		enableKSPAdmission         = viper.GetBool("enable-ksp-admission")
		enableOPAAdmission         = viper.GetBool("enable-opa-admission")
		enableKarydiaAdmission     = viper.GetBool("enable-karydia-admission")
	)
	if enableDefaultNetworkPolicy || enableKSPAdmission {
		enableController = true
	}

	tlsConfig, err := tls.CreateTLSConfig(viper.GetString("tls-cert"), viper.GetString("tls-key"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create TLS config: %v\n", err)
		os.Exit(1)
	}

	ctx, cancelCtx := context.WithCancel(context.Background())
	defer cancelCtx()

	webHook, err := webhook.New(&webhook.Config{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load webhook: %v\n", err)
		os.Exit(1)
	}

	kubeConfig := viper.GetString("kubeconfig")
	kubeServer := viper.GetString("server")

	kubeClientset, err := k8sutil.Clientset(kubeServer, kubeConfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create clientset: %v\n", err)
		os.Exit(1)
	}

	if enableKarydiaAdmission {
		karydiaAdmission, err := karydiaadmission.New(
			&karydiaadmission.Config{
				KubeClientset: kubeClientset,
			},
			&karydiaadmission.Policy{
				DisableAutomountServiceAccountToken: viper.GetBool("karydia-admission-disable-automount-service-account-token"),
			},
		)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to load karydia admission: %v\n", err)
			os.Exit(1)
		}

		webHook.RegisterAdmissionPlugin(karydiaAdmission)
	}

	var defaultNetworkPolicy *networkingv1.NetworkPolicy
	if enableDefaultNetworkPolicy {
		defaultNetworkPolicyIdentifier := viper.GetString("default-network-policy-configmap")
		group := strings.SplitN(defaultNetworkPolicyIdentifier, ":", 2)
		if len(group) < 2 {
			fmt.Fprintf(os.Stderr, "default-network-policy-configmap must be provided in format <namespace>:<name>, got %q\n", defaultNetworkPolicyIdentifier)
			os.Exit(1)
		}
		namespace := group[0]
		name := group[1]
		networkPolicyConfigmap, err := kubeClientset.CoreV1().ConfigMaps(namespace).Get(name, metav1.GetOptions{})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to get default network policy configmap %q in namespace %q: %v\n", name, namespace, err)
			os.Exit(1)
		}
		var policy networkingv1.NetworkPolicy
		if err := yaml.Unmarshal([]byte(networkPolicyConfigmap.Data["policy"]), &policy); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to unmarshal default network policy configmap ('%s:%s') into network policy object: %v\n", namespace, name, err)
			os.Exit(1)
		}
		defaultNetworkPolicy = &policy
	}

	var ctrler *controller.Controller
	if enableController {
		ctrler, err = controller.New(ctx, &controller.Config{
			DefaultNetworkPolicy:         defaultNetworkPolicy,
			DefaultNetworkPolicyExcludes: viper.GetStringSlice("default-network-policy-excludes"),

			Kubeconfig: kubeConfig,
			MasterURL:  kubeServer,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to load controller: %v\n", err)
			os.Exit(1)
		}
	}

	if enableKSPAdmission {
		kspAdmission, err := kspadmission.New()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to load karydia security policy admission: %v\n", err)
			os.Exit(1)
		}

		rbacAuthorizer, err := k8sutil.NewRBACAuthorizer(ctrler.KubeInformerFactory())
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to load rbac authorizer: %v\n", err)
			os.Exit(1)
		}

		kspAdmission.SetAuthorizer(rbacAuthorizer)
		kspAdmission.SetExternalInformerFactory(ctrler.KarydiaInformerFactory())

		webHook.RegisterAdmissionPlugin(kspAdmission)
	}

	if enableOPAAdmission {
		opaAdmission, err := opaadmission.New(&opaadmission.Config{
			OPAURL: viper.GetString("opa-api-endpoint"),
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to load opa admission: %v\n", err)
			os.Exit(1)
		}

		webHook.RegisterAdmissionPlugin(opaAdmission)
	}

	serverConfig := &server.Config{
		Addr:      viper.GetString("addr"),
		TLSConfig: tlsConfig,
	}

	s, err := server.New(serverConfig, webHook)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load server: %v\n", err)
		os.Exit(1)
	}

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := s.ListenAndServe(); err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "HTTP ListenAndServe error: %v\n", err)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		<-ctx.Done()

		shutdownCtx, cancelShutdownCtx := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancelShutdownCtx()

		if err := s.Shutdown(shutdownCtx); err != nil {
			fmt.Fprintf(os.Stderr, "HTTP Shutdown error: %v\n", err)
		}
	}()

	if enableController {
		wg.Add(1)
		go func() {
			defer wg.Done()

			if err := ctrler.Run(2); err != nil {
				fmt.Fprintf(os.Stderr, "Error running controller: %v\n", err)
			}
		}()
	}

	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

		<-sigChan

		fmt.Println("Received signal, shutting down gracefully ...")

		cancelCtx()

		<-sigChan

		fmt.Println("Received second signal - aborting")
		os.Exit(1)
	}()

	<-ctx.Done()

	wg.Wait()
}
