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
	"strconv"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var changeDefault = "change-default"
var changeAll = "change-all"
var vTrue = true
var vFalse = false

type AutomountTokenTestCase struct {
	serviceAccount               string
	nsAnnotation                 *string
	automountTokenServiceAccount *bool
	automountTokenPod            *bool
	mounted                      bool
}

func TestAutomountServiceAccountToken(t *testing.T) {
	testCases := []AutomountTokenTestCase{
		{"default", nil, nil, nil, false},
		{"default", &changeDefault, nil, nil, false},
		{"default", &changeAll, nil, nil, false},

		{"dedicated", nil, nil, nil, true},
		{"dedicated", &changeDefault, nil, nil, true},
		{"dedicated", &changeAll, nil, nil, false},
	}

	for _, tc := range testCases {
		t.Run(stringifyTestCase(tc), func(t *testing.T) {

			var namespace *corev1.Namespace
			var err error

			if tc.nsAnnotation == nil {
				namespace, err = f.CreateTestNamespace()
				if err != nil {
					t.Fatalf("failed to create test namespace: %v", err)
				}
			} else {
				/* Directly annotate namespace to prevent race-conditions with the
				   automatically created default service account */
				annotation := map[string]string{
					"karydia.gardener.cloud/automountServiceAccountToken": *tc.nsAnnotation,
					"karydia.gardener.cloud/podSecurityContext":           "root",
				}
				namespace, err = f.CreateTestNamespaceWithAnnotation(annotation)
				if err != nil {
					t.Fatalf("failed to create test namespace: %v", err)
				}
			}

			ns := namespace.ObjectMeta.Name

			if tc.serviceAccount != "" && tc.serviceAccount != "default" {
				sAcc, err := f.CreateServiceAccount(tc.serviceAccount, ns)
				if err != nil {
					t.Fatalf("failed to create service account: %v", err)
				}
				if tc.nsAnnotation != nil && *tc.nsAnnotation == "change-default" {
					if sAcc.AutomountServiceAccountToken != nil {
						t.Fatalf("expected service account automount to be undefined")
					}
				} else if tc.nsAnnotation != nil && *tc.nsAnnotation == "change-all" {
					if sAcc.AutomountServiceAccountToken != nil && *sAcc.AutomountServiceAccountToken != false {
						t.Fatalf("expected service account automount to be false")
					}
				}
			} else {
				timeout := 3000 * time.Millisecond
				if err := f.WaitDefaultServiceAccountCreated(ns, timeout); err != nil {
					t.Fatalf("default service account not created: %v", err)
				}
			}

			/* Try twice due to infrequent lock of service account resource after creation */
			err = updateServiceAccount(tc, ns)
			if err != nil {
				err = updateServiceAccount(tc, ns)
				if err != nil {
					t.Fatalf("could not update service account: %v", err)
				}
			}

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

			pod.Spec.ServiceAccountName = tc.serviceAccount
			if tc.automountTokenPod != nil {
				pod.Spec.AutomountServiceAccountToken = tc.automountTokenPod
			}

			createdPod, err := f.KubeClientset.CoreV1().Pods(ns).Create(pod)
			if err != nil {
				t.Fatalf("failed to create pod: %v", err)
			}

			if createdPod.Spec.AutomountServiceAccountToken != nil && *createdPod.Spec.AutomountServiceAccountToken != tc.mounted {
				t.Fatalf("expected automountServiceAccountToken to be %v but is %v", tc.mounted, *createdPod.Spec.AutomountServiceAccountToken)
			}

			if (len(createdPod.Spec.Volumes) == 1) != tc.mounted {
				t.Fatalf("expected is mounted to be %v but is %v with %v", tc.mounted, len(createdPod.Spec.Volumes) == 1, createdPod.Spec.Volumes)
			}

			timeout := 2 * time.Minute
			if err := f.WaitPodRunning(pod.ObjectMeta.Namespace, pod.ObjectMeta.Name, timeout); err != nil {
				t.Fatalf("pod never reached state running")
			}
		})
	}
}

/*
Single test case for default ServiceAccount in the default Namespace

This represents a special case, since this service account
is already present when karydia is installed configuration via
web-hook won't work.
*/
func TestAutomountServiceAccountTokenInDefaultNamespace(t *testing.T) {
	var ns = "default"
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
		t.Fatalf("failed to create pod: %v", err)
	}

	if createdPod.Spec.AutomountServiceAccountToken != nil {
		t.Fatalf("expected automountServiceAccountToken to be nil but is %v", createdPod.Spec.AutomountServiceAccountToken)
	}

	if !(len(createdPod.Spec.Volumes) == 0) {
		/* Change to t.Fatalf when karydia setup is finished */
		t.Logf("expected is mounted to be false but is true")
	}

	timeout := 2 * time.Minute
	if err := f.WaitPodRunning(pod.ObjectMeta.Namespace, pod.ObjectMeta.Name, timeout); err != nil {
		t.Fatalf("pod never reached state running")
	}
}

func TestAutomountServiceAccountTokenEditServiceAccount(t *testing.T) {
	var namespace *corev1.Namespace
	var err error

	annotation := map[string]string{
		"karydia.gardener.cloud/automountServiceAccountToken": "change-all",
		"karydia.gardener.cloud/podSecurityContext":           "root",
	}
	namespace, err = f.CreateTestNamespaceWithAnnotation(annotation)
	if err != nil {
		t.Fatalf("failed to create test namespace: %v", err)
	}

	ns := namespace.ObjectMeta.Name

	sAcc, err := f.CreateServiceAccount("dedicated", ns)
	if err != nil {
		t.Fatalf("failed to create service account: %v", err)
	}

	automount := true
	sAcc.AutomountServiceAccountToken = &automount
	sAcc, err = f.KubeClientset.CoreV1().ServiceAccounts(ns).Update(sAcc)
	if err != nil {
		t.Fatalf("failed to update service account: %v", err)
	}

	/* Test update resource ServiceAccount */
	sAcc.AutomountServiceAccountToken = nil
	sAcc, err = f.KubeClientset.CoreV1().ServiceAccounts(ns).Update(sAcc)
	if err != nil {
		t.Fatalf("failed to update service account: %v", err)
	}

	if sAcc.AutomountServiceAccountToken == nil {
		t.Fatalf("expected updated service account to have automoutnServiceAccountToken set to false but is nil")
	}
}

func TestAutomountServiceAccountTokenDefaultServiceAccountFromConfig(t *testing.T) {
	var namespace *corev1.Namespace
	var err error

	annotation := map[string]string{
		"karydia.gardener.cloud/podSecurityContext": "root",
	}
	namespace, err = f.CreateTestNamespaceWithAnnotation(annotation)
	if err != nil {
		t.Fatalf("failed to create test namespace: %v", err)
	}

	ns := namespace.ObjectMeta.Name
	timeout := 3000 * time.Millisecond
	if err := f.WaitDefaultServiceAccountCreated(ns, timeout); err != nil {
		t.Fatalf("default service account not created: %v", err)
	}
	sAcc, err := f.KubeClientset.CoreV1().ServiceAccounts(ns).Get("default", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("failed to get service account: %v", err)
	}

	if *sAcc.AutomountServiceAccountToken != false {
		t.Fatalf("expected service account to have automountServiceAccountToken set to false but is %v",
			sAcc.AutomountServiceAccountToken)
	}

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
		t.Fatalf("failed to create pod: %v", err)
	}

	if createdPod.Spec.AutomountServiceAccountToken != nil {
		t.Fatalf("expected automountServiceAccountToken to be nil but is %v", createdPod.Spec.AutomountServiceAccountToken)
	}

	if !(len(createdPod.Spec.Volumes) == 0) {
		t.Fatalf("expected is mounted to be false but is true")
	}

	timeout = 2 * time.Minute
	if err := f.WaitPodRunning(pod.ObjectMeta.Namespace, pod.ObjectMeta.Name, timeout); err != nil {
		t.Fatalf("pod never reached state running")
	}
}

func TestAutomountServiceAccountTokenDedicatedServiceAccountFromConfig(t *testing.T) {
	var namespace *corev1.Namespace
	var err error

	annotation := map[string]string{
		"karydia.gardener.cloud/podSecurityContext": "root",
	}
	namespace, err = f.CreateTestNamespaceWithAnnotation(annotation)
	if err != nil {
		t.Fatalf("failed to create test namespace: %v", err)
	}

	ns := namespace.ObjectMeta.Name
	timeout := 3000 * time.Millisecond
	sAcc, err := f.CreateServiceAccount("dedicated", ns)
	if err != nil {
		t.Fatalf("failed to create service account: %v", err)
	}
	sAcc, err = f.KubeClientset.CoreV1().ServiceAccounts(ns).Get("dedicated", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("failed to get service account: %v", err)
	}

	if sAcc.AutomountServiceAccountToken != nil {
		t.Fatalf("expected service account to have automountServiceAccountToken set to nil but is %v",
			sAcc.AutomountServiceAccountToken)
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "karydia-e2e-test-pod",
			Namespace: ns,
		},
		Spec: corev1.PodSpec{
			ServiceAccountName: sAcc.Name,
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
		t.Fatalf("failed to create pod: %v", err)
	}

	if createdPod.Spec.AutomountServiceAccountToken != nil {
		t.Fatalf("expected automountServiceAccountToken to be nil but is %v", createdPod.Spec.AutomountServiceAccountToken)
	}

	if !(len(createdPod.Spec.Volumes) == 1) {
		t.Fatalf("expected is mounted to be true but is false")
	}

	timeout = 2 * time.Minute
	if err := f.WaitPodRunning(pod.ObjectMeta.Namespace, pod.ObjectMeta.Name, timeout); err != nil {
		t.Fatalf("pod never reached state running")
	}
}

func updateServiceAccount(tc AutomountTokenTestCase, ns string) error {
	if tc.automountTokenServiceAccount != nil {
		sAcc, err := f.KubeClientset.CoreV1().ServiceAccounts(ns).Get(tc.serviceAccount, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("failed to retrieve service account: %v", err)
		}
		sAcc.AutomountServiceAccountToken = tc.automountTokenServiceAccount
		sAcc, err = f.KubeClientset.CoreV1().ServiceAccounts(ns).Update(sAcc)
		if err != nil {
			return fmt.Errorf("failed to update service account: %v", err)
		}
	}
	return nil
}

func stringifyTestCase(tc AutomountTokenTestCase) string {
	var serviceAccount = tc.serviceAccount
	var annotation string
	var automountTokenPod string
	var automountTokenServiceAccount string

	if tc.nsAnnotation != nil {
		annotation = *tc.nsAnnotation
	} else {
		annotation = "Undefined"
	}

	if tc.automountTokenServiceAccount != nil {
		automountTokenServiceAccount = strconv.FormatBool(*tc.automountTokenServiceAccount)
	} else {
		automountTokenServiceAccount = "Undefined"
	}

	if tc.automountTokenPod != nil {
		automountTokenPod = strconv.FormatBool(*tc.automountTokenPod)
	} else {
		automountTokenPod = "Undefined"
	}

	return fmt.Sprintf("%vServiceAccount%vAnnotation%vServiceAccountAutomount%vPodAutomount",
		serviceAccount, annotation, automountTokenServiceAccount, automountTokenPod)
}
