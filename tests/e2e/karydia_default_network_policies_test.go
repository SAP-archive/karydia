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
)

const (
	Success int = 0
	TimeOut int = 1
)

var (
	defaultNetworkPolicyNames = [3]string{"karydia-default-network-policy-l1", "karydia-default-network-policy-l2", "karydia-default-network-policy-l3"}
)

func TestNetworkPolicyLevel1(t *testing.T) {
	err := updateDefaultNetworkPolicy(defaultNetworkPolicyNames[0])
	if err != nil {
		t.Fatal("Could not change defaultNetworkPolicies in karydiaConfig", err)
	}

	// Create test namespace
	namespace, err := f.CreateTestNamespace()
	if err != nil {
		t.Fatal("failed to create test namespace:", err)
	}

	namespaceName := namespace.ObjectMeta.Name

	timeout := 3000 * time.Millisecond
	if err := f.WaitNetworkPolicyCreated(namespaceName, defaultNetworkPolicyNames[0], timeout); err != nil {
		t.Fatal("failed to create default network policy for new namespace:", err)
	}

	// Create test pod
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "karydia-network-policy-l1",
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

	podName := createdPod.ObjectMeta.Name

	if err := f.WaitPodRunning(namespaceName, podName, 2*timeout); err != nil {
		t.Fatal("Pod never reached state running", err.Error())
	}

	// security implications of network policy l1
	testMetaDataServicesReachability(t, namespaceName, podName, TimeOut)

	testHostNetworkReachability(t, namespaceName, podName, TimeOut)

	testCommunicationToKubeSystemNamespace(t, namespaceName, podName, TimeOut)

	// allowed communication
	testEgressCommunication(t, namespaceName, podName, Success)
	// testDifferentNamespaceReachability(t)
	// testSameNamespaceReachability(t)

	// Delete test pod
	err = f.KubeClientset.CoreV1().Pods(namespaceName).Delete(podName, &metav1.DeleteOptions{})
	if err != nil {
		t.Fatal("Pod could not be deleted", err.Error())
	}

	err = f.KubeClientset.CoreV1().Namespaces().Delete(namespaceName, &metav1.DeleteOptions{})
	if err != nil {
		t.Fatal("Namespace could not be deleted", err.Error())
	}
}

func testMetaDataServicesReachability(t *testing.T, namespaceName string, podName string, expectedExitCode int) {
	destination := "169.254.169.254" // meta data services (Azure, AWS, GCP, OpenStack)

	cmd := "kubectl exec --namespace=" + namespaceName + " " + podName + " -- wget --spider --timeout 3 " + destination
	execCommandAssertExitCode(t, cmd, expectedExitCode)
}

func testHostNetworkReachability(t *testing.T, namespaceName string, podName string, expectedExitCode int) {
	destinations := [3]string{
		"10.250.0.0",
		"10.250.255.255",
		"10.250.100.100",
	}

	for _, destination := range destinations {
		cmd := "kubectl exec --namespace=" + namespaceName + " " + podName + " -- wget --spider --timeout 3 " + destination
		execCommandAssertExitCode(t, cmd, expectedExitCode)
	}
}

func testCommunicationToKubeSystemNamespace(t *testing.T, namespaceName string, podName string, expectedExitCode int) {
	// creeate deployment in "kube-system" namespace
	deploymentSepc := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-deployment",
			Namespace: "kube-system",
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{
				"run": "test-deployment",
			},
			},
			Replicas: func() *int32 { i := int32(1); return &i }(),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"run": "test-deployment",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "test-deployment",
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

	deployment, err := f.KubeClientset.AppsV1().Deployments("kube-system").Create(&deploymentSepc)
	if err != nil {
		t.Fatal("Failed to create service:", err.Error())
	}

	deploymentName := deployment.ObjectMeta.Name

	// create service in "kube-system" namespace
	serviceSpec := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-deployment",
			Namespace: "kube-system",
			Labels: map[string]string{
				"run": "test-deployment",
			},
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{Port: 80, Protocol: "TCP"},
			},
			Selector: map[string]string{
				"run": "test-deployment",
			},
		},
	}

	service, err := f.KubeClientset.CoreV1().Services("kube-system").Create(serviceSpec)
	if err != nil {
		t.Fatal("Failed to create service:", err.Error())
	}

	serviceName := service.ObjectMeta.Name
	timeout := 3000 * time.Millisecond

	if err := f.WaitEndpointCreated("kube-system", serviceName, 5*timeout); err != nil {
		t.Fatal("Endpoint was never fully created")
	}

	endpoints, err := f.KubeClientset.CoreV1().Endpoints("kube-system").Get(serviceName, metav1.GetOptions{})
       	if err != nil {
        	t.Fatal("Could not get endpoint", err.Error())
        }	

	serviceIp := endpoints.Subsets[0].Addresses[0].IP // ip of the test-deployment service

	destinations := [2]string{
		"test-deployment.kube-system",
		serviceIp,
	}

	for _, destination := range destinations {
		cmd := "kubectl exec --namespace=" + namespaceName + " " + podName + " -- wget --spider --timeout 3 " + destination
		execCommandAssertExitCode(t, cmd, expectedExitCode)
	}

	// clean-up deplyoment and service
	err = f.KubeClientset.AppsV1().Deployments("kube-system").Delete(deploymentName, &metav1.DeleteOptions{})
	if err != nil {
		t.Fatal("Deployment could not be deleted", err.Error())
	}

	err = f.KubeClientset.CoreV1().Services("kube-system").Delete(serviceName, &metav1.DeleteOptions{})
	if err != nil {
		t.Fatal("Service could not be deleted", err.Error())
	}
}

func TestNetworkPolicyLevel2(t *testing.T) {
	/*err := updateDefaultNetworkPolicy(defaultNetworkPolicyNames[1])
	if err != nil {
		t.Fatal("Could not change defaultNetworkPolicies in karydiaConfig", err)
	}

		// security implications of network policy l1
		testMetaDataServicesReachability(t)

		testHostNetworkReachability(t)

		testCommunicationToKubeSystemNamespace(t)

		// security implications of network policy l2
		testEgressCommunication(t)

		// allowed communication
		testDifferentNamespaceReachability(t)
		testSameNamespaceReachability(t)
	*/
}

func testEgressCommunication(t *testing.T, namespaceName string, podName string, expectedExitCode int) {
	/*destinations := [3]string{
		"https://8.8.8.8", // Google DNS
		"google.com",      // Google domain
		"sap.com",         // SAP domain
	}*/

        destinations := [2]string{
                "google.com",      // Google domain
                "sap.com",         // SAP domain
        }

	for _, destination := range destinations {
		cmd := "kubectl exec --namespace=" + namespaceName + " " + podName + " -- wget --spider --timeout 3 " + destination
		execCommandAssertExitCode(t, cmd, expectedExitCode)
	}
}

func TestNetworkPolicyLevel3(t *testing.T) {
	/*err := updateDefaultNetworkPolicy(defaultNetworkPolicyNames[2])
	if err != nil {
		t.Fatal("Could not change defaultNetworkPolicies in karydiaConfig", err)
	}

		// security implications of network policy l1
		testMetaDataServicesReachability(t)

		testHostNetworkReachability(t)

		testCommunicationToKubeSystemNamespacet()

		// security implications of network policy l2
		testEgressCommunication(t)

		// security implications of network policy l3
		testDifferentNamespaceReachability(t)

		// allowed communication
		testSameNamespaceReachability(t)
	*/
}

func testDifferentNamespaceReachability(t *testing.T) {

}

func testSameNamespaceReachability(t *testing.T) {

}

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

func updateDefaultNetworkPolicy(defaultNetworkPolicies string) error {
	curKarydiaConfig, curErr := f.KarydiaClientset.KarydiaV1alpha1().KarydiaConfigs().Get("karydia-config", metav1.GetOptions{})

	if curErr != nil {
		return curErr
	}

	curKarydiaConfig.Spec.NetworkPolicies = defaultNetworkPolicies
	f.KarydiaClientset.KarydiaV1alpha1().KarydiaConfigs().Update(curKarydiaConfig)

	newKarydiaConfig, newErr := f.KarydiaClientset.KarydiaV1alpha1().KarydiaConfigs().Get("karydia-config", metav1.GetOptions{})

	if newErr != nil {
		return newErr
	}

	if newKarydiaConfig.Spec.NetworkPolicies != defaultNetworkPolicies {
		return fmt.Errorf("defaultNetworkPolicies in karydiaConfig did not change but should")
	}

	return nil
}
