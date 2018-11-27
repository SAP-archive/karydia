package k8sutil

import (
	"fmt"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func Clientset(server, kubeconfig string) (*kubernetes.Clientset, error) {
	cfg, err := clientcmd.BuildConfigFromFlags(server, kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("error building kubeconfig: %v", err)
	}
	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("error building kubernetes clientset: %v", err)
	}
	return clientset, nil
}
