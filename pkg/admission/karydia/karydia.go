package karydia

import (
	"encoding/json"
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
	resourcePod = metav1.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}
	kindPod     = metav1.GroupVersionKind{Group: "", Version: "v1", Kind: "Pod"}
)

type KarydiaAdmission struct {
	logger *logrus.Logger

	kubeClientset *kubernetes.Clientset
}

type Config struct {
	KubeClientset *kubernetes.Clientset
}

type patch struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value"`
}

func (p *patch) String() (string, error) {
	data, err := json.Marshal(p)
	if err != nil {
		return "", err
	}
	return string(data), nil
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

	if ar.Request.Kind == kindPod && ar.Request.Resource == resourcePod {
		response = k.AdmitPod(ar, mutationAllowed)
	} else {
		response = &v1beta1.AdmissionResponse{
			Allowed: true,
		}
	}

	return response
}

func (k *KarydiaAdmission) AdmitPod(ar v1beta1.AdmissionReview, mutationAllowed bool) *v1beta1.AdmissionResponse {
	var (
		collectedPatches          []patch
		collectedValidationErrors []string
	)

	raw := ar.Request.Object.Raw
	pod := corev1.Pod{}
	deserializer := scheme.Codecs.UniversalDeserializer()
	if _, _, err := deserializer.Decode(raw, nil, &pod); err != nil {
		e := fmt.Errorf("failed to decode object: %v", err)
		k.logger.Errorf("%v", e)
		return k8sutil.ErrToAdmissionResponse(e)
	}

	type HandlerFunc func(ar v1beta1.AdmissionReview, mutationAllowed bool, pod *corev1.Pod) ([]patch, []string, error)
	for _, f := range []HandlerFunc{
		k.admitPodAutomountServiceAccountToken,
		k.admitPodSeccompProfile,
	} {
		patches, validationErrors, err := f(ar, mutationAllowed, &pod)
		if err != nil {
			k.logger.Errorf("%v", err)
			return k8sutil.ErrToAdmissionResponse(err)
		}
		collectedPatches = append(collectedPatches, patches...)
		collectedValidationErrors = append(collectedValidationErrors, validationErrors...)
	}

	if len(collectedValidationErrors) > 0 {
		return &v1beta1.AdmissionResponse{
			Allowed: false,
			Result: &metav1.Status{
				Message: fmt.Sprintf("%+v", collectedValidationErrors),
			},
		}
	}

	response := &v1beta1.AdmissionResponse{
		Allowed: true,
	}

	if len(collectedPatches) > 0 {
		collectedPatchesStr := make([]string, len(collectedPatches))
		for i, p := range collectedPatches {
			str, err := p.String()
			if err != nil {
				e := fmt.Errorf("failed to marshal patch: %v", err)
				k.logger.Errorf("%v", e)
				return k8sutil.ErrToAdmissionResponse(e)
			}
			collectedPatchesStr[i] = str
		}
		patchesStr := strings.Join(collectedPatchesStr, ",")
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
