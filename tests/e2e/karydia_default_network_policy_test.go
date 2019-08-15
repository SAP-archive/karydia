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
	"os/exec"
	"strings"
	"syscall"
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	Success int = 0
	TimeOut int = 1
)

func execCommandAssertExitCode(t *testing.T, command string, expectedExitCode int) {
	exitCode := Success

	args := strings.Fields(command)
	cmd := exec.Command(args[0], args[1:]...)
	err := cmd.Run()

	if err != nil {
		// try to get the exit code
		if exitError, ok := err.(*exec.ExitError); ok {
			ws := exitError.Sys().(syscall.WaitStatus)
			exitCode = ws.ExitStatus()

			if exitCode != expectedExitCode {
				t.Fatal("Exit status with unexpected code:", exitCode, command)
			}
		} else {
			t.Fatal("Could not get exit code.")
		}
	} else {
		// success, exitCode should be 0 if go is ok
		ws := cmd.ProcessState.Sys().(syscall.WaitStatus)
		exitCode = ws.ExitStatus()

		if exitCode != expectedExitCode {
			t.Fatal("Exit status with unexpected code:", exitCode, command)
		}
	}
}

func getPodDescription(ns string, podName string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: ns,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:    "karydia-e2e-test-container",
					Image:   "alpine",
					Command: []string{"tail", "-f", "/dev/null"},
				},
			},
		},
	}
}

func networkPolicyTestCases(t *testing.T, podName string, ns string, level int) {

	// Host Network (AWS only)
	cmd1 := "kubectl exec --namespace=" + ns + " " + podName + " -- wget --spider --timeout 3 10.250.0.0"
	execCommandAssertExitCode(t, cmd1, TimeOut)

	cmd2 := "kubectl exec --namespace=" + ns + " " + podName + " -- wget --spider --timeout 3 10.250.1.1"
	execCommandAssertExitCode(t, cmd2, TimeOut)

	cmd3 := "kubectl exec --namespace=" + ns + " " + podName + " -- wget --spider --timeout 3 10.250.50.200"
	execCommandAssertExitCode(t, cmd3, TimeOut)

	// Meta Data Services (AWS, GCP, Azure)
	cmd4 := "kubectl exec --namespace=" + ns + " " + podName + " -- wget --spider --timeout 3 169.254.169.254"
	execCommandAssertExitCode(t, cmd4, TimeOut)

	cmd5 := "kubectl exec --namespace=" + ns + " " + podName + " -- wget --spider --timeout 3 169.254.0.0"
	execCommandAssertExitCode(t, cmd5, TimeOut)

	cmd6 := "kubectl exec --namespace=" + ns + " " + podName + " -- wget --spider --timeout 3 169.254.2.2"
	execCommandAssertExitCode(t, cmd6, TimeOut)

	// Meta Data Services (Alibaba Cloud)
	cmd7 := "kubectl exec --namespace=" + ns + " " + podName + " -- wget --spider --timeout 3 100.100.0.0"
	execCommandAssertExitCode(t, cmd7, TimeOut)

	cmd8 := "kubectl exec --namespace=" + ns + " " + podName + " -- wget --spider --timeout 3 100.100.1.3"
	execCommandAssertExitCode(t, cmd8, TimeOut)

	cmd9 := "kubectl exec --namespace=" + ns + " " + podName + " -- wget --spider --timeout 3 100.100.60.155"
	execCommandAssertExitCode(t, cmd9, TimeOut)

	expectedExitCode := Success
	if level == 2 || level == 3 {
		expectedExitCode = TimeOut
	}

	// External traffic with static IPs
	// Google DNS
	cmd10 := "kubectl exec --namespace=" + ns + " " + podName + " -- wget --spider --timeout 3 https://8.8.8.8"
	execCommandAssertExitCode(t, cmd10, expectedExitCode)

	// Cloudflare DNS
	cmd11 := "kubectl exec --namespace=" + ns + " " + podName + " -- wget --spider --timeout 3 1.1.1.1"
	execCommandAssertExitCode(t, cmd11, expectedExitCode)

	// External traffic with domain names
	cmd12 := "kubectl exec --namespace=" + ns + " " + podName + " -- wget --spider --timeout 3 www.google.de"
	execCommandAssertExitCode(t, cmd12, expectedExitCode)

	cmd13 := "kubectl exec --namespace=" + ns + " " + podName + " -- wget --spider --timeout 3 www.sap.com"
	execCommandAssertExitCode(t, cmd13, expectedExitCode)
}

func testNetworkPolicyLevel(t *testing.T, podName string, level int) {

	var namespace *corev1.Namespace
	var err error

	namespace, err = f.CreateTestNamespace()
	if err != nil {
		t.Fatalf("failed to create test namespace: %v", err)
	}
	ns := namespace.ObjectMeta.Name

	if level == 2 {
		namespace.SetAnnotations(map[string]string{
			"karydia.gardener.cloud/networkPolicy": "network-policy-l2",
		})
		namespace, err = f.KubeClientset.CoreV1().Namespaces().Update(namespace)
		if err != nil {
			t.Fatal("failed to update test namespace:", err.Error())
		}
	} else if level == 3 {
		namespace.SetAnnotations(map[string]string{
			"karydia.gardener.cloud/networkPolicy": "network-policy-l3",
		})
		namespace, err = f.KubeClientset.CoreV1().Namespaces().Update(namespace)
		if err != nil {
			t.Fatal("failed to update test namespace:", err.Error())
		}
	}

	pod := getPodDescription(ns, podName)
	createdPod, err := f.KubeClientset.CoreV1().Pods(ns).Create(pod)
	if err != nil {
		t.Fatalf("Failed to create: " + err.Error())
	}
	timeout := 2 * time.Minute
	if err := f.WaitPodRunning(ns, podName, timeout); err != nil {
		t.Fatalf("pod never reached state running")
	}

	networkPolicyTestCases(t, podName, ns, level)

	err = f.KubeClientset.CoreV1().Pods(ns).Delete(createdPod.ObjectMeta.Name, &metav1.DeleteOptions{})
	if err != nil {
		t.Fatal("pod could not be deleted")
	}
}

func TestNetworkPolicyLevel1(t *testing.T) {
	testNetworkPolicyLevel(t, "karydia-network-policy-1", 1)
}

func TestNetworkPolicyLevel2(t *testing.T) {
	testNetworkPolicyLevel(t, "karydia-network-policy-2", 2)
}

func TestNetworkPolicyLevel3(t *testing.T) {
	testNetworkPolicyLevel(t, "karydia-network-policy-3", 3)
}

func int32Ptr(i int32) *int32 { return &i }

func getDeploymentDescription(ns string) *appsv1.Deployment {
	defaultUserAndGroup := int64(0)
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "network-policy-deployment",
			Namespace: ns,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(2),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "nginx",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "nginx",
					},
				},
				Spec: corev1.PodSpec{
					SecurityContext: &corev1.PodSecurityContext{
						RunAsUser:  &defaultUserAndGroup,
						RunAsGroup: &defaultUserAndGroup,
					},
					Containers: []corev1.Container{
						{
							Name:  "nginx",
							Image: "nginx",
							Ports: []corev1.ContainerPort{
								{
									Name:          "http",
									Protocol:      corev1.ProtocolTCP,
									ContainerPort: 80,
								},
							},
						},
					},
				},
			},
		},
	}
}

func getServiceDescription(ns string) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: "network-policy-service",
			Labels: map[string]string{
				"app": "nginx",
			},
			Namespace: ns,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name:       "foo",
					Port:       80,
					Protocol:   "TCP",
					TargetPort: intstr.FromInt(80),
				},
			},
			Selector: map[string]string{
				"app": "nginx",
			},
		},
	}
}

func testServiceDeployment(t *testing.T, level int) {

	var namespace *corev1.Namespace
	var err error

	namespace, err = f.CreateTestNamespace()
	/*
		WithAnnotation(map[string]string{
			"karydia.gardener.cloud/podSecurityContext": "none",
		})
	*/
	if err != nil {
		t.Fatalf("failed to create test namespace: %v", err.Error())
	}
	ns := namespace.ObjectMeta.Name

	deployment := getDeploymentDescription(ns)
	createdDeployment, err := f.KubeClientset.AppsV1().Deployments(ns).Create(deployment)
	if err != nil {
		t.Fatalf("failed to create deployment: %v", err.Error())
	}
	deploymentName := createdDeployment.ObjectMeta.Name

	service := getServiceDescription(ns)
	createdService, err := f.KubeClientset.CoreV1().Services(ns).Create(service)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err.Error())
	}
	serviceName := createdService.ObjectMeta.Name

	if level == 1 {

		podName := "karydia-service-deployment-test-1"
		pod := getPodDescription(ns, podName)
		if _, err := f.KubeClientset.CoreV1().Pods(ns).Create(pod); err != nil {
			t.Fatalf("Failed to create pod: %v", err.Error())
		}
		timeout := 2 * time.Minute
		if err := f.WaitPodRunning(ns, podName, timeout); err != nil {
			t.Fatalf("pod never reached state running: %v", err.Error())
		}

		expectedExitCode := Success
		clusterIP := createdService.Spec.ClusterIP
		clusterPort := fmt.Sprint(createdService.Spec.Ports[0].Port)

		cmd := "kubectl exec --namespace=" + ns + " " + podName + " -- wget --spider --timeout 3 " + clusterIP + ":" + clusterPort
		execCommandAssertExitCode(t, cmd, expectedExitCode)

		if err = f.KubeClientset.CoreV1().Pods(ns).Delete(podName, &metav1.DeleteOptions{}); err != nil {
			t.Fatalf("pod could not be deleted: %v", err.Error())
		}

	}
	if level == 2 || level == 3 {
		namespace.SetAnnotations(map[string]string{
			"karydia.gardener.cloud/networkPolicy": "network-policy-l" + fmt.Sprint(level),
		})
		if namespace, err = f.KubeClientset.CoreV1().Namespaces().Update(namespace); err != nil {
			t.Fatalf("failed to update test namespace: %v", err.Error())
		}

		podName := "karydia-service-deployment-test-" + fmt.Sprint(level)
		pod := getPodDescription(ns, podName)
		if _, err := f.KubeClientset.CoreV1().Pods(ns).Create(pod); err != nil {
			t.Fatalf("Failed to create pod: %v", err.Error())
		}
		timeout := 2 * time.Minute
		if err := f.WaitPodRunning(ns, podName, timeout); err != nil {
			t.Fatalf("pod never reached state running: %v", err.Error())
		}

		expectedExitCode := TimeOut
		clusterIP := createdService.Spec.ClusterIP
		clusterPort := fmt.Sprint(createdService.Spec.Ports[0].Port)

		cmd := "kubectl exec --namespace=" + ns + " " + podName + " -- wget --spider --timeout 3 " + clusterIP + ":" + clusterPort
		execCommandAssertExitCode(t, cmd, expectedExitCode)

		if err = f.KubeClientset.CoreV1().Pods(ns).Delete(podName, &metav1.DeleteOptions{}); err != nil {
			t.Fatalf("pod could not be deleted: %v", err.Error())
		}
	}

	if err = f.KubeClientset.CoreV1().Services(ns).Delete(serviceName, &metav1.DeleteOptions{}); err != nil {
		t.Fatal("service could not be deleted: %v" + err.Error())
	}

	if err = f.KubeClientset.AppsV1().Deployments(ns).Delete(deploymentName, &metav1.DeleteOptions{}); err != nil {
		t.Fatalf("Failed to delete: " + err.Error())
	}

}

func TestServiceDeploymentLevel1(t *testing.T) {
	testServiceDeployment(t, 1)
}

func TestServiceDeploymentLevel2(t *testing.T) {
	testServiceDeployment(t, 2)
}

func TestServiceDeploymentLevel3(t *testing.T) {
	testServiceDeployment(t, 3)
}
