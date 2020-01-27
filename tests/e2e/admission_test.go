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
	"fmt"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestTerminatingPod(t *testing.T) {
	err := setAdmission(false)
	if err != nil {
		t.Fatal("Could not change admission in karydiaConfig", err)
	}

	// Create pod
	namespace, err := f.CreateTestNamespace()
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
					Name:    "alpine",
					Image:   "alpine",
					Command: []string{"tail", "-f", "/dev/null"},
				},
			},
		},
	}

	createdPod, err := f.KubeClientset.CoreV1().Pods(ns).Create(pod)
	if err != nil {
		t.Fatal("failed to create pod:", err)
	}

	timeout := 1 * time.Minute
	if err = f.WaitPodRunning(ns, pod.ObjectMeta.Name, timeout); err != nil {
		t.Fatal("pod never reached state running")
	}

	// Check pod
	if profile := createdPod.ObjectMeta.Annotations["seccomp.security.alpha.kubernetes.io/pod"]; profile != "unconfined" {
		t.Fatalf("expected seccomp profile to be %v but is %v", "unconfined", profile)
	}

	if createdPod.Spec.SecurityContext.RunAsUser != nil || createdPod.Spec.SecurityContext.RunAsGroup != nil {
		t.Fatal("expected security context not to be defined by admission")
	}

	err = setAdmission(true)
	if err != nil {
		t.Fatal("Could not change admission in karydiaConfig", err)
	}

	// Delete pod
	err = f.KubeClientset.CoreV1().Pods(ns).Delete(pod.ObjectMeta.Name, &metav1.DeleteOptions{})
	if err != nil {
		t.Fatal("could not delete pod")
	}

	timeout = 1 * time.Minute
	if err = f.WaitPodDeleted(ns, pod.ObjectMeta.Name, timeout); err != nil {
		t.Fatal("pod was never deleted")
	}
}

// helper functions

func setAdmission(admission bool) error {
	curKarydiaConfig, curErr := f.KarydiaClientset.KarydiaV1alpha1().KarydiaConfigs().Get("karydia-config", metav1.GetOptions{})

	if curErr != nil {
		return curErr
	}

	// This does not disable the admission but does not change the pod
	if !admission {
		curKarydiaConfig.Spec.SeccompProfile = "unconfined"
		curKarydiaConfig.Spec.PodSecurityContext = "none"
		curKarydiaConfig.Spec.AutomountServiceAccountToken = "no-change"
	} else {
		curKarydiaConfig.Spec.SeccompProfile = "runtime/default"
		curKarydiaConfig.Spec.PodSecurityContext = "nobody"
		curKarydiaConfig.Spec.AutomountServiceAccountToken = "change-default"
	}

	f.KarydiaClientset.KarydiaV1alpha1().KarydiaConfigs().Update(curKarydiaConfig)

	newKarydiaConfig, newErr := f.KarydiaClientset.KarydiaV1alpha1().KarydiaConfigs().Get("karydia-config", metav1.GetOptions{})

	if newErr != nil {
		return newErr
	}

	if !admission {
		if newKarydiaConfig.Spec.SeccompProfile != "unconfined" && newKarydiaConfig.Spec.PodSecurityContext != "none" && newKarydiaConfig.Spec.AutomountServiceAccountToken != "no-change" {
			return fmt.Errorf("admission in karydiaConfig did not change but should")
		}
	} else {
		if newKarydiaConfig.Spec.SeccompProfile != "change-default" && newKarydiaConfig.Spec.PodSecurityContext != "nobody" && newKarydiaConfig.Spec.AutomountServiceAccountToken != "runtime/default" {
			return fmt.Errorf("admission in karydiaConfig did not change but should")
		}
	}

	return nil
}
