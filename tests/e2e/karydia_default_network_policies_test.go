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

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	defaultNetworkPolicyNames = [3]string{"karydia-default-network-policy-l1", "karydia-default-network-policy-l2", "karydia-default-network-policy-l3"}
)

func TestNetworkPolicyLevel1(t *testing.T) {
	err := updateDefaultNetworkPolicy(defaultNetworkPolicyNames[0])
	if err != nil {
		t.Fatal("Could not change defaultNetworkPolicies in karydiaConfig", err)
	}

	// security implications of network policy l1
	testMetaDataServicesReachability(t)

	testHostNetworkReachability(t)

	testCommunicationToKubeSystemNamespace(t)

	// allowed communication
	testEgressCommunication(t)
	testDifferentNamespaceReachability(t)
	testSameNamespaceReachability(t)
}

func testMetaDataServicesReachability(t *testing.T) {
	namespace, err := f.CreateTestNamespace()
	if err != nil {
		t.Fatal("failed to create test namespace:", err)
	}

	namespaceName := namespace.getName()

	timeout := 3000 * time.Millisecond
	if err := f.WaitNetworkPolicyCreated(namespaceName, defaultNetworkPolicyNames[0], timeout); err != nil {
		t.Fatal("failed to create default network policy for new namespace:", err)
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "karydia-network-policy-1",
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

	if err := f.WaitPodRunning(namespaceName, createdPod.getName(), timeout); err != nil {
		t.Fatal("Pod never reached state running", err.Error())
	}

	podName := createdPod.ObjectMeta.Name

	destinations := [3]string{
		"169.254.169.254", // meta data services (Azure, AWS, GCP, OpenStack)
		"100.100.100.200", // meta data services (AliCloud)
	}

	for destination := range destinations {
		cmd := "kubectl exec --namespace=" + namespaceName + " " + podName + " -- wget --spider --timeout 3 " + destination
		execCommandAssertExitCode(t, cmd6, TimeOut)
	}

	err = f.KubeClientset.CoreV1().Pods(namespaceName).Delete(podName, &metav1.DeleteOptions{})
	if err != nil {
		t.Fatal("Pod could not be deleted", err.Error())
	}
}

func testHostNetworkReachability(t *testing.T) {

}

func testCommunicationToKubeSystemNamespace(t *testing.T) {

}

func TestNetworkPolicyLevel2(t *testing.T) {
	err := updateDefaultNetworkPolicy(defaultNetworkPolicyNames[1])
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
}

func testEgressCommunication(t *testing.T) {

}

func TestNetworkPolicyLevel3(t *testing.T) {
	err := updateDefaultNetworkPolicy(defaultNetworkPolicyNames[2])
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
