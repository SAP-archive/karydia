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

func TestSecurityContextWithNamespaceAnnotationUndefinedContext(t *testing.T) {
	annotation := map[string]string{
		"karydia.gardener.cloud/podSecurityContext": "nobody",
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
					Name:  "redis",
					Image: "redis",
				},
			},
		},
	}

	timeout := 3000 * time.Millisecond
	if err := f.WaitDefaultServiceAccountCreated(ns, timeout); err != nil {
		t.Fatal("default service account not created:", err)
	}

	createdPod, err := f.KubeClientset.CoreV1().Pods(ns).Create(pod)
	if err != nil {
		t.Fatal("failed to create pod:", err)
	}

	secCtx := createdPod.Spec.SecurityContext
	if secCtx == nil {
		t.Fatal("expected security context to be defined by admission but is nil")
	} else if *secCtx.RunAsUser != 65534 {
		t.Fatalf("expected security context user id to be %v but is %v", 65534, *secCtx.RunAsUser)
	} else if *secCtx.RunAsGroup != 65534 {
		t.Fatalf("expected security context group id to be %v but is %v", 65534, *secCtx.RunAsGroup)
	}

	containers := createdPod.Spec.Containers
	for i := range containers {
		secCtxContainers := createdPod.Spec.Containers[i].SecurityContext
		if secCtxContainers == nil {
			t.Fatal("expected security context to be definded by admission but is nil")
		} else if *secCtxContainers.AllowPrivilegeEscalation != false {
			t.Fatalf("expected container security context allow priviledge escalation to be %v but is %v", false, *secCtxContainers.AllowPrivilegeEscalation)
		}
	}

	timeout = 2 * time.Minute
	if err := f.WaitPodRunning(pod.ObjectMeta.Namespace, pod.ObjectMeta.Name, timeout); err != nil {
		t.Fatal("pod never reached state running")
	}
}

func TestSecurityContextWithNamespaceAnnotationDefinedContext(t *testing.T) {
	annotation := map[string]string{
		"karydia.gardener.cloud/podSecurityContext": "nobody",
	}
	namespace, err := f.CreateTestNamespaceWithAnnotation(annotation)
	if err != nil {
		t.Fatal("failed to create test namespace:", err)
	}

	ns := namespace.ObjectMeta.Name

	var uid int64 = 1000
	var allowPrivilegeEscalation = true

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "karydia-e2e-test-pod",
			Namespace: ns,
		},
		Spec: corev1.PodSpec{
			SecurityContext: &corev1.PodSecurityContext{
				RunAsUser: &uid,
			},
			Containers: []corev1.Container{
				{
					Name:  "redis",
					Image: "redis",
					SecurityContext: &corev1.SecurityContext{
						AllowPrivilegeEscalation: &allowPrivilegeEscalation,
					},
				},
			},
		},
	}

	createdPod, err := f.KubeClientset.CoreV1().Pods(ns).Create(pod)
	if err != nil {
		t.Fatal("failed to create pod:", err)
	}

	secCtx := createdPod.Spec.SecurityContext
	if secCtx == nil {
		t.Fatal("expected security context to be defined by pod definition but is nil")
	} else if *secCtx.RunAsUser != 1000 {
		t.Fatalf("expected security context user id to be %v but is %v", 1000, *secCtx.RunAsUser)
	} else if secCtx.RunAsGroup != nil {
		t.Fatalf("expected security context group id to be %v but is %v", "nil", *secCtx.RunAsGroup)
	}

	containers := createdPod.Spec.Containers
	for i := range containers {
		secCtxContainers := createdPod.Spec.Containers[i].SecurityContext
		if secCtxContainers == nil {
			t.Fatal("expected security context to be definded by pod definition but is nil")
		} else if *secCtxContainers.AllowPrivilegeEscalation != true {
			t.Fatalf("expected container security context allow priviledge escalation to be %v but is %v", true, *secCtxContainers.AllowPrivilegeEscalation)
		}
	}

	timeout := 2 * time.Minute
	if err := f.WaitPodRunning(pod.ObjectMeta.Namespace, pod.ObjectMeta.Name, timeout); err != nil {
		t.Fatal("pod never reached state running")
	}
}

func TestSecurityContextWithoutNamespaceAnnotationUndefinedContextFromConfig(t *testing.T) {
	annotation := map[string]string{}
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
					Name:  "redis",
					Image: "redis",
				},
			},
		},
	}

	createdPod, err := f.KubeClientset.CoreV1().Pods(ns).Create(pod)
	if err != nil {
		t.Fatal("failed to create pod:", err)
	}

	secCtx := createdPod.Spec.SecurityContext
	if secCtx == nil {
		t.Fatal("expected security context to be defined by admission but is nil")
	} else if *secCtx.RunAsUser != 65534 {
		t.Fatalf("expected security context user id to be %v but is %v", 65534, *secCtx.RunAsUser)
	} else if *secCtx.RunAsGroup != 65534 {
		t.Fatalf("expected security context group id to be %v but is %v", 65534, *secCtx.RunAsGroup)
	}

	containers := createdPod.Spec.Containers
	for i := range containers {
		secCtxContainers := createdPod.Spec.Containers[i].SecurityContext
		if secCtxContainers == nil {
			t.Fatal("expected security context to be defined by admission but is nil")
		} else if *secCtxContainers.AllowPrivilegeEscalation != false {
			t.Fatalf("expected container security context allow priviledge escalation to be %v but is %v", false, *secCtxContainers.AllowPrivilegeEscalation)
		}
	}

	timeout := 2 * time.Minute
	if err := f.WaitPodRunning(pod.ObjectMeta.Namespace, pod.ObjectMeta.Name, timeout); err != nil {
		t.Fatal("pod never reached state running")
	}
}
