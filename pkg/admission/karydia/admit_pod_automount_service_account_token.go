package karydia

import (
	"fmt"

	"k8s.io/api/admission/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (k *KarydiaAdmission) admitPodAutomountServiceAccountToken(ar v1beta1.AdmissionReview, mutationAllowed bool, pod *corev1.Pod) ([]patch, []string, error) {
	var validationErrors []string

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

	automountServiceAccountToken, doCheck := namespace.ObjectMeta.Annotations["karydia.gardener.cloud/automountServiceAccountToken"]
	if doCheck {
		if automountServiceAccountToken == "forbidden" {
			if doesAutomountServiceAccountToken(pod) {
				validationErrors = append(validationErrors, "automount of service account not allowed")
			}
		} else if automountServiceAccountToken == "non-default" {
			if doesAutomountServiceAccountToken(pod) && pod.Spec.ServiceAccountName == "default" {
				validationErrors = append(validationErrors, "automount of service account 'default' not allowed")
			}
		}
	}

	return nil, validationErrors, nil
}
