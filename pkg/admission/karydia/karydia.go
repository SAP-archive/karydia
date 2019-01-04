package karydia

import (
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
	"k8s.io/api/admission/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/kinvolk/karydia/pkg/k8sutil"
	"github.com/kinvolk/karydia/pkg/k8sutil/scheme"
)

var (
	resourceNamespace = metav1.GroupVersionResource{Group: "", Version: "v1", Resource: "namespaces"}
	resourcePod       = metav1.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}
)

type KarydiaAdmission struct {
	logger *logrus.Logger

	kubeClientset *kubernetes.Clientset
}

type Config struct {
	KubeClientset *kubernetes.Clientset
}

func New(config *Config) (*KarydiaAdmission, error) {
	logger := logrus.New()
	logger.Level = logrus.InfoLevel

	return &KarydiaAdmission{
		logger: logger,

		kubeClientset: config.KubeClientset,
	}, nil
}

func (k *KarydiaAdmission) Admit(ar v1beta1.AdmissionReview, mutationAllowed bool) *v1beta1.AdmissionResponse {
	if ignore, err := shouldIgnore(ar); err != nil {
		k.logger.Errorf("failed to determine if admission request should be ignored: %v", err)
		return &v1beta1.AdmissionResponse{
			Allowed: true,
		}
	} else if ignore {
		return &v1beta1.AdmissionResponse{
			Allowed: true,
		}
	}

	var response *v1beta1.AdmissionResponse

	switch ar.Request.Resource {
	case resourcePod:
		response = k.AdmitPod(ar, mutationAllowed)
	default:
		response = &v1beta1.AdmissionResponse{
			Allowed: true,
		}
	}

	return response
}

func (k *KarydiaAdmission) AdmitPod(ar v1beta1.AdmissionReview, mutationAllowed bool) *v1beta1.AdmissionResponse {
	var patches, validationErrors []string

	raw := ar.Request.Object.Raw
	pod := corev1.Pod{}
	deserializer := scheme.Codecs.UniversalDeserializer()
	if _, _, err := deserializer.Decode(raw, nil, &pod); err != nil {
		return k8sutil.ErrToAdmissionResponse(fmt.Errorf("failed to decode object: %v", err))
	}

	namespaceRequest := ar.Request.Namespace
	if namespaceRequest == "" {
		return k8sutil.ErrToAdmissionResponse(fmt.Errorf("received request with empty namespace"))
	}
	namespace, err := k.kubeClientset.CoreV1().Namespaces().Get(namespaceRequest, metav1.GetOptions{})
	if err != nil {
		return k8sutil.ErrToAdmissionResponse(fmt.Errorf("failed to determine pod's namespace: %v", err))
	}

	automountServiceAccountToken, doCheck := namespace.ObjectMeta.Annotations["karydia.gardener.cloud/automountServiceAccountToken"]
	if doCheck {
		if automountServiceAccountToken == "forbidden" {
			if doesAutomountServiceAccountToken(&pod) {
				validationErrors = append(validationErrors, "automount of service account not allowed")
			}
		} else if automountServiceAccountToken == "non-default" {
			if doesAutomountServiceAccountToken(&pod) && pod.Spec.ServiceAccountName == "default" {
				validationErrors = append(validationErrors, "automount of service account 'default' not allowed")
			}
		}
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

	if len(validationErrors) > 0 {
		return &v1beta1.AdmissionResponse{
			Allowed: false,
			Result: &metav1.Status{
				Message: fmt.Sprintf("%+v", validationErrors),
			},
		}
	}

	response := &v1beta1.AdmissionResponse{
		Allowed: true,
	}

	if len(patches) > 0 {
		patchesStr := strings.Join(patches, ",")
		patchType := v1beta1.PatchTypeJSONPatch

		response.Patch = []byte(fmt.Sprintf("[%s]", patchesStr))
		response.PatchType = &patchType
	}

	return response
}

func doesAutomountServiceAccountToken(pod *corev1.Pod) bool {
	return pod.Spec.AutomountServiceAccountToken == nil || *pod.Spec.AutomountServiceAccountToken
}

func shouldIgnore(ar v1beta1.AdmissionReview) (bool, error) {
	// Right now we only care about 'CREATE' events. Needs to be
	// updated depending on the kind of admission requests that
	// `KarydiaAdmission` should handle in this package.
	// https://github.com/kubernetes/api/blob/kubernetes-1.12.2/admission/v1beta1/types.go#L118-L127
	if ar.Request.Operation != v1beta1.Create {
		return true, nil
	}
	return false, nil
}
