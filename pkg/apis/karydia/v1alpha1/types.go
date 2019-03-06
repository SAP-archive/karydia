// Copyright 2019 Copyright (c) 2019 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file.
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
