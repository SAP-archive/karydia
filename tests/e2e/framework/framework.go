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
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"net"
	"time"

	admissionv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	appsv1beta1 "k8s.io/api/apps/v1beta1"
	certsv1beta1 "k8s.io/api/certificates/v1beta1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/cert"
	podv1 "k8s.io/kubernetes/pkg/api/v1/pod"
	utilpointer "k8s.io/utils/pointer"

	"github.com/karydia/karydia/pkg/k8sutil"
)

type Framework struct {
	KubeClientset kubernetes.Interface

	Namespace string
}

func Setup(server, kubeconfig, namespace string) (*Framework, error) {
	KubeClientset, err := k8sutil.Clientset(server, kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create clientset: %v", err)
	}

	return &Framework{
		KubeClientset: KubeClientset,
		Namespace:     namespace,
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
		return fmt.Errorf("failed to create namespace %q: %v", f.Namespace, err)
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
		return nil, fmt.Errorf("failed to create namespace %q: %v", f.Namespace, err)
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
		return nil, fmt.Errorf("failed to create namespace %q: %v", f.Namespace, err)
	}
	return ns, nil
}

func (f *Framework) SetupKarydia() error {
	// Create service account
	sa := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "karydia",
			Namespace: f.Namespace,
		},
	}
	if _, err := f.KubeClientset.CoreV1().ServiceAccounts(f.Namespace).Create(sa); err != nil {
		return fmt.Errorf("failed to create service account: %v", err)
	}

	// Create deployment
	d := &appsv1beta1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "karydia-",
			Namespace:    f.Namespace,
		},
		Spec: appsv1beta1.DeploymentSpec{
			Replicas: utilpointer.Int32Ptr(1),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "karydia",
					},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: sa.ObjectMeta.Name,
					Containers: []corev1.Container{
						{
							Name:            "karydia",
							Image:           "karydia/karydia",
							ImagePullPolicy: corev1.PullIfNotPresent,
							Command: []string{
								"karydia",
								"runserver",
								"--tls-cert",
								"/etc/karydia/tls/cert.pem",
								"--tls-key",
								"/etc/karydia/tls/key.pem",
								"--enable-karydia-admission",
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "karydia-tls",
									MountPath: "/etc/karydia/tls",
								},
							},
							LivenessProbe: &corev1.Probe{
								Handler: corev1.Handler{
									HTTPGet: &corev1.HTTPGetAction{
										Path:   "/healthz",
										Port:   intstr.FromInt(33333),
										Scheme: corev1.URISchemeHTTPS,
									},
								},
								InitialDelaySeconds: 10,
								PeriodSeconds:       10,
								SuccessThreshold:    1,
								TimeoutSeconds:      1,
								FailureThreshold:    1,
							},
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 33333,
								},
							},
						},
						{
							Name:            "opa",
							Image:           "openpolicyagent/opa:0.10.2",
							ImagePullPolicy: corev1.PullIfNotPresent,
							Args: []string{
								"run",
								"--server",
								"--addr=http://127.0.0.1:8181",
							},
						},
						{
							Name:            "kube-mgmt",
							Image:           "openpolicyagent/kube-mgmt:0.7",
							ImagePullPolicy: corev1.PullIfNotPresent,
							Args: []string{
								"--replicate-cluster=v1/pods",
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "karydia-tls",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: "karydia-tls",
								},
							},
						},
					},
				},
			},
		},
	}
	if _, err := f.KubeClientset.AppsV1beta1().Deployments(f.Namespace).Create(d); err != nil {
		return fmt.Errorf("failed to create deployment: %v", err)
	}

	// Create service
	s := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "karydia",
			Namespace: f.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Port:       443,
					TargetPort: intstr.FromInt(33333),
				},
			},
			Selector: map[string]string{
				"app": "karydia",
			},
		},
	}
	if _, err := f.KubeClientset.CoreV1().Services(f.Namespace).Create(s); err != nil {
		return fmt.Errorf("failed to create service: %v", err)
	}

	// Create cluster role binding for karydia service account
	crb := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "karydia-e2e-test-",
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      "karydia",
				Namespace: f.Namespace,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     "cluster-admin",
		},
	}

	if _, err := f.KubeClientset.RbacV1().ClusterRoleBindings().Create(crb); err != nil {
		return fmt.Errorf("failed to create cluster role binding for karydia: %v", err)
	}

	return nil
}

func (f *Framework) ConfigureWebhook() error {
	// Partially adapted from
	// https://github.com/kubernetes/kubernetes/blob/master/test/e2e/auth/certificates.go

	privateKey, err := cert.NewPrivateKey()
	if err != nil {
		return err
	}

	// Create a CSR

	dnsSANs := []string{
		"localhost",
		"karydia",
		fmt.Sprintf("karydia.%s", f.Namespace),
		fmt.Sprintf("karydia.%s.svc", f.Namespace),
	}
	ipSANs := []net.IP{
		net.IPv4(127, 0, 0, 1),
	}
	csrBytes, err := cert.MakeCSR(privateKey, &pkix.Name{CommonName: "karydia", Organization: []string{}}, dnsSANs, ipSANs)
	if err != nil {
		return err
	}

	csr := &certsv1beta1.CertificateSigningRequest{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "karydia-e2e-test-",
		},
		Spec: certsv1beta1.CertificateSigningRequestSpec{
			Usages: []certsv1beta1.KeyUsage{
				certsv1beta1.UsageSigning,
				certsv1beta1.UsageKeyEncipherment,
				certsv1beta1.UsageServerAuth,
			},
			Request: csrBytes,
		},
	}

	csrs := f.KubeClientset.CertificatesV1beta1().CertificateSigningRequests()

	csr, err = csrs.Create(csr)
	if err != nil {
		return fmt.Errorf("failed to create kubernetes CSR: %v", err)
	}

	csrName := csr.Name

	// Approve the CSR and get the certificate

	if err := wait.Poll(5*time.Second, time.Minute, func() (bool, error) {
		csr.Status.Conditions = []certsv1beta1.CertificateSigningRequestCondition{
			{
				Type:    certsv1beta1.CertificateApproved,
				Reason:  "e2e",
				Message: "karydia e2e test",
			},
		}
		csr, err = csrs.UpdateApproval(csr)
		if err != nil {
			csr, _ = csrs.Get(csrName, metav1.GetOptions{})
			return false, nil
		}
		return true, nil
	}); err != nil {
		return fmt.Errorf("failed to approve CSR: %v", err)
	}

	if err := wait.Poll(5*time.Second, time.Minute, func() (bool, error) {
		csr, err = csrs.Get(csrName, metav1.GetOptions{})
		if err != nil {
			return false, nil
		}
		if len(csr.Status.Certificate) == 0 {
			return false, nil
		}
		return true, nil
	}); err != nil {
		return fmt.Errorf("certificate still not signed after one minute")
	}

	// Get cert and key PEM data

	certPEM := csr.Status.Certificate

	privateKeyBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	privateKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privateKeyBytes,
	})

	// Create a kubernetes secret with cert and key

	tlsSecret := &corev1.Secret{
		Type: corev1.SecretTypeOpaque,
		ObjectMeta: metav1.ObjectMeta{
			Name:      "karydia-tls",
			Namespace: f.Namespace,
		},
		Data: map[string][]byte{
			"key.pem":  privateKeyPEM,
			"cert.pem": certPEM,
		},
	}

	if _, err := f.KubeClientset.CoreV1().Secrets(f.Namespace).Create(tlsSecret); err != nil {
		return fmt.Errorf("failed to create kubernetes secret: %v", err)
	}

	// Get kube-apiserver CA bundle

	caBundle, err := f.KubeClientset.CoreV1().ConfigMaps("kube-system").Get("extension-apiserver-authentication", metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get kubernetes CA bundle: %v", err)
	}

	caCertPEM := caBundle.Data["client-ca-file"]

	// Finally, configure karydia webhook

	webhookName := "karydiamutating.e2e.test"
	webhookPath := "/webhook/mutating"
	mutatingWebhookConfig := &admissionv1beta1.MutatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: webhookName,
		},
		Webhooks: []admissionv1beta1.Webhook{
			{
				Name: webhookName,
				Rules: []admissionv1beta1.RuleWithOperations{{
					Operations: []admissionv1beta1.OperationType{admissionv1beta1.OperationAll},
					Rule: admissionv1beta1.Rule{
						APIGroups:   []string{"*"},
						APIVersions: []string{"*"},
						Resources:   []string{"*/*"},
					},
				}},
				ClientConfig: admissionv1beta1.WebhookClientConfig{
					Service: &admissionv1beta1.ServiceReference{
						Namespace: f.Namespace,
						Name:      "karydia",
						Path:      &webhookPath,
					},
					CABundle: []byte(caCertPEM),
				},
			},
		},
	}

	webhookName = "karydiavalidating.e2e.test"
	valWebhookPath := "/webhook/validating"
	validatingWebhookConfig := &admissionv1beta1.ValidatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: webhookName,
		},
		Webhooks: []admissionv1beta1.Webhook{
			{
				Name: webhookName,
				Rules: []admissionv1beta1.RuleWithOperations{{
					Operations: []admissionv1beta1.OperationType{admissionv1beta1.OperationAll},
					Rule: admissionv1beta1.Rule{
						APIGroups:   []string{"*"},
						APIVersions: []string{"*"},
						Resources:   []string{"*/*"},
					},
				}},
				ClientConfig: admissionv1beta1.WebhookClientConfig{
					Service: &admissionv1beta1.ServiceReference{
						Namespace: f.Namespace,
						Name:      "karydia",
						Path:      &valWebhookPath,
					},
					CABundle: []byte(caCertPEM),
				},
			},
		},
	}

	if _, err := f.KubeClientset.AdmissionregistrationV1beta1().MutatingWebhookConfigurations().Create(mutatingWebhookConfig); err != nil {
		return fmt.Errorf("failed to register mutating webhook: %v", err)
	}

	if _, err := f.KubeClientset.AdmissionregistrationV1beta1().ValidatingWebhookConfigurations().Create(validatingWebhookConfig); err != nil {
		return fmt.Errorf("failed to register validating webhook: %v", err)
	}

	return nil
}

func (f *Framework) DeleteWebhook() error {
	errMut := f.KubeClientset.AdmissionregistrationV1beta1().MutatingWebhookConfigurations().Delete("karydiamutating.e2e.test", &metav1.DeleteOptions{})
	errVal := f.KubeClientset.AdmissionregistrationV1beta1().ValidatingWebhookConfigurations().Delete("karydiavalidating.e2e.test", &metav1.DeleteOptions{})
	if errMut != nil || errVal != nil {
		return fmt.Errorf("Deletion of webhooks failed: %d %d", errMut, errVal)
	}
	return nil
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
			return fmt.Errorf("failed to delete namespace %q: %v", name, err)
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
