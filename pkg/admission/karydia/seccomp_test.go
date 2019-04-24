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

package karydia

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
)

//kubectl annotate ns default karydia.gardener.cloud/seccompProfile=runtime/default
func TestValidatePodWithExistingSeccompDefaultAnnotation(t *testing.T) {
	pod := corev1.Pod{}
	var patches []string
	var validationErrors []string

	pod.Annotations = make(map[string]string)
	pod.Annotations["seccomp.security.alpha.kubernetes.io/pod"] = "runtime/default"

	patches, validationErrors = admitSeccompProfile(pod, "runtime/default", false, patches, validationErrors)
	if len(patches) != 0 {
		t.Errorf("expected 0 patches but got: %+v", patches)
	}
	if len(validationErrors) != 0 {
		t.Errorf("expected 0 validationErrors but got: %+v", validationErrors)
	}
}

func TestMutatePodWithSeccompDefaultAnnotation(t *testing.T) {
	pod := corev1.Pod{}
	var patches []string
	var validationErrors []string

	patches, validationErrors = admitSeccompProfile(pod, "runtime/default", true, patches, validationErrors)
	if len(patches) != 1 {
		t.Errorf("expected 1 patches but got: %+v", patches)
	}
	if len(validationErrors) != 0 {
		t.Errorf("expected 0 validationErrors but got: %+v", validationErrors)
	}
}

func TestValidatePodWithSeccompDefaultAnnotation(t *testing.T) {
	pod := corev1.Pod{}
	var patches []string
	var validationErrors []string

	patches, validationErrors = admitSeccompProfile(pod, "runtime/default", false, patches, validationErrors)
	if len(patches) != 0 {
		t.Errorf("expected 0 patches but got: %+v", patches)
	}
	if len(validationErrors) != 1 {
		t.Errorf("expected 1 validationErrors but got: %+v", validationErrors)
	}
}
