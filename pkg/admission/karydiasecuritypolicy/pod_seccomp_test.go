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

package karydiasecuritypolicy

import (
	"testing"

	corev1 "k8s.io/api/core/v1"

	"github.com/karydia/karydia/pkg/apis/karydia/v1alpha1"
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
				SeccompProfile: "docker/default",
			},
		},
	}

	patches, validationErrors = validatePod(policy, pod)
	if len(validationErrors) != 0 {
		t.Errorf("expected 0 validation errors but got: %+v", validationErrors)
	}
	if len(patches) != 1 {
		t.Errorf("expected 1 patch but got: %+v", patches)
	}

	pod.ObjectMeta.Annotations = map[string]string{
		"seccomp.security.alpha.kubernetes.io/pod": "my-seccomp-profile",
	}

	patches, validationErrors = validatePod(policy, pod)
	if len(validationErrors) != 1 {
		t.Errorf("expected 2 validation errors but got: %+v", validationErrors)
	}
	if len(patches) != 0 {
		t.Errorf("expected no patches but got: %+v", patches)
	}
}
