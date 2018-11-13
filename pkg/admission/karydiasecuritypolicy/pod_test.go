package karydiasecuritypolicy

import (
	"testing"

	corev1 "k8s.io/api/core/v1"

	"github.com/kinvolk/karydia/pkg/apis/karydia/v1alpha1"
)

func TestValidatePod(t *testing.T) {
	pod := &corev1.Pod{}

	patches, validationErrors := validatePod(&v1alpha1.KarydiaSecurityPolicy{}, pod)
	if len(validationErrors) != 0 {
		t.Errorf("expected 0 validation errors but got: %+v", validationErrors)
	}
	if len(patches) != 0 {
		t.Errorf("expected no patches but got: %+v", patches)
	}

	policy := &v1alpha1.KarydiaSecurityPolicy{
		Spec: v1alpha1.KarydiaSecurityPolicySpec{
			Pod: v1alpha1.Pod{
				AutomountServiceAccountToken: "forbidden",
				SeccompProfile:               "docker/default",
			},
		},
	}

	patches, validationErrors = validatePod(policy, pod)
	if len(validationErrors) != 1 {
		t.Errorf("expected 1 validation errors but got: %+v", validationErrors)
	}
	if len(patches) != 1 {
		t.Errorf("expected 1 patch but got: %+v", patches)
	}

	pod.ObjectMeta.Annotations = map[string]string{
		"seccomp.security.alpha.kubernetes.io/pod": "my-seccomp-profile",
	}

	patches, validationErrors = validatePod(policy, pod)
	if len(validationErrors) != 2 {
		t.Errorf("expected 2 validation errors but got: %+v", validationErrors)
	}
	if len(patches) != 0 {
		t.Errorf("expected no patches but got: %+v", patches)
	}
}
