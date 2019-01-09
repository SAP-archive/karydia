package karydia

import (
	"fmt"

	"k8s.io/api/admission/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (k *KarydiaAdmission) admitPodAutomountServiceAccountToken(ar v1beta1.AdmissionReview, mutationAllowed bool, pod *corev1.Pod) ([]patch, []string, error) {
	var (
		patches          []patch
		validationErrors []string
	)

	if ignore, err := shouldIgnore(ar); err != nil {
		return nil, nil, fmt.Errorf("failed to determine if admission request should be ignored: %v", err)
	} else if ignore {
		return nil, nil, nil
	}

	namespaceRequest := ar.Request.Namespace
	if namespaceRequest == "" {
		return nil, nil, fmt.Errorf("received request with empty namespace")
	}
	namespace, err := k.kubeClientset.CoreV1().Namespaces().Get(namespaceRequest, metav1.GetOptions{})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to determine pod's namespace: %v", err)
	}

	if doesAutomountServiceAccountToken(pod) {
		if k.policy.DisableAutomountServiceAccountToken {
			patches = append(patches, patch{
				Op:    "replace",
				Path:  "/spec/automountServiceAccountToken",
				Value: false,
			})
		}

		automountServiceAccountToken, doCheck := namespace.ObjectMeta.Annotations["karydia.gardener.cloud/automountServiceAccountToken"]
		if doCheck {
			if automountServiceAccountToken == "forbidden" {
				validationErrors = append(validationErrors, "automount of service account not allowed")
			} else if automountServiceAccountToken == "non-default" {
				if pod.Spec.ServiceAccountName == "default" {
					validationErrors = append(validationErrors, "automount of service account 'default' not allowed")
				}
			}
		}
	}

	return patches, validationErrors, nil
}
