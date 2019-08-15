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

package e2e

import (
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestSeccompWithNamespaceAnnotationUndefinedProfile(t *testing.T) {
	annotation := map[string]string{
		"karydia.gardener.cloud/seccompProfile":     "runtime/default",
		"karydia.gardener.cloud/podSecurityContext": "root",
	}
	namespace, err := f.CreateTestNamespaceWithAnnotation(annotation)
	if err != nil {
		t.Fatal("failed to create test namespace:", err)
	}

	ns := namespace.ObjectMeta.Name

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "karydia-e2e-test-pod",
			Namespace: ns,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "nginx",
					Image: "nginx",
				},
			},
		},
	}

	createdPod, err := f.KubeClientset.CoreV1().Pods(ns).Create(pod)
	if err != nil {
		t.Fatal("failed to create pod:", err)
	}

	if profile := createdPod.ObjectMeta.Annotations["seccomp.security.alpha.kubernetes.io/pod"]; profile != "runtime/default" {
		t.Fatalf("expected seccomp profile to be %v but is %v", "runtime/default", profile)
	}

	timeout := 2 * time.Minute
	if err := f.WaitPodRunning(pod.ObjectMeta.Namespace, pod.ObjectMeta.Name, timeout); err != nil {
		t.Fatal("pod never reached state running")
	}
}

func TestSeccompWithNamespaceAnnotationDefinedProfile(t *testing.T) {
	annotation := map[string]string{
		"karydia.gardener.cloud/seccompProfile":     "docker/specific",
		"karydia.gardener.cloud/podSecurityContext": "root",
	}
	namespace, err := f.CreateTestNamespaceWithAnnotation(annotation)
	if err != nil {
		t.Fatal("failed to create test namespace:", err)
	}

	ns := namespace.ObjectMeta.Name

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "karydia-e2e-test-pod",
			Namespace: ns,
			Annotations: map[string]string{
				"seccomp.security.alpha.kubernetes.io/pod": "runtime/default",
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "nginx",
					Image: "nginx",
				},
			},
		},
	}

	createdPod, err := f.KubeClientset.CoreV1().Pods(ns).Create(pod)
	if err != nil {
		t.Fatal("failed to create pod:", err)
	}

	if profile := createdPod.ObjectMeta.Annotations["seccomp.security.alpha.kubernetes.io/pod"]; profile != "runtime/default" {
		t.Fatalf("expected seccomp profile to be %v but is %v", "runtime/default", profile)
	}

	timeout := 2 * time.Minute
	if err := f.WaitPodRunning(pod.ObjectMeta.Namespace, pod.ObjectMeta.Name, timeout); err != nil {
		t.Fatal("pod never reached state running")
	}
}

func TestSeccompWithoutNamespaceAnnotationUndefinedProfileFromConfig(t *testing.T) {
	annotation := map[string]string{
		"karydia.gardener.cloud/podSecurityContext": "root",
	}
	namespace, err := f.CreateTestNamespaceWithAnnotation(annotation)
	if err != nil {
		t.Fatal("failed to create test namespace:", err)
	}

	ns := namespace.ObjectMeta.Name

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "karydia-e2e-test-pod",
			Namespace: ns,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "nginx",
					Image: "nginx",
				},
			},
		},
	}

	createdPod, err := f.KubeClientset.CoreV1().Pods(ns).Create(pod)
	if err != nil {
		t.Fatal("failed to create pod:", err)
	}

	if profile := createdPod.ObjectMeta.Annotations["seccomp.security.alpha.kubernetes.io/pod"]; profile != "runtime/default" {
		t.Fatalf("expected seccomp profile to be %v but is %v", "runtime/default", profile)
	}

	timeout := 2 * time.Minute
	if err := f.WaitPodRunning(pod.ObjectMeta.Namespace, pod.ObjectMeta.Name, timeout); err != nil {
		t.Fatal("pod never reached state running")
	}
}

func TestSeccompWithNamespaceAnnotationUndefinedProfileFromConfig(t *testing.T) {
	annotation := map[string]string{
		"karydia.gardener.cloud/seccompProfile":     "unconfined",
		"karydia.gardener.cloud/podSecurityContext": "root",
	}
	namespace, err := f.CreateTestNamespaceWithAnnotation(annotation)
	if err != nil {
		t.Fatal("failed to create test namespace:", err)
	}

	ns := namespace.ObjectMeta.Name

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "karydia-e2e-test-pod",
			Namespace: ns,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "nginx",
					Image: "nginx",
				},
			},
		},
	}

	createdPod, err := f.KubeClientset.CoreV1().Pods(ns).Create(pod)
	if err != nil {
		t.Fatal("failed to create pod:", err)
	}

	if profile := createdPod.ObjectMeta.Annotations["seccomp.security.alpha.kubernetes.io/pod"]; profile != "unconfined" {
		t.Fatalf("expected seccomp profile to be %v but is %v", "unconfined", profile)
	}

	timeout := 2 * time.Minute
	if err := f.WaitPodRunning(pod.ObjectMeta.Namespace, pod.ObjectMeta.Name, timeout); err != nil {
		t.Fatal("pod never reached state running")
	}
}

func TestSeccompWithoutNamespaceAnnotationDefinedProfile(t *testing.T) {
	annotation := map[string]string{
		"karydia.gardener.cloud/podSecurityContext": "root",
	}
	namespace, err := f.CreateTestNamespaceWithAnnotation(annotation)
	if err != nil {
		t.Fatal("failed to create test namespace:", err)
	}

	ns := namespace.ObjectMeta.Name

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "karydia-e2e-test-pod",
			Namespace: ns,
			Annotations: map[string]string{
				"seccomp.security.alpha.kubernetes.io/pod": "runtime/default",
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "nginx",
					Image: "nginx",
				},
			},
		},
	}

	createdPod, err := f.KubeClientset.CoreV1().Pods(ns).Create(pod)
	if err != nil {
		t.Fatal("failed to create pod:", err)
	}

	if profile := createdPod.ObjectMeta.Annotations["seccomp.security.alpha.kubernetes.io/pod"]; profile != "runtime/default" {
		t.Fatalf("expected seccomp profile to be %v but is %v", "runtime/default", profile)
	}

	timeout := 2 * time.Minute
	if err := f.WaitPodRunning(pod.ObjectMeta.Namespace, pod.ObjectMeta.Name, timeout); err != nil {
		t.Fatal("pod never reached state running")
	}
}
