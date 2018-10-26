package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type KarydiaSecurityPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KarydiaSecurityPolicySpec   `json:"spec"`
	Status KarydiaSecurityPolicyStatus `json:"status"`
}

type KarydiaSecurityPolicySpec struct {
	Pod Pod `json:"pod"`
}

type Pod struct {
	// AutomountServiceAccountToken can be used to restrict auto-mounting
	// of service account tokens. Can be 'forbidden' or 'non-default'.
	AutomountServiceAccountToken string `json:"automountServiceAccountToken"`

	// SeccompProfile can be used to enforce a seccomp profile annotation
	SeccompProfile string `json:"seccompProfile"`
}

type KarydiaSecurityPolicyStatus struct {
	ServiceToken string `json:"serviceToken"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type KarydiaSecurityPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []KarydiaSecurityPolicy `json:"items"`
}
