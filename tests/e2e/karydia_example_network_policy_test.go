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

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	Success    int = 0
	NoResponse int = 1
	Error      int = 2
)

func commandWithErrorHandling(t *testing.T, command string, expectedExitCode int) {
	cmd := exec.Command(command)
	err := cmd.Run()
	exitCode := Error

	if err != nil {
		// try to get the exit code
		if exitError, ok := err.(*exec.ExitError); ok {

			// The program has exited with an exit code != 0
			exitCode = (exitError.Sys().(syscall.WaitStatus)).ExitStatus()
			if exitCode != expectedExitCode {
				t.Fatalf("Exit status with unexpected code: %d", exitCode)
			}
		}
	} else {

		// success, exitCode should be 0 if go is ok
		exitCode = (cmd.ProcessState.Sys().(syscall.WaitStatus)).ExitStatus()
		if exitCode != expectedExitCode {
			t.Fatalf("Unallowed command is allowed but it should not be: %d", exitCode)
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
			Name:      "karydia-e2e-test-pod",
			Namespace: ns,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "alpine",
					Image: "alpine",
				},
			},
		},
	}

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "nginx-service",
			Namespace: "kube-system",
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Protocol:   corev1.ProtocolTCP,
					Port:       80,
					TargetPort: intstr.FromInt(8080),
				},
			},
		},
	}

	createdPod, err := f.KubeClientset.CoreV1().Pods(ns).Create(pod)
	podName := createdPod.ObjectMeta.Name

	createdService, err := f.KubeClientset.CoreV1().Services(ns).Create(svc)
	serviceName := createdService.ObjectMeta.Name

	createdPod2, err := f.KubeClientset.CoreV1().Pods(ns).Create(pod)
	pod2IP := createdPod2.Status.PodIP

	pod.Namespace = "kube-system"
	createdPod3, err := f.KubeClientset.CoreV1().Pods(ns).Create(pod)
	pod3IP := createdPod3.Status.PodIP

	hostNetwork := createdPod.Status.HostIP

	// ----------------------------------------------------------------------------------- //

	cmd1 := "kubectl exec -it --namespace=" + ns + " " + podName + " -- ping -c 5 " + hostNetwork
	commandWithErrorHandling(t, cmd1, NoResponse)

	cmd2 := "kubectl exec -it --namespace=" + ns + " " + podName + " -- ping -c 5 169.254.169.254"
	commandWithErrorHandling(t, cmd2, NoResponse)

	cmd3 := "kubectl exec -it --namespace=" + ns + " " + podName + " -- ping -c 5 google.de"
	commandWithErrorHandling(t, cmd3, Success)

	cmd4 := "kubectl exec -it --namespace=" + ns + " " + podName + " -- nslookup " + serviceName + ".kube-system.svc.cluster.local"
	commandWithErrorHandling(t, cmd4, Success)

	cmd5 := "kubectl exec -it --namespace=" + ns + " " + podName + " -- ping -c 5 " + pod2IP
	commandWithErrorHandling(t, cmd5, Success)

	cmd6 := "kubectl exec -it --namespace=" + ns + " " + podName + " -- ping -c 5 " + pod3IP
	commandWithErrorHandling(t, cmd6, NoResponse)
}
