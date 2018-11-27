package cmd

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

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

	runserverCmd.Flags().Bool("enable-opa", false, "Enable OPA module")
	runserverCmd.Flags().Bool("enable-ksp", false, "Enable KSP module")

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
}

func runserverFunc(cmd *cobra.Command, args []string) {
	tlsConfig, err := tls.CreateTLSConfig(viper.GetString("tls-cert"), viper.GetString("tls-key"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create TLS config: %v\n", err)
		os.Exit(1)
	}

	ctx, cancelCtx := context.WithCancel(context.Background())
	defer cancelCtx()

	controller, err := controller.New(ctx, &controller.Config{viper.GetString("kubeconfig"), viper.GetString("server"), nil})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load controller: %v\n", err)
		os.Exit(1)
	}

	var kspAdmission *kspadmission.KarydiaSecurityPolicyAdmission
	if viper.GetBool("enable-ksp") {
		kspAdmission, err = kspadmission.New()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to load karydia security policy admission: %v\n", err)
			os.Exit(1)
		}

		rbacAuthorizer, err := k8sutil.NewRBACAuthorizer(controller.KubeInformerFactory())
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to load rbac authorizer: %v\n", err)
			os.Exit(1)
		}

		kspAdmission.SetAuthorizer(rbacAuthorizer)
		kspAdmission.SetExternalInformerFactory(controller.KarydiaInformerFactory())
	}

	var opaAdmission *opaadmission.OPAAdmission
	if viper.GetBool("enable-opa") {
		opaAdmission, err = opaadmission.New(&opaadmission.Config{
			OPAURL: viper.GetString("opa-api-endpoint"),
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to load opa admission: %v\n", err)
			os.Exit(1)
		}
	}

	webHook, err := webhook.New(&webhook.Config{
		KSPAdmission: kspAdmission,
		OPAAdmission: opaAdmission,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load webhook: %v\n", err)
		os.Exit(1)
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

	wg.Add(1)
	go func() {
		defer wg.Done()

		if err := controller.Run(2); err != nil {
			fmt.Fprintf(os.Stderr, "Error running controller: %v\n", err)
		}
	}()

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
