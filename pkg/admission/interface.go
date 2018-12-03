package admission

import (
	"k8s.io/api/admission/v1beta1"
)

type AdmissionPlugin interface {
	Admit(v1beta1.AdmissionReview) *v1beta1.AdmissionResponse
}
