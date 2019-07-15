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
	"os/exec"
	"syscall"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	Success int = 0
	TimeOut int = 1
)

func execCommandAssertExitCode(t *testing.T, command string, expectedExitCode int) {
	exitCode := Success

	cmd := exec.Command("/bin/sh", "-c", command)
	err := cmd.Run()

	if err != nil {
		// try to get the exit code
		if exitError, ok := err.(*exec.ExitError); ok {
			ws := exitError.Sys().(syscall.WaitStatus)
			exitCode = ws.ExitStatus()

			if exitCode != expectedExitCode {
				t.Fatalf("Exit status with unexpected code: %d %s", exitCode, command)
			}
		} else {
			t.Fatalf("Could not get exit code.")
		}
	} else {
		// success, exitCode should be 0 if go is ok
		ws := cmd.ProcessState.Sys().(syscall.WaitStatus)
		exitCode = ws.ExitStatus()

		if exitCode != expectedExitCode {
			t.Fatalf("Exit status with unexpected code: %d %s", exitCode, command)
		}
	}
}

func TestNetworkPolicyLevel1(t *testing.T) {

	var namespace *corev1.Namespace
	var err error

	namespace, err = f.CreateTestNamespace()
	if err != nil {
		t.Fatalf("failed to create test namespace: %v", err)
	}

	ns := namespace.ObjectMeta.Name

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "karydia-network-policy-1",
			Namespace: ns,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "alpine",
					Image: "alpine",
					Args: []string{
						"/bin/sh",
						"-c",
						"sleep 60m",
					},
				},
			},
		},
	}

	createdPod, err := f.KubeClientset.CoreV1().Pods(ns).Create(pod)
	if err != nil {
		t.Fatalf("Failed to create: " + err.Error())
	}
	podName := createdPod.ObjectMeta.Name

	timeout := 2 * time.Minute
	if err := f.WaitPodRunning(ns, podName, timeout); err != nil {
		t.Fatalf("pod never reached state running")
	}

	// Test Host network
	cmd1 := "kubectl exec -it --namespace=" + ns + " " + podName + " -- wget --spider --timeout 3 10.250.0.0"
	execCommandAssertExitCode(t, cmd1, TimeOut)

	cmd2 := "kubectl exec -it --namespace=" + ns + " " + podName + " -- wget --spider --timeout 3 10.250.1.1"
	execCommandAssertExitCode(t, cmd2, TimeOut)

	cmd3 := "kubectl exec -it --namespace=" + ns + " " + podName + " -- wget --spider --timeout 3 10.250.50.200"
	execCommandAssertExitCode(t, cmd3, TimeOut)

	// Meta Data Servicess
	cmd4 := "kubectl exec -it --namespace=" + ns + " " + podName + " -- wget --spider --timeout 3 169.254.169.254"
	execCommandAssertExitCode(t, cmd4, TimeOut)

	cmd5 := "kubectl exec -it --namespace=" + ns + " " + podName + " -- wget --spider --timeout 3 169.254.0.0"
	execCommandAssertExitCode(t, cmd5, TimeOut)

	cmd6 := "kubectl exec -it --namespace=" + ns + " " + podName + " -- wget --spider --timeout 3 169.254.2.2"
	execCommandAssertExitCode(t, cmd6, TimeOut)

	// Alibaba Clouds
	cmd7 := "kubectl exec -it --namespace=" + ns + " " + podName + " -- wget --spider --timeout 3 100.100.0.0"
	execCommandAssertExitCode(t, cmd7, TimeOut)

	cmd8 := "kubectl exec -it --namespace=" + ns + " " + podName + " -- wget --spider --timeout 3 100.100.1.3"
	execCommandAssertExitCode(t, cmd8, TimeOut)

	cmd9 := "kubectl exec -it --namespace=" + ns + " " + podName + " -- wget --spider --timeout 3 100.100.60.155"
	execCommandAssertExitCode(t, cmd9, TimeOut)

	// External traffic with static IPs
	// Google website
	cmd10 := "kubectl exec -it --namespace=" + ns + " " + podName + " -- wget --spider --timeout 3 172.217.21.227"
	execCommandAssertExitCode(t, cmd10, Success)

	// External traffic with domain names
	cmd13 := "kubectl exec -it --namespace=" + ns + " " + podName + " -- wget --spider --timeout 3 www.google.de"
	execCommandAssertExitCode(t, cmd13, Success)

	cmd14 := "kubectl exec -it --namespace=" + ns + " " + podName + " -- wget --spider --timeout 3 www.spiegel.de"
	execCommandAssertExitCode(t, cmd14, Success)

	cmd15 := "kubectl exec -it --namespace=" + ns + " " + podName + " -- wget --spider --timeout 3 www.sap.com"
	execCommandAssertExitCode(t, cmd15, Success)

	err = f.KubeClientset.CoreV1().Pods(ns).Delete(createdPod.ObjectMeta.Name, &metav1.DeleteOptions{})
	if err != nil {
		t.Fatalf("pod could not be deleted")
	}

}
