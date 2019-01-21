package karydia

import (
	"fmt"

	"k8s.io/api/admission/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (k *KarydiaAdmission) admitPodSeccompProfile(ar v1beta1.AdmissionReview, mutationAllowed bool, pod *corev1.Pod) ([]string, []string, error) {
	var patches, validationErrors []string

	namespaceRequest := ar.Request.Namespace
	if namespaceRequest == "" {
		return nil, nil, fmt.Errorf("received request with empty namespace")
	}
	namespace, err := k.kubeClientset.CoreV1().Namespaces().Get(namespaceRequest, metav1.GetOptions{})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to determine pod's namespace: %v", err)
	}

	seccompProfile, doCheck := namespace.ObjectMeta.Annotations["karydia.gardener.cloud/seccompProfile"]
	if doCheck {
		seccompPod, ok := pod.ObjectMeta.Annotations["seccomp.security.alpha.kubernetes.io/pod"]
		if !ok && mutationAllowed {
			if len(pod.ObjectMeta.Annotations) == 0 {
				// If no annotation object exists yet, we have
				// to create it. Otherwise we will encounter
				// the following error:
				// 'jsonpatch add operation does not apply: doc is missing path: "/metadata/annotations..."'
				patches = append(patches, fmt.Sprintf(`{"op": "add", "path": "/metadata/annotations", "value": {"%s": "%s"}}`, "seccomp.security.alpha.kubernetes.io/pod", seccompProfile))
			} else {
				patches = append(patches, fmt.Sprintf(`{"op": "add", "path": "/metadata/annotations/%s", "value": "%s"}`, "seccomp.security.alpha.kubernetes.io~1pod", seccompProfile))
			}
		} else if seccompPod != seccompProfile {
			validationErrorMsg := fmt.Sprintf("seccomp profile ('seccomp.security.alpha.kubernetes.io/pod') must be '%s'", seccompProfile)
			validationErrors = append(validationErrors, validationErrorMsg)
		}
	}

	return patches, validationErrors, nil
}
