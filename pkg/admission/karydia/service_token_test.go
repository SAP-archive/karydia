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

package karydia

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
)

/* Mutating and Validating Webhook
 * Removes token mounts of the default service account when automountServiceToken is undefined.
 * kubectl annotate ns default karydia.gardener.cloud/automountServiceAccountToken=remove-default
 */
func TestPodRemoveDefaultAnnotationDefaultServiceAccount(t *testing.T) {
	pod := corev1.Pod{}
	var patches []string
	var validationErrors []string

	pod.Spec.ServiceAccountName = "default"
	pod.Spec.Volumes = append([]corev1.Volume{}, corev1.Volume{Name: "default-token-abcd", VolumeSource: corev1.VolumeSource{}})
	mounts := append([]corev1.VolumeMount{}, corev1.VolumeMount{Name: "default-token-abcd"})
	pod.Spec.Containers = append([]corev1.Container{}, corev1.Container{Name: "first-container", VolumeMounts: mounts})

	patches = mutatePodServiceAccountToken(pod, "remove-default", patches)
	if len(patches) != 3 {
		t.Errorf("expected 3 patches but got: %+v", patches)
	}
	validationErrors = validatePodServiceAccountToken(pod, "remove-default", validationErrors)
	if len(validationErrors) != 1 {
		t.Errorf("expected 1 validationErrors but got: %+v", validationErrors)
	}
}

func TestPodRemoveDefaultAnnotationSpecificServiceAccount(t *testing.T) {
	pod := corev1.Pod{}
	var patches []string
	var validationErrors []string

	pod.Spec.ServiceAccountName = "test"
	pod.Spec.Volumes = append([]corev1.Volume{}, corev1.Volume{Name: "test-token-abcd", VolumeSource: corev1.VolumeSource{}})
	mounts := append([]corev1.VolumeMount{}, corev1.VolumeMount{Name: "test-token-abcd"})
	pod.Spec.Containers = append([]corev1.Container{}, corev1.Container{Name: "first-container", VolumeMounts: mounts})

	patches = mutatePodServiceAccountToken(pod, "remove-default", patches)
	if len(patches) != 0 {
		t.Errorf("expected 0 patches but got: %+v", patches)
	}
	validationErrors = validatePodServiceAccountToken(pod, "remove-default", validationErrors)
	if len(validationErrors) != 0 {
		t.Errorf("expected 0 validationErrors but got: %+v", validationErrors)
	}
}

func TestPodRemoveDefaultAnnotationDefaultSerivceAccountMultipleVolumes(t *testing.T) {
	pod := corev1.Pod{}
	var patches []string
	var validationErrors []string

	pod.Spec.ServiceAccountName = "default"
	pod.Spec.Volumes = append([]corev1.Volume{}, corev1.Volume{Name: "default-token-abcd", VolumeSource: corev1.VolumeSource{}})
	pod.Spec.Volumes = append(pod.Spec.Volumes, corev1.Volume{Name: "test-token-abcd", VolumeSource: corev1.VolumeSource{}})
	mounts := append([]corev1.VolumeMount{}, corev1.VolumeMount{Name: "default-token-abcd"})
	mounts = append(mounts, corev1.VolumeMount{Name: "test-token-abcd"})
	pod.Spec.Containers = append([]corev1.Container{}, corev1.Container{Name: "first-container", VolumeMounts: mounts})

	patches = mutatePodServiceAccountToken(pod, "remove-default", patches)
	if len(patches) != 3 {
		t.Errorf("expected 3 patches but got: %+v", patches)
	}
	validationErrors = validatePodServiceAccountToken(pod, "remove-default", validationErrors)
	if len(validationErrors) != 1 {
		t.Errorf("expected 1 validationErrors but got: %+v", validationErrors)
	}
}

/* Validating webhook that disallows token mounts of any service account
 * kubectl annotate ns default karydia.gardener.cloud/automountServiceAccountToken=forbidden
 */
func TestPodForbiddentAnnotationDefaultServiceAccount(t *testing.T) {
	pod := corev1.Pod{}
	var patches []string
	var validationErrors []string

	pod.Spec.ServiceAccountName = "default"

	patches = mutatePodServiceAccountToken(pod, "forbidden", patches)
	if len(patches) != 0 {
		t.Errorf("expected 0 patches but got: %+v", patches)
	}
	validationErrors = validatePodServiceAccountToken(pod, "forbidden", validationErrors)
	if len(validationErrors) != 1 {
		t.Errorf("expected 1 validationErrors but got: %+v", validationErrors)
	}
}

/* Validating webhook that disallows token mounts of the default service account
 * kubectl annotate ns default karydia.gardener.cloud/automountServiceAccountToken=non-default
 */
func TestPodNonDefaultAnnotationDefaultServiceAccount(t *testing.T) {
	pod := corev1.Pod{}
	var patches []string
	var validationErrors []string

	pod.Spec.ServiceAccountName = "default"
	pod.Spec.Volumes = append([]corev1.Volume{}, corev1.Volume{Name: "default-token-abcd", VolumeSource: corev1.VolumeSource{}})
	mounts := append([]corev1.VolumeMount{}, corev1.VolumeMount{Name: "default-token-abcd"})
	pod.Spec.Containers = append([]corev1.Container{}, corev1.Container{Name: "first-container", VolumeMounts: mounts})

	patches = mutatePodServiceAccountToken(pod, "non-default", patches)
	if len(patches) != 0 {
		t.Errorf("expected 0 patches but got: %+v", patches)
	}
	validationErrors = validatePodServiceAccountToken(pod, "non-default", validationErrors)
	if len(validationErrors) != 1 {
		t.Errorf("expected 1 validationErrors but got: %+v", validationErrors)
	}
}

func TestPodNonDefaultAnnotationSpecificServiceAccount(t *testing.T) {
	pod := corev1.Pod{}
	var patches []string
	var validationErrors []string

	pod.Spec.ServiceAccountName = "test"
	pod.Spec.Volumes = append([]corev1.Volume{}, corev1.Volume{Name: "test-token-abcd", VolumeSource: corev1.VolumeSource{}})
	mounts := append([]corev1.VolumeMount{}, corev1.VolumeMount{Name: "test-token-abcd"})
	pod.Spec.Containers = append([]corev1.Container{}, corev1.Container{Name: "first-container", VolumeMounts: mounts})
	patches = mutatePodServiceAccountToken(pod, "non-default", patches)
	if len(patches) != 0 {
		t.Errorf("expected 0 patches but got: %+v", patches)
	}
	validationErrors = validatePodServiceAccountToken(pod, "non-default", validationErrors)
	if len(validationErrors) != 0 {
		t.Errorf("expected 0 validationErrors but got: %+v", validationErrors)
	}
}
