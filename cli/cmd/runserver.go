// Copyright (C) 2019 SAP SE or an SAP affiliate company. All rights reserved.
// This file is licensed under the Apache Software License, v. 2 except as
// noted otherwise in the LICENSE file.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	karydiainformers "github.com/karydia/karydia/pkg/client/informers/externalversions"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	karydiaadmission "github.com/karydia/karydia/pkg/admission/karydia"
	clientset "github.com/karydia/karydia/pkg/client/clientset/versioned"
	"github.com/karydia/karydia/pkg/controller"
	"github.com/karydia/karydia/pkg/k8sutil"
	"github.com/karydia/karydia/pkg/server"
	"github.com/karydia/karydia/pkg/util/tls"
	"github.com/karydia/karydia/pkg/webhook"
)

const resyncInterval = 30 * time.Second

var runserverCmd = &cobra.Command{
	Use:   "runserver",
	Short: "Run the karydia server",
	Run:   runserverFunc,
}

func init() {
	rootCmd.AddCommand(runserverCmd)

	runserverCmd.Flags().String("config", "karydia-config", "Custom Resource where to load the configuration from, in the format <name>")

	runserverCmd.Flags().String("addr", "0.0.0.0:33333", "Address to listen on")

	runserverCmd.Flags().Bool("enable-karydia-admission", false, "Enable the Karydia admission plugin")

	runserverCmd.Flags().String("tls-cert", "cert.pem", "Path to TLS certificate file")
	runserverCmd.Flags().String("tls-key", "key.pem", "Path to TLS private key file")

	runserverCmd.Flags().String("kubeconfig", "", "Path to the kubeconfig file")
	runserverCmd.Flags().String("server", "", "The address and port of the Kubernetes API server")

	runserverCmd.Flags().Bool("enable-default-network-policy", false, "Whether to install a default network policy in namespaces")
	runserverCmd.Flags().StringSlice("default-network-policy-excludes", []string{"kube-system"}, "List of namespaces where the default network policy should not be installed")
}

func runserverFunc(cmd *cobra.Command, args []string) {
	var (
		enableController           bool
		enableDefaultNetworkPolicy = viper.GetBool("enable-default-network-policy")
		enableKarydiaAdmission     = viper.GetBool("enable-karydia-admission")
		kubeInformerFactory        kubeinformers.SharedInformerFactory
		karydiaInformerFactory     karydiainformers.SharedInformerFactory
		karydiaControllers         = []controller.ControllerInterface{}
	)
	if enableDefaultNetworkPolicy {
		enableController = true
	}

	tlsConfig, err := tls.CreateTLSConfig(viper.GetString("tls-cert"), viper.GetString("tls-key"))
	if err != nil {
		log.Fatalln("Failed to create TLS config:", err)
	}

	ctx, cancelCtx := context.WithCancel(context.Background())
	defer cancelCtx()

	webHook, err := webhook.New(&webhook.Config{})
	if err != nil {
		log.Fatalln("Failed to load webhook:", err)
	}

	kubeConfig := viper.GetString("kubeconfig")
	kubeServer := viper.GetString("server")

	kubeClientset, err := k8sutil.Clientset(kubeServer, kubeConfig)
	if err != nil {
		log.Fatalln("Failed to create clientset:", err)
	}

	cfg, err := clientcmd.BuildConfigFromFlags(kubeServer, kubeConfig)
	if err != nil {
		log.Fatalln("Failed to build kubeconfig:", err)
	}
	karydiaClientset, err := clientset.NewForConfig(cfg)
	if err != nil {
		log.Fatalln("Failed to build karydia clientset:", err)
	}

	karydiaConfig, err := karydiaClientset.KarydiaV1alpha1().KarydiaConfigs().Get(viper.GetString("config"), metav1.GetOptions{})

	if err != nil {
		log.Fatalln("Failed to load karydia config:", err)
	}
	log.Infoln("KarydiaConfig Name:", karydiaConfig.Name)
	log.Infoln("KarydiaConfig AutomountServiceAccountToken:", karydiaConfig.Spec.AutomountServiceAccountToken)
	log.Infoln("KarydiaConfig SeccompProfile:", karydiaConfig.Spec.SeccompProfile)
	log.Infoln("KarydiaConfig NetworkPolicy:", karydiaConfig.Spec.NetworkPolicy)
	log.Infoln("KarydiaConfig PodSecurityContext:", karydiaConfig.Spec.PodSecurityContext)

	if enableKarydiaAdmission {
		karydiaAdmission, err := karydiaadmission.New(&karydiaadmission.Config{
			KubeClientset: kubeClientset,
			KarydiaConfig: karydiaConfig,
		})
		if err != nil {
			log.Fatalln("Failed to load karydia admission:", err)
		}

		webHook.RegisterAdmissionPlugin(karydiaAdmission)
		karydiaControllers = append(karydiaControllers, karydiaAdmission)
	}

	defaultNetworkPolicies := make(map[string]*networkingv1.NetworkPolicy)
	if enableDefaultNetworkPolicy {

		karydiaDefaultNetworkPolicyName := karydiaConfig.Spec.NetworkPolicy
		karydiaDefaulNetworkPolicy, err := karydiaClientset.KarydiaV1alpha1().KarydiaNetworkPolicies().Get(karydiaDefaultNetworkPolicyName, metav1.GetOptions{})
		if err != nil {
			log.Fatalln("Failed to load KarydiaDefaultNetworkPolicy:", err)
		}
		var policy networkingv1.NetworkPolicy
		policy.Spec = *karydiaDefaulNetworkPolicy.Spec.DeepCopy()
		policy.Name = karydiaDefaultNetworkPolicyName
		defaultNetworkPolicies[karydiaDefaultNetworkPolicyName] = &policy
	}

	var reconciler *controller.NetworkpolicyReconciler
	if enableController {
		cfg, err := clientcmd.BuildConfigFromFlags(kubeServer, kubeConfig)
		if err != nil {
			log.Fatalln("error building kubeconfig:", err)
		}
		kubeClientset, err := kubernetes.NewForConfig(cfg)
		if err != nil {
			log.Fatalln("error building kubernetes clientset:", err)
		}
		kubeInformerFactory = kubeinformers.NewSharedInformerFactory(kubeClientset, resyncInterval)
		namespaceInformer := kubeInformerFactory.Core().V1().Namespaces()
		networkPolicyInformer := kubeInformerFactory.Networking().V1().NetworkPolicies()
		reconciler = controller.NewNetworkpolicyReconciler(kubeClientset, karydiaClientset, networkPolicyInformer, namespaceInformer, defaultNetworkPolicies, karydiaConfig.Spec.NetworkPolicy, viper.GetStringSlice("default-network-policy-excludes"))
		karydiaControllers = append(karydiaControllers, reconciler)
	}

	serverConfig := &server.Config{
		Addr:      viper.GetString("addr"),
		TLSConfig: tlsConfig,
	}

	s, err := server.New(serverConfig, webHook)
	if err != nil {
		log.Fatalln("Failed to load server:", err)
	}

	karydiaInformerFactory = karydiainformers.NewSharedInformerFactory(karydiaClientset, resyncInterval)
	karydiaConfigReconciler := controller.NewConfigReconciler(*karydiaConfig, karydiaControllers, karydiaClientset, karydiaInformerFactory.Karydia().V1alpha1().KarydiaConfigs())

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := s.ListenAndServe(); err != http.ErrServerClosed {
			log.Errorln("HTTP ListenAndServe error:", err)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		<-ctx.Done()

		shutdownCtx, cancelShutdownCtx := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancelShutdownCtx()

		if err := s.Shutdown(shutdownCtx); err != nil {
			log.Errorln("HTTP Shutdown error:", err)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		karydiaInformerFactory.Start(ctx.Done())
		if err := karydiaConfigReconciler.Run(2, ctx.Done()); err != nil {
			log.Errorln("Error running config reconciler:", err)
		}
	}()

	if enableController {
		wg.Add(1)
		go func() {
			defer wg.Done()
			kubeInformerFactory.Start(ctx.Done())
			if err := reconciler.Run(2, ctx.Done()); err != nil {
				log.Errorln("Error running controller:", err)
			}
		}()
	}

	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

		<-sigChan

		log.Infoln("Received signal, shutting down gracefully ...")

		cancelCtx()

		<-sigChan

		log.Infoln("Received second signal - aborting")
		os.Exit(1)
	}()

	<-ctx.Done()

	wg.Wait()
}
