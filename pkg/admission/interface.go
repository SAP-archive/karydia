package admission

import (
	"k8s.io/api/admission/v1beta1"
)

type AdmissionPlugin interface {
	// Admit takes an admission review and a boolean flag if the
	// mutation of specs (i.e. patching) is allowed.
	Admit(v1beta1.AdmissionReview, bool) *v1beta1.AdmissionResponse
}
