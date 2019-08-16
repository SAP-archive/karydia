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

/* Mutating and Validating Webhook
 * Removes token mounts of the default service account when automountServiceToken is undefined.
 * kubectl annotate ns default karydia.gardener.cloud/seccompProfile=runtime/default
 */
func TestPodSeccompDefaultProfileWithAnnotation(t *testing.T) {
	pod := corev1.Pod{}
	var patches Patches
	var validationErrors []string

	pod.Annotations = make(map[string]string)
	pod.Annotations["seccomp.security.alpha.kubernetes.io/pod"] = "runtime/default"

	setting := Setting{value: "runtime/default", src: "namespace"}

	patches = mutatePodSeccompProfile(pod, setting, patches)
	if len(patches.operations) != 0 {
		t.Error("expected 0 patches but got:", patches.operations)
	}
	mutatedPod, err := patchPod(pod, patches)
	if err != nil {
		t.Error("failed to apply patches:", err)
	}
	// Zero validation errors expected for mutated pod
	validationErrors = validatePodSeccompProfile(mutatedPod, setting, validationErrors)
	if len(validationErrors) != 0 {
		t.Error("expected 0 validationErrors but got:", validationErrors)
	}
	validationErrors = []string{}
	// Zero validation error expected for initial pod
	validationErrors = validatePodSeccompProfile(pod, setting, validationErrors)
	if len(validationErrors) != 0 {
		t.Error("expected 0 validationErrors but got:", validationErrors)
	}
}

func TestPodSeccompDefaultProfileNoAnnotation(t *testing.T) {
	pod := corev1.Pod{}
	var patches Patches
	var validationErrors []string

	setting := Setting{value: "runtime/default", src: "namespace"}

	patches = mutatePodSeccompProfile(pod, setting, patches)
	if len(patches.operations) != 2 {
		t.Error("expected 2 patches but got:", patches.operations)
	}

	t.Log(patches)

	mutatedPod, err := patchPod(pod, patches)
	if err != nil {
		t.Error("failed to apply patches:", err)
	}

	t.Log(mutatedPod)
	// Zero validation errors expected for mutated pod
	validationErrors = validatePodSeccompProfile(mutatedPod, setting, validationErrors)
	if len(validationErrors) != 0 {
		t.Error("expected 0 validationErrors but got:", validationErrors)
	}
	validationErrors = []string{}
	// One validation error expected for initial pod
	validationErrors = validatePodSeccompProfile(pod, setting, validationErrors)
	if len(validationErrors) != 1 {
		t.Error("expected 1 validationErrors but got:", validationErrors)
	}
}

func TestPodSeccompDefaultProfileOtherAnnotation(t *testing.T) {
	pod := corev1.Pod{}
	var patches Patches
	var validationErrors []string

	setting := Setting{value: "runtime/default", src: "namespace"}

	pod.Annotations = make(map[string]string)
	pod.Annotations["seccomp.security.alpha.kubernetes.io/pod"] = "runtime/other"

	patches = mutatePodSeccompProfile(pod, setting, patches)
	if len(patches.operations) != 0 {
		t.Error("expected 0 patches but got:", patches.operations)
	}
	mutatedPod, err := patchPod(pod, patches)
	if err != nil {
		t.Error("failed to apply patches:", err)
	}
	// No validation error expected for mutated pod
	validationErrors = validatePodSeccompProfile(mutatedPod, setting, validationErrors)
	if len(validationErrors) != 0 {
		t.Error("expected 0 validationErrors but got:", validationErrors)
	}
	validationErrors = []string{}
	// No validation error expected for initial pod
	validationErrors = validatePodSeccompProfile(pod, setting, validationErrors)
	if len(validationErrors) != 0 {
		t.Error("expected 0 validationErrors but got:", validationErrors)
	}
}
