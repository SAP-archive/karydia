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

// kubectl annotate ns default karydia.gardener.cloud/automountServiceAccountToken=remove-default
func TestMutatePodWithRemoveDefaultAnnotation(t *testing.T) {
	pod := corev1.Pod{}
	var patches []string
	var validationErrors []string

	pod.Spec.ServiceAccountName = "default"
	pod.Spec.Volumes = append([]corev1.Volume{}, corev1.Volume{Name: "default-token-abcd", VolumeSource: corev1.VolumeSource{}})
	mounts := append([]corev1.VolumeMount{}, corev1.VolumeMount{Name: "default-token-abcd"})
	pod.Spec.Containers = append([]corev1.Container{}, corev1.Container{Name: "first-container", VolumeMounts: mounts})

	patches, validationErrors = secureAutomountServiceAccountToken(pod, "remove-default", patches, validationErrors)
	if len(patches) != 4 {
		t.Errorf("expected 4 patches but got: %+v", patches)
	}
	if len(validationErrors) != 0 {
		t.Errorf("expected 0 validationErrors but got: %+v", validationErrors)
	}
}

func TestMutatePodWithRemoveDefaultAnnotationNonDefaultServiceAccount(t *testing.T) {
	pod := corev1.Pod{}
	var patches []string
	var validationErrors []string

	pod.Spec.ServiceAccountName = "test"
	pod.Spec.Volumes = append([]corev1.Volume{}, corev1.Volume{Name: "test-token-abcd", VolumeSource: corev1.VolumeSource{}})
	mounts := append([]corev1.VolumeMount{}, corev1.VolumeMount{Name: "test-token-abcd"})
	pod.Spec.Containers = append([]corev1.Container{}, corev1.Container{Name: "first-container", VolumeMounts: mounts})

	patches, validationErrors = secureAutomountServiceAccountToken(pod, "remove-default", patches, validationErrors)
	if len(patches) != 0 {
		t.Errorf("expected 0 patches but got: %+v", patches)
	}
	if len(validationErrors) != 0 {
		t.Errorf("expected 0 validationErrors but got: %+v", validationErrors)
	}
}

func TestMutatePodWithRemoveDefaultAnnotationMultipleVolumes(t *testing.T) {
	pod := corev1.Pod{}
	var patches []string
	var validationErrors []string

	pod.Spec.ServiceAccountName = "default"
	pod.Spec.Volumes = append([]corev1.Volume{}, corev1.Volume{Name: "default-token-abcd", VolumeSource: corev1.VolumeSource{}})
	pod.Spec.Volumes = append(pod.Spec.Volumes, corev1.Volume{Name: "test-token-abcd", VolumeSource: corev1.VolumeSource{}})
	mounts := append([]corev1.VolumeMount{}, corev1.VolumeMount{Name: "default-token-abcd"})
	mounts = append(mounts, corev1.VolumeMount{Name: "test-token-abcd"})
	pod.Spec.Containers = append([]corev1.Container{}, corev1.Container{Name: "first-container", VolumeMounts: mounts})

	patches, validationErrors = secureAutomountServiceAccountToken(pod, "remove-default", patches, validationErrors)
	if len(patches) != 4 {
		t.Errorf("expected 4 patches but got: %+v", patches)
	}
	if len(validationErrors) != 0 {
		t.Errorf("expected 0 validationErrors but got: %+v", validationErrors)
	}
}

//kubectl annotate ns default karydia.gardener.cloud/automountServiceAccountToken=forbidden
func TestValidatePodWithForbiddentAnnotation(t *testing.T) {
	pod := corev1.Pod{}
	var patches []string
	var validationErrors []string

	pod.Spec.ServiceAccountName = "default"

	patches, validationErrors = secureAutomountServiceAccountToken(pod, "forbidden", patches, validationErrors)
	if len(patches) != 0 {
		t.Errorf("expected 0 patches but got: %+v", patches)
	}
	if len(validationErrors) != 1 {
		t.Errorf("expected 1 validationErrors but got: %+v", validationErrors)
	}
}

//kubectl annotate ns default karydia.gardener.cloud/automountServiceAccountToken=non-default
func TestValidatePodWithNonDefaultAnnotation(t *testing.T) {
	pod := corev1.Pod{}
	var patches []string
	var validationErrors []string

	pod.Spec.ServiceAccountName = "default"
	patches, validationErrors = secureAutomountServiceAccountToken(pod, "non-default", patches, validationErrors)
	if len(patches) != 0 {
		t.Errorf("expected 0 patches but got: %+v", patches)
	}
	if len(validationErrors) != 1 {
		t.Errorf("expected 1 validationErrors but got: %+v", validationErrors)
	}
}

func TestValidatePodWithNonDefaultAnnotationNonDefaultServiceAccount(t *testing.T) {
	pod := corev1.Pod{}
	var patches []string
	var validationErrors []string

	pod.Spec.ServiceAccountName = "test"
	patches, validationErrors = secureAutomountServiceAccountToken(pod, "non-default", patches, validationErrors)
	if len(patches) != 0 {
		t.Errorf("expected 0 patches but got: %+v", patches)
	}
	if len(validationErrors) != 0 {
		t.Errorf("expected 0 validationErrors but got: %+v", validationErrors)
	}
}
