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

package framework

import (
	"fmt"
	"os"
	"time"

	"github.com/karydia/karydia/pkg/client/clientset/versioned"
	"k8s.io/client-go/tools/clientcmd"

	clientset "github.com/karydia/karydia/pkg/client/clientset/versioned"
	corev1 "k8s.io/api/core/v1"
	apiextension "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	podv1 "k8s.io/kubernetes/pkg/api/v1/pod"

	"github.com/karydia/karydia/pkg/k8sutil"
)

type Framework struct {
	KubeClientset    kubernetes.Interface
	ApiExtClientset  apiextension.Interface
	KarydiaClientset versioned.Interface

	Namespace string
}

func Setup(server, kubeconfig, namespace string) (*Framework, error) {
	KubeClientset, err := k8sutil.Clientset(server, kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create clientset: %v", err)
	}
	cfg, err := clientcmd.BuildConfigFromFlags(server, kubeconfig)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to build kubeconfig:", err)
		os.Exit(1)
	}
	ApiExtClientset, err := apiextension.NewForConfig(cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to build api extension clientset:", err)
		os.Exit(1)
	}
	KarydiaClientset, err := clientset.NewForConfig(cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to build karydia clientset:", err)
		os.Exit(1)
	}

	return &Framework{
		KubeClientset:    KubeClientset,
		ApiExtClientset:  ApiExtClientset,
		KarydiaClientset: KarydiaClientset,
		Namespace:        namespace,
	}, nil
}

func (f *Framework) CreateServiceAccount(serviceAccountName string, namespace string) (*corev1.ServiceAccount, error) {
	serviceAccount := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceAccountName,
			Namespace: namespace,
		},
	}

	serviceAccount, err := f.KubeClientset.CoreV1().ServiceAccounts(namespace).Create(serviceAccount)
	if err != nil {
		return nil, fmt.Errorf("failed to create service account: %v", err)
	}
	return serviceAccount, nil
}

func (f *Framework) CreateNamespace() error {
	var objectMeta metav1.ObjectMeta
	objectMeta.Labels = map[string]string{
		"app": "karydia-e2e-test",
	}
	if f.Namespace == "" {
		objectMeta.GenerateName = "karydia-e2e-test-"
	} else {
		objectMeta.Name = f.Namespace
	}
	ns, err := f.KubeClientset.CoreV1().Namespaces().Create(&corev1.Namespace{
		ObjectMeta: objectMeta,
	})
	if err != nil {
		return fmt.Errorf("failed to create namespace '%s': %v", f.Namespace, err)
	}
	f.Namespace = ns.ObjectMeta.Name
	return nil
}

func (f *Framework) CreateTestNamespace() (*corev1.Namespace, error) {
	ns, err := f.KubeClientset.CoreV1().Namespaces().Create(&corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "karydia-e2e-test-",
			Labels: map[string]string{
				"app": "karydia-e2e-test",
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create namespace '%s': %v", f.Namespace, err)
	}
	return ns, nil
}

func (f *Framework) CreateTestNamespaceWithAnnotation(annotations map[string]string) (*corev1.Namespace, error) {
	ns, err := f.KubeClientset.CoreV1().Namespaces().Create(&corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "karydia-e2e-test-",
			Labels: map[string]string{
				"app": "karydia-e2e-test",
			},
			Annotations: annotations,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create namespace '%s': %v", f.Namespace, err)
	}
	return ns, nil
}

func (f *Framework) DeleteAll() error {
	zero := int64(0)
	ls := labels.SelectorFromSet(map[string]string{
		"app": "karydia-e2e-test",
	})
	namespaces, err := f.KubeClientset.CoreV1().Namespaces().List(metav1.ListOptions{
		LabelSelector: ls.String(),
	})
	if err != nil {
		return fmt.Errorf("failed to list namespaces: %v", err)
	}
	for _, ns := range namespaces.Items {
		name := ns.ObjectMeta.Name
		if err := f.KubeClientset.CoreV1().Namespaces().Delete(name, &metav1.DeleteOptions{
			GracePeriodSeconds: &zero,
		}); err != nil {
			return fmt.Errorf("failed to delete namespace '%s': %v", name, err)
		}
	}
	/* Delete single pod in default namespace */
	err = f.KubeClientset.CoreV1().Pods("default").Delete("karydia-e2e-test-pod", &metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete pod in default namespace")
	}

	return nil
}

func (f *Framework) WaitRunning(timeout time.Duration) error {
	return wait.Poll(3*time.Second, timeout, func() (bool, error) {
		ls := labels.SelectorFromSet(map[string]string{
			"app": "karydia",
		})
		pods, err := f.KubeClientset.CoreV1().Pods(f.Namespace).List(
			metav1.ListOptions{
				LabelSelector: ls.String(),
			},
		)
		if err != nil {
			return false, fmt.Errorf("failed to list pods: %v", err)
		}
		for _, pod := range pods.Items {
			if !podv1.IsPodReady(&pod) {
				return false, nil
			}
		}
		return true, nil
	})
}

func (f *Framework) WaitPodRunning(namespace, name string, timeout time.Duration) error {
	return wait.Poll(3*time.Second, timeout, func() (bool, error) {
		pod, err := f.KubeClientset.CoreV1().Pods(namespace).Get(name, metav1.GetOptions{})
		if err != nil {
			return false, fmt.Errorf("failed to get pods: %v", err)
		}
		if !podv1.IsPodReady(pod) {
			return false, nil
		}
		return true, nil
	})
}

func (f *Framework) WaitDefaultServiceAccountCreated(ns string, timeout time.Duration) error {
	return wait.Poll(200*time.Millisecond, timeout, func() (bool, error) {
		_, err := f.KubeClientset.CoreV1().ServiceAccounts(ns).Get("default", metav1.GetOptions{})
		if err != nil {
			return false, nil
		}
		return true, nil
	})
}

func (f *Framework) WaitNetworkPolicyCreated(ns string, name string, timeout time.Duration) error {
	return wait.Poll(200*time.Millisecond, timeout, func() (bool, error) {
		_, err := f.KubeClientset.NetworkingV1().NetworkPolicies(ns).Get(name, metav1.GetOptions{})
		if err != nil {
			return false, nil
		}
		return true, nil
	})
}

func (f *Framework) WaitEndpointCreated(ns string, name string, timeout time.Duration) error {
	return wait.Poll(200*time.Millisecond, timeout, func() (bool, error) {
		endpoint, err := f.KubeClientset.CoreV1().Endpoints(ns).Get(name, metav1.GetOptions{})
		if err != nil {
			return false, fmt.Errorf("failed to get endpoint: %v", err)
		}

		if len(endpoint.Subsets) == 0 {
			return false, nil
		}

		return true, nil
	})
}
