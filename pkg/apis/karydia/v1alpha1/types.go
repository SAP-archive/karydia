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

package v1alpha1

import (
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type KarydiaConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KarydiaConfigSpec   `json:"spec"`
	Status KarydiaConfigStatus `json:"status"`
}

type KarydiaConfigSpec struct {
	// Enforcement can be used to enforce this default karydia configuration
	// and disable the "opt-out via annotation" functionality
	Enforcement bool `json:"enforcement"`

	// AutomountServiceAccountToken can be used to restrict auto-mounting
	// of service account tokens by default
	AutomountServiceAccountToken string `json:"automountServiceAccountToken"`

	// SeccompProfile can be used to set a default seccomp profile
	SeccompProfile string `json:"seccompProfile"`

	// NetworkPolicies can be used to set default network policies
	NetworkPolicies string `json:"networkPolicies"`

	// PodSecurityContext can be used to set a pod security context
	PodSecurityContext string `json:"podSecurityContext"`
}

type KarydiaConfigStatus struct {
	ServiceToken string `json:"serviceToken"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type KarydiaConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []KarydiaConfig `json:"items"`
}

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type KarydiaNetworkPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec networkingv1.NetworkPolicySpec `json:"spec"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type KarydiaNetworkPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []KarydiaNetworkPolicy `json:"items"`
}
