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
	"strings"
	"syscall"
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	SuccessInt int           = 0
	TimeOutInt int           = 1
	timeout    time.Duration = 3000 * time.Millisecond
)

func TestNetworkPolicies(t *testing.T) {
	// ===== set-up =====
	// set-up test namespace
	namespaceTest, namespaceName := setUpTestNamespace(t)

	// set-up test pod
	podName := setUpPod(t, namespaceName, "karydia-test-network-policy-pod")

	// set-up same namespace deployment
	deploymentName := setUpDeployment(t, namespaceName, "karydia-test-network-policy-deployment")

	// set-up same namespace service
	serviceName, destinationsTest := setUpService(t, namespaceName, "karydia-test-network-policy-service", deploymentName)

	// set-up other namespace
	_, namespaceNameOther := setUpTestNamespace(t)

	// set-up other namespace deployment
	deploymentNameOther := setUpDeployment(t, namespaceNameOther, "karydia-test-network-policy-deployment")

	// set-up other namespace service
	serviceNameOther, destinationsOther := setUpService(t, namespaceNameOther, "karydia-test-network-policy-service", deploymentNameOther)

	// set-up kube-system deployment
	deploymentNameKubeSystem := setUpDeployment(t, "kube-system", "karydia-test-network-policy-deployment")

	// set-up kube-system service
	serviceNameKubeSystem, destinationsKubeSystem := setUpService(t, "kube-system", "karydia-test-network-policy-service", deploymentNameKubeSystem)

	destinationsMetadata := [1]string{
		"169.254.169.254", // meta data services (Azure, AWS, GCP, OpenStack)
	}

	destinationsHostNetwork := [3]string{
		"10.250.0.0",
		"10.250.255.255",
		"10.250.100.100",
	}

	destinationsEgress := [3]string{
		"https://8.8.8.8", // Google DNS
		"google.com",      // Google domain
		"sap.com",         // SAP domain
	}

	// ===== run tests =====
	namespaceTest = testNetworkPolicyLevel1(t, podName, namespaceTest, namespaceName, destinationsMetadata, destinationsHostNetwork, destinationsEgress, destinationsKubeSystem[:], destinationsTest[:], destinationsOther[:])
	namespaceTest = testNetworkPolicyLevel2(t, podName, namespaceTest, namespaceName, destinationsMetadata, destinationsHostNetwork, destinationsEgress, destinationsKubeSystem[:], destinationsTest[:], destinationsOther[:])
	namespaceTest = testNetworkPolicyLevel3(t, podName, namespaceTest, namespaceName, destinationsMetadata, destinationsHostNetwork, destinationsEgress, destinationsKubeSystem[:], destinationsTest[:], destinationsOther[:])

	// ===== clean-up =====
	cleanUp(t, namespaceName, deploymentName, serviceName, podName, deploymentNameKubeSystem, serviceNameKubeSystem, namespaceNameOther, deploymentNameOther, serviceNameOther)
}

func setUpTestNamespace(t *testing.T) (*corev1.Namespace, string) {
	namespace, err := f.CreateTestNamespace()
	if err != nil {
		t.Fatal("failed to create test namespace:", err)
	}

	return namespace, namespace.ObjectMeta.Name
}

func setUpPod(t *testing.T, namespaceName string, podName string) string {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: namespaceName,
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

	createdPod, err := f.KubeClientset.CoreV1().Pods(namespaceName).Create(pod)
	if err != nil {
		t.Fatal("Failed to create pod:", err.Error())
	}

	if err := f.WaitPodRunning(namespaceName, podName, 2*timeout); err != nil {
		t.Fatal("Pod never reached state running", err.Error())
	}

	return createdPod.ObjectMeta.Name
}

func setUpDeployment(t *testing.T, namespaceName string, deploymentName string) string {
	deploymentSepc := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      deploymentName,
			Namespace: namespaceName,
			Labels: map[string]string{
				"app": "karydia",
			},
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{
				"run": deploymentName,
			},
			},
			Replicas: func() *int32 { i := int32(1); return &i }(),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"run": deploymentName,
						"app": "karydia",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  deploymentName,
							Image: "nginx",
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 80,
								},
							},
						},
					},
				},
			},
		},
	}

	deployment, err := f.KubeClientset.AppsV1().Deployments(namespaceName).Create(&deploymentSepc)
	if err != nil {
		t.Fatal("Failed to create deployment:", err.Error())
	}

	if err := f.WaitDeploymentCreated(namespaceName, deployment.ObjectMeta.Name, timeout); err != nil {
		t.Fatal("Failed to get deployment", err.Error())
	}

	return deployment.ObjectMeta.Name
}

func setUpService(t *testing.T, namespaceName string, serviceName string, deploymentName string) (string, []string) {
	serviceSpec := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceName,
			Namespace: namespaceName,
			Labels: map[string]string{
				"run": deploymentName,
			},
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{Port: 80, Protocol: "TCP"},
			},
			Selector: map[string]string{
				"run": deploymentName,
			},
		},
	}

	service, err := f.KubeClientset.CoreV1().Services(namespaceName).Create(serviceSpec)
	if err != nil {
		t.Fatal("Failed to create service:", err.Error())
	}

	if err := f.WaitEndpointCreated(namespaceName, serviceName, 10*timeout); err != nil {
		t.Fatal("Endpoint was never fully created")
	}

	endpoints, err := f.KubeClientset.CoreV1().Endpoints(namespaceName).Get(serviceName, metav1.GetOptions{})
	if err != nil {
		t.Fatal("Could not get endpoint", err.Error())
	}

	var destinations []string
	destinations = append(destinations, serviceName+"."+namespaceName)

	for _, subset := range endpoints.Subsets {
		for _, address := range subset.Addresses {
			destinations = append(destinations, address.IP)
		}
	}

	return service.ObjectMeta.Name, destinations[:]
}

func cleanUp(t *testing.T, namespaceName string, deploymentName string, serviceName string, podName string, deploymentNameKubeSystem string, serviceNameKubeSystem string, namespaceNameOther string, deploymentNameOther string, serviceNameOther string) {
	// delete test pod
	err := f.KubeClientset.CoreV1().Pods(namespaceName).Delete(podName, &metav1.DeleteOptions{})
	if err != nil {
		t.Log("Pod (test) could not be deleted", err.Error())
	}

	// delete test deployment
	err = f.KubeClientset.AppsV1().Deployments(namespaceName).Delete(deploymentName, &metav1.DeleteOptions{})
	if err != nil {
		t.Log("Deployment (test) could not be deleted", err.Error())
	}

	// delete test service
	err = f.KubeClientset.CoreV1().Services(namespaceName).Delete(serviceName, &metav1.DeleteOptions{})
	if err != nil {
		t.Log("Service (test) could not be deleted", err.Error())
	}

	// delete test namespace
	err = f.KubeClientset.CoreV1().Namespaces().Delete(namespaceName, &metav1.DeleteOptions{})
	if err != nil {
		t.Log("Namespace (test) could not be deleted", err.Error())
	}

	// delete kube-system deployment
	err = f.KubeClientset.AppsV1().Deployments("kube-system").Delete(deploymentNameKubeSystem, &metav1.DeleteOptions{})
	if err != nil {
		t.Log("Deployment (kube-system) could not be deleted", err.Error())
	}

	// delete kube-system service
	err = f.KubeClientset.CoreV1().Services("kube-system").Delete(serviceNameKubeSystem, &metav1.DeleteOptions{})
	if err != nil {
		t.Log("Service (kube-system) could not be deleted", err.Error())
	}

	// delete other deployment
	err = f.KubeClientset.AppsV1().Deployments(namespaceNameOther).Delete(deploymentNameOther, &metav1.DeleteOptions{})
	if err != nil {
		t.Log("Deployment (other) could not be deleted", err.Error())
	}

	// delete other service
	err = f.KubeClientset.CoreV1().Services(namespaceNameOther).Delete(serviceNameOther, &metav1.DeleteOptions{})
	if err != nil {
		t.Log("Service (other) could not be deleted", err.Error())
	}

	// delete other namespace
	err = f.KubeClientset.CoreV1().Namespaces().Delete(namespaceNameOther, &metav1.DeleteOptions{})
	if err != nil {
		t.Log("Namespace (other) could not be deleted", err.Error())
	}
}

func testNetworkPolicyLevel1(t *testing.T, testPodName string, namespaceTest *corev1.Namespace, testNamespaceName string, destinationsMetadata [1]string, destinationsHostNetwork [3]string, destinationsEgress [3]string, destinationsKubeSystem []string, destinationsTest []string, destinationsOther []string) *corev1.Namespace {
	annotations := make(map[string]string)
	annotations["karydia.gardener.cloud/networkPolicy"] = defaultNetworkPolicyNames[0]
	namespaceTest.SetAnnotations(annotations)
	namespaceTest, err := f.KubeClientset.CoreV1().Namespaces().Update(namespaceTest)
	if err != nil {
		t.Fatal("failed to update test namespace:", err)
	}

	if err := f.WaitNetworkPolicyCreated(testNamespaceName, defaultNetworkPolicyNames[0], timeout); err != nil {
		t.Fatal("failed to create default network policy for new namespace:", err)
	}

	// ===== security implications of network policy l1 =====
	// test metadata service reachability
	testReachability(t, testNamespaceName, testPodName, destinationsMetadata[:], TimeOutInt)

	// test host network reachability
	testReachability(t, testNamespaceName, testPodName, destinationsHostNetwork[:], TimeOutInt)

	// test kube-system namespace reachability
	testReachability(t, testNamespaceName, testPodName, destinationsKubeSystem[:], TimeOutInt)

	// ===== allowed communication =====
	// test egress reachability
	testReachability(t, testNamespaceName, testPodName, destinationsEgress[:], SuccessInt)

	// test other namespace reachability
	testReachability(t, testNamespaceName, testPodName, destinationsOther[:], SuccessInt)

	// test same namespace reachability
	testReachability(t, testNamespaceName, testPodName, destinationsTest[:], SuccessInt)

	return namespaceTest
}

func testNetworkPolicyLevel2(t *testing.T, testPodName string, namespaceTest *corev1.Namespace, testNamespaceName string, destinationsMetadata [1]string, destinationsHostNetwork [3]string, destinationsEgress [3]string, destinationsKubeSystem []string, destinationsTest []string, destinationsOther []string) *corev1.Namespace {
	annotations := make(map[string]string)
	annotations["karydia.gardener.cloud/networkPolicy"] = defaultNetworkPolicyNames[1]
	namespaceTest.SetAnnotations(annotations)
	namespaceTest, err := f.KubeClientset.CoreV1().Namespaces().Update(namespaceTest)
	if err != nil {
		t.Fatal("failed to update test namespace:", err)
	}

	if err := f.WaitNetworkPolicyCreated(testNamespaceName, defaultNetworkPolicyNames[1], timeout); err != nil {
		t.Fatal("failed to create default network policy for new namespace:", err)
	}

	// ===== security implications of network policy l1 =====
	// test metadata service reachability
	testReachability(t, testNamespaceName, testPodName, destinationsMetadata[:], TimeOutInt)

	// test host network reachability
	testReachability(t, testNamespaceName, testPodName, destinationsHostNetwork[:], TimeOutInt)

	// test kube-system namespace reachability
	testReachability(t, testNamespaceName, testPodName, destinationsKubeSystem[:], TimeOutInt)

	// ===== security implications of network policy l2 =====
	// test egress reachability
	testReachability(t, testNamespaceName, testPodName, destinationsEgress[:], TimeOutInt)

	// ===== allowed communication =====
	// test other namespace reachability
	testReachability(t, testNamespaceName, testPodName, destinationsOther[:], SuccessInt)

	// test same namespace reachability
	testReachability(t, testNamespaceName, testPodName, destinationsTest[:], SuccessInt)

	return namespaceTest
}

func testNetworkPolicyLevel3(t *testing.T, testPodName string, namespaceTest *corev1.Namespace, testNamespaceName string, destinationsMetadata [1]string, destinationsHostNetwork [3]string, destinationsEgress [3]string, destinationsKubeSystem []string, destinationsTest []string, destinationsOther []string) *corev1.Namespace {
	annotations := make(map[string]string)
	annotations["karydia.gardener.cloud/networkPolicy"] = defaultNetworkPolicyNames[2]
	namespaceTest.SetAnnotations(annotations)
	namespaceTest, err := f.KubeClientset.CoreV1().Namespaces().Update(namespaceTest)
	if err != nil {
		t.Fatal("failed to update test namespace:", err)
	}

	if err := f.WaitNetworkPolicyCreated(testNamespaceName, defaultNetworkPolicyNames[2], timeout); err != nil {
		t.Fatal("failed to create default network policy for new namespace:", err)
	}

	// ===== security implications of network policy l1 =====
	// test metadata service reachability
	testReachability(t, testNamespaceName, testPodName, destinationsMetadata[:], TimeOutInt)

	// test host network reachability
	testReachability(t, testNamespaceName, testPodName, destinationsHostNetwork[:], TimeOutInt)

	// test kube-system namespace reachability
	testReachability(t, testNamespaceName, testPodName, destinationsKubeSystem[:], TimeOutInt)

	// ===== security implications of network policy l2 =====
	// test egress reachability
	testReachability(t, testNamespaceName, testPodName, destinationsEgress[:], TimeOutInt)

	// ===== security implications of network policy l3 =====
	// test other namespace reachability
	testReachability(t, testNamespaceName, testPodName, destinationsOther[:], TimeOutInt)

	// ===== allowed communication =====
	// test same namespace reachability
	testReachability(t, testNamespaceName, testPodName, destinationsTest[:], SuccessInt)

	return namespaceTest
}

func testReachability(t *testing.T, namespaceName string, podName string, destinations []string, expectedExitCode int) {
	for _, destination := range destinations {
		cmd := "kubectl exec --namespace=" + namespaceName + " " + podName + " -- wget --spider --timeout 3 " + destination
		execCommandAssertExitCode(t, cmd, expectedExitCode)
	}
}

func execCommandAssertExitCode(t *testing.T, command string, expectedExitCode int) {
	exitCode := SuccessInt

	args := strings.Fields(command)
	cmd := exec.Command(args[0], args[1:]...)
	err := cmd.Run()

	if err != nil {
		// try to get the exit code
		if exitError, ok := err.(*exec.ExitError); ok {
			ws := exitError.Sys().(syscall.WaitStatus)
			exitCode = ws.ExitStatus()

			if exitCode != expectedExitCode {
				t.Fatal("Exit status with unexpected code:", exitCode, expectedExitCode, command)
			}
		} else {
			t.Fatal("Could not get exit code.")
		}
	} else {
		// success, exitCode should be 0 if go is ok
		ws := cmd.ProcessState.Sys().(syscall.WaitStatus)
		exitCode = ws.ExitStatus()

		if exitCode != expectedExitCode {
			t.Fatal("Exit status with unexpected code:", exitCode, expectedExitCode, command)
		}
	}
}
