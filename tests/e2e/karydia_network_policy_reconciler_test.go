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
	"bytes"
	"testing"
	"time"

	//	"github.com/karydia/karydia/pkg/apis/karydia/v1alpha1"
	networkingv1 "k8s.io/api/networking/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	//	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	defaultNetworkPolicyNames = [3]string{"karydia-default-network-policy", "karydia-default-network-policy-l2", "karydia-default-network-policy-l3"}

//	defaultNetworkPolicyName   = "karydia-default-network-policy"
//	defaultNetworkPolicyL2Name = "karydia-default-network-policy-l2"
)

// e2e tests for single network policy

/*func TestCreateKarydiaNetworkPolicyForNewNamespace(t *testing.T) {
	defaultNetworkPolicy := &networkingv1.NetworkPolicy{}
	defaultNetworkPolicy.Name = defaultNetworkPolicyName
	karydiaNetworkpolicy, err := f.KarydiaClientset.KarydiaV1alpha1().KarydiaNetworkPolicies().Get(defaultNetworkPolicyName, meta_v1.GetOptions{})
	if err != nil {
		t.Fatal("failed to get karydia default network policy:", err)
	}
	defaultNetworkPolicy.Spec = *karydiaNetworkpolicy.Spec.DeepCopy()
	namespace, err := f.CreateTestNamespace()
	if err != nil {
		t.Fatal("failed to create test namespace:", err)
	}

	timeout := 3000 * time.Millisecond
	if err := f.WaitNetworkPolicyCreated(namespace.GetName(), defaultNetworkPolicyName, timeout); err != nil {
		t.Fatal("failed to create default network policy for new namespace:", err)
	}

	namespaceNetworkPolicy, err := f.KubeClientset.NetworkingV1().NetworkPolicies(namespace.GetName()).Get(defaultNetworkPolicyName, meta_v1.GetOptions{})
	if err != nil {
		t.Fatal("failed to create default network policy for new namespace:", err)
	}

	if !networkPoliciesAreEqual(namespaceNetworkPolicy, defaultNetworkPolicy) {
		t.Fatal("Network policy for created namespace is not equal to the default network policy:", err)
	}

	err = f.KubeClientset.NetworkingV1().NetworkPolicies(namespace.GetName()).Delete(defaultNetworkPolicyName, &meta_v1.DeleteOptions{})
	if err != nil {
		t.Fatal("failed to delete default network policy for new namespace:", err)
	}

	if err := f.WaitNetworkPolicyCreated(namespace.GetName(), defaultNetworkPolicyName, timeout); err != nil {
		t.Fatal("failed to create default network policy for new namespace:", err)
	}

	namespaceNetworkPolicy, err = f.KubeClientset.NetworkingV1().NetworkPolicies(namespace.GetName()).Get(defaultNetworkPolicyName, meta_v1.GetOptions{})
	if err != nil {
		t.Fatal("Reconciler failed to recreate default network policy for new namespace:", err)
	}
	//Update NP
	namespaceNetworkPolicy.Spec = networkingv1.NetworkPolicySpec{
		PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeIngress}}

	updatedNetworkPolicy, err := f.KubeClientset.NetworkingV1().NetworkPolicies(namespace.GetName()).Update(namespaceNetworkPolicy)
	if err != nil {
		t.Fatal("failed to update default network policy for new namespace:", err)
	}

	duration := 3 * time.Second
	time.Sleep(duration)

	updatedNetworkPolicy, err = f.KubeClientset.NetworkingV1().NetworkPolicies(namespace.GetName()).Get(defaultNetworkPolicyName, meta_v1.GetOptions{})
	if err != nil {
		t.Fatal("failed to get default network policy for new namespace:", err)
	}
	if !networkPoliciesAreEqual(updatedNetworkPolicy, defaultNetworkPolicy) {
		t.Fatal("Reconcilation failed after network policy has changed")
	}
}

func TestCreateKarydiaNetworkPolicyForAnnotatedNamespace(t *testing.T) {
	defaultNetworkPolicy := &networkingv1.NetworkPolicy{}
	defaultNetworkPolicy.Name = defaultNetworkPolicyL2Name
	defaultNetworkPolicy.Spec = networkingv1.NetworkPolicySpec{
		PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeIngress},
	}

	defaultKarydiaNetworkPolicyL2 := &v1alpha1.KarydiaNetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				"app": "karydia-e2e-test",
			},
			Name: defaultNetworkPolicyL2Name,
		},
		Spec: networkingv1.NetworkPolicySpec{
			PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeIngress},
		},
	}

	_, err := f.KarydiaClientset.KarydiaV1alpha1().KarydiaNetworkPolicies().Create(defaultKarydiaNetworkPolicyL2)
	if err != nil {
		t.Fatal("failed to create:", defaultKarydiaNetworkPolicyL2)
	}

	annotations := make(map[string]string)
	annotations["karydia.gardener.cloud/networkPolicy"] = defaultNetworkPolicyL2Name
	namespace, err := f.CreateTestNamespaceWithAnnotation(annotations)
	if err != nil {
		t.Fatal("failed to create test namespace:", err)
	}

	timeout := 3000 * time.Millisecond
	if err := f.WaitNetworkPolicyCreated(namespace.GetName(), defaultNetworkPolicyL2Name, timeout); err != nil {
		t.Fatal("failed to create default network policy for new namespace:", err)
	}

	_, err = f.KubeClientset.NetworkingV1().NetworkPolicies(namespace.GetName()).Get(defaultNetworkPolicyName, meta_v1.GetOptions{})
	if err == nil {
		t.Fatal("Default level 1 network policy should not be found")
	}

	namespaceNetworkPolicy, err := f.KubeClientset.NetworkingV1().NetworkPolicies(namespace.GetName()).Get(defaultNetworkPolicyL2Name, meta_v1.GetOptions{})
	if err != nil {
		t.Fatal("failed to create default network policy for new namespace:", err)
	}

	if !networkPoliciesAreEqual(namespaceNetworkPolicy, defaultNetworkPolicy) {
		t.Fatal("Network policy for created namespace is not equal to the default network policy:", err)
	}

	if err := f.KarydiaClientset.KarydiaV1alpha1().KarydiaNetworkPolicies().Delete(defaultNetworkPolicyL2Name, &meta_v1.DeleteOptions{}); err != nil {
		t.Fatal("Failed to delete karydia default network policy l2:", err)
	}
}

func TestCreateNamespaceAndUpdateWithAnnotation(t *testing.T) {
	defaultNetworkPolicy := &networkingv1.NetworkPolicy{}
	defaultNetworkPolicy.Name = defaultNetworkPolicyL2Name
	defaultNetworkPolicy.Spec = networkingv1.NetworkPolicySpec{
		PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeIngress},
	}

	defaultKarydiaNetworkPolicyL2 := &v1alpha1.KarydiaNetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				"app": "karydia-e2e-test",
			},
			Name: defaultNetworkPolicyL2Name,
		},
		Spec: networkingv1.NetworkPolicySpec{
			PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeIngress},
		},
	}

	_, err := f.KarydiaClientset.KarydiaV1alpha1().KarydiaNetworkPolicies().Create(defaultKarydiaNetworkPolicyL2)
	if err != nil {
		t.Fatal("failed to create:", defaultKarydiaNetworkPolicyL2)
	}

	namespace, err := f.CreateTestNamespace()
	if err != nil {
		t.Fatal("failed to create test namespace:", err)
	}

	timeout := 3000 * time.Millisecond
	if err := f.WaitNetworkPolicyCreated(namespace.GetName(), defaultNetworkPolicyName, timeout); err != nil {
		t.Fatal("failed to create default network policy for new namespace:", err)
	}

	namespaceNetworkPolicy, err := f.KubeClientset.NetworkingV1().NetworkPolicies(namespace.GetName()).Get(defaultNetworkPolicyName, meta_v1.GetOptions{})
	if err != nil {
		t.Fatal("failed to create default network policy for new namespace:", err)
	}

	annotations := make(map[string]string)
	annotations["karydia.gardener.cloud/networkPolicy"] = defaultNetworkPolicyL2Name
	namespace.SetAnnotations(annotations)
	namespace, err = f.KubeClientset.CoreV1().Namespaces().Update(namespace)
	if err != nil {
		t.Fatal("failed to update test namespace:", err)
	}

	if err := f.WaitNetworkPolicyCreated(namespace.GetName(), defaultNetworkPolicyL2Name, timeout); err != nil {
		t.Fatal("failed to create default network policy for new namespace:", err)
	}

	_, err = f.KubeClientset.NetworkingV1().NetworkPolicies(namespace.GetName()).Get(defaultNetworkPolicyName, meta_v1.GetOptions{})
	if err == nil {
		t.Fatal("Default level 1 network policy should not be found")
	}

	namespaceNetworkPolicy, err = f.KubeClientset.NetworkingV1().NetworkPolicies(namespace.GetName()).Get(defaultNetworkPolicyL2Name, meta_v1.GetOptions{})
	if err != nil {
		t.Fatal("failed to create default network policy for new namespace:", err)
	}

	if !networkPoliciesAreEqual(namespaceNetworkPolicy, defaultNetworkPolicy) {
		t.Fatal("Network policy for created namespace is not equal to the default network policy:", err)
	}

	if err := f.KarydiaClientset.KarydiaV1alpha1().KarydiaNetworkPolicies().Delete(defaultNetworkPolicyL2Name, &meta_v1.DeleteOptions{}); err != nil {
		t.Fatal("Failed to delete karydia default network policy l2:", err)
	}
}

func TestGetKarydiaNetworkPolicyForExcludedNamespace(t *testing.T) {
	if _, err := f.KubeClientset.NetworkingV1().NetworkPolicies("kube-system").Get(defaultNetworkPolicyName, meta_v1.GetOptions{}); err == nil {
		t.Fatal("Default network policy should not be found for excluded namespace")
	}

}*/

// e2e tests for multiple default network policies

func TestCreateMultipleKarydiaNetworkPoliciesForNewNamespace(t *testing.T) {
	defaultNetworkPolicies := make(map[string]*networkingv1.NetworkPolicy, 3)

	for _, dnpName := range defaultNetworkPolicyNames {
		defaultNetworkPolicies[dnpName] = &networkingv1.NetworkPolicy{}
		defaultNetworkPolicies[dnpName].Name = dnpName

		karydiaNetworkpolicy, err := f.KarydiaClientset.KarydiaV1alpha1().KarydiaNetworkPolicies().Get(dnpName, meta_v1.GetOptions{})
		if err != nil {
			t.Fatal("failed to get karydia default network policy("+dnpName+"):", err)
		}

		defaultNetworkPolicies[dnpName].Spec = *karydiaNetworkpolicy.Spec.DeepCopy()
	}

	namespace, err := f.CreateTestNamespace()
	if err != nil {
		t.Fatal("failed to create test namespace:", err)
	}

	timeout := 3000 * time.Millisecond

	for _, dnpName := range defaultNetworkPolicyNames {
		if err := f.WaitNetworkPolicyCreated(namespace.GetName(), dnpName, timeout); err != nil {
			t.Fatal("failed to create default network policy("+dnpName+") for new namespace:", err)
		}

		namespaceNetworkPolicy, err := f.KubeClientset.NetworkingV1().NetworkPolicies(namespace.GetName()).Get(dnpName, meta_v1.GetOptions{})
		if err != nil {
			t.Fatal("failed to get default network policy("+dnpName+") for new namespace:", err)
		}

		if !networkPoliciesAreEqual(namespaceNetworkPolicy, defaultNetworkPolicies[dnpName]) {
			t.Fatal("Network policy for created namespace is not equal to the default network policy("+dnpName+"):", err)
		}

		err = f.KubeClientset.NetworkingV1().NetworkPolicies(namespace.GetName()).Delete(dnpName, &meta_v1.DeleteOptions{})
		if err != nil {
			t.Fatal("failed to delete default network policy("+dnpName+") for new namespace:", err)
		}

		if err := f.WaitNetworkPolicyCreated(namespace.GetName(), dnpName, timeout); err != nil {
			t.Fatal("failed to create default network policy("+dnpName+") for new namespace:", err)
		}

		namespaceNetworkPolicy, err = f.KubeClientset.NetworkingV1().NetworkPolicies(namespace.GetName()).Get(dnpName, meta_v1.GetOptions{})
		if err != nil {
			t.Fatal("Reconciler failed to recreate default network policy("+dnpName+") for new namespace:", err)
		}
	}

	//Update single default network policy
	namespaceNetworkPolicy, err := f.KubeClientset.NetworkingV1().NetworkPolicies(namespace.GetName()).Get(defaultNetworkPolicyNames[0], meta_v1.GetOptions{})
	if err != nil {
		t.Fatal("Failed to get default network policy for new namespace:", err)
	}

	namespaceNetworkPolicy.Spec = networkingv1.NetworkPolicySpec{
		PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeIngress}}

	updatedNetworkPolicy, err := f.KubeClientset.NetworkingV1().NetworkPolicies(namespace.GetName()).Update(namespaceNetworkPolicy)
	if err != nil {
		t.Fatal("failed to update default network policy for new namespace:", err)
	}

	duration := 3 * time.Second
	time.Sleep(duration)

	updatedNetworkPolicy, err = f.KubeClientset.NetworkingV1().NetworkPolicies(namespace.GetName()).Get(defaultNetworkPolicyNames[0], meta_v1.GetOptions{})
	if err != nil {
		t.Fatal("failed to get default network policy for new namespace:", err)
	}

	if !networkPoliciesAreEqual(updatedNetworkPolicy, defaultNetworkPolicies[defaultNetworkPolicyNames[0]]) {
		t.Fatal("Reconcilation failed after network policy has changed")
	}
}

func TestCreateMultipleKarydiaNetworkPoliciesForAnnotatedNamespace(t *testing.T) {
        defaultNetworkPolicies := make(map[string]*networkingv1.NetworkPolicy, 3)

        for _, dnpName := range defaultNetworkPolicyNames {
                defaultNetworkPolicies[dnpName] = &networkingv1.NetworkPolicy{}
                defaultNetworkPolicies[dnpName].Name = dnpName

                karydiaNetworkpolicy, err := f.KarydiaClientset.KarydiaV1alpha1().KarydiaNetworkPolicies().Get(dnpName, meta_v1.GetOptions{})
                if err != nil {
                        t.Fatal("failed to get karydia default network policy("+dnpName+"):", err)
                }

                defaultNetworkPolicies[dnpName].Spec = *karydiaNetworkpolicy.Spec.DeepCopy()
        }

	annotations := make(map[string]string)
	annotations["karydia.gardener.cloud/networkPolicy"] = defaultNetworkPolicyNames[1] + ";" + defaultNetworkPolicyNames[2]
	namespace, err := f.CreateTestNamespaceWithAnnotation(annotations)
	if err != nil {
		t.Fatal("failed to create test namespace:", err)
	}

	timeout := 3000 * time.Millisecond

        for _, dnpName := range defaultNetworkPolicyNames[1:2] {
  		if err := f.WaitNetworkPolicyCreated(namespace.GetName(), dnpName, timeout); err != nil {
                        t.Fatal("failed to create default network policy("+dnpName+") for new namespace:", err)
                }

                namespaceNetworkPolicy, err := f.KubeClientset.NetworkingV1().NetworkPolicies(namespace.GetName()).Get(dnpName, meta_v1.GetOptions{})
                if err != nil {
                        t.Fatal("failed to get default network policy("+dnpName+") for new namespace:", err)
                }

                if !networkPoliciesAreEqual(namespaceNetworkPolicy, defaultNetworkPolicies[dnpName]) {
                        t.Fatal("Network policy for created namespace is not equal to the default network policy("+dnpName+"):", err)
                }

                err = f.KubeClientset.NetworkingV1().NetworkPolicies(namespace.GetName()).Delete(dnpName, &meta_v1.DeleteOptions{})
                if err != nil {
                        t.Fatal("failed to delete default network policy("+dnpName+") for new namespace:", err)
                }
	}

	_, err = f.KubeClientset.NetworkingV1().NetworkPolicies(namespace.GetName()).Get(defaultNetworkPolicyNames[0], meta_v1.GetOptions{})

	if err == nil {
		t.Fatal("Default level 0 network policy should not be found")
	}
}

func TestCreateMultipleKarydiaNetworkPoliciesForNamespaceAndUpdateWithAnnotation(t *testing.T) {
        defaultNetworkPolicies := make(map[string]*networkingv1.NetworkPolicy, 3)

        for _, dnpName := range defaultNetworkPolicyNames {
                defaultNetworkPolicies[dnpName] = &networkingv1.NetworkPolicy{}
                defaultNetworkPolicies[dnpName].Name = dnpName

                karydiaNetworkpolicy, err := f.KarydiaClientset.KarydiaV1alpha1().KarydiaNetworkPolicies().Get(dnpName, meta_v1.GetOptions{})
                if err != nil {
                        t.Fatal("failed to get karydia default network policy("+dnpName+"):", err)
                }

                defaultNetworkPolicies[dnpName].Spec = *karydiaNetworkpolicy.Spec.DeepCopy()
        }

	namespace, err := f.CreateTestNamespace()
	if err != nil {
		t.Fatal("failed to create test namespace:", err)
	}

	timeout := 3000 * time.Millisecond

        for _, dnpName := range defaultNetworkPolicyNames[1:2] {
                if err := f.WaitNetworkPolicyCreated(namespace.GetName(), dnpName, timeout); err != nil {
                        t.Fatal("failed to create default network policy("+dnpName+") for new namespace:", err)
                }

	        namespaceNetworkPolicy, err := f.KubeClientset.NetworkingV1().NetworkPolicies(namespace.GetName()).Get(dnpName, meta_v1.GetOptions{})
                if err != nil {
                        t.Fatal("failed to get default network policy("+dnpName+") for new namespace:", err)
                }

                if !networkPoliciesAreEqual(namespaceNetworkPolicy, defaultNetworkPolicies[dnpName]) {
                        t.Fatal("Network policy for created namespace is not equal to the default network policy("+dnpName+"):", err)
                }
	}

	annotations := make(map[string]string)
	annotations["karydia.gardener.cloud/networkPolicy"] = defaultNetworkPolicyNames[0] + ";" + defaultNetworkPolicyNames[1]
	namespace.SetAnnotations(annotations)
	namespace, err = f.KubeClientset.CoreV1().Namespaces().Update(namespace)
	if err != nil {
		t.Fatal("failed to update test namespace:", err)
	}

	for _, dnpName := range defaultNetworkPolicyNames[0:1] {
                if err := f.WaitNetworkPolicyCreated(namespace.GetName(), dnpName, timeout); err != nil {
                        t.Fatal("failed to create default network policy("+dnpName+") for new namespace:", err)
                }
                
                namespaceNetworkPolicy, err := f.KubeClientset.NetworkingV1().NetworkPolicies(namespace.GetName()).Get(dnpName, meta_v1.GetOptions{})
                if err != nil {
                        t.Fatal("failed to get default network policy("+dnpName+") for new namespace:", err)
                }
                        
                if !networkPoliciesAreEqual(namespaceNetworkPolicy, defaultNetworkPolicies[dnpName]) {
                        t.Fatal("Network policy for created namespace is not equal to the default network policy("+dnpName+"):", err)
                }
                
                err = f.KubeClientset.NetworkingV1().NetworkPolicies(namespace.GetName()).Delete(dnpName, &meta_v1.DeleteOptions{})
                if err != nil {
                        t.Fatal("failed to delete default network policy("+dnpName+") for new namespace:", err)
                }
        }


	_, err = f.KubeClientset.NetworkingV1().NetworkPolicies(namespace.GetName()).Get(defaultNetworkPolicyNames[2], meta_v1.GetOptions{})
	if err == nil {
		t.Fatal("Default level 2 network policy should not be found")
	}
}

func TestGetKarydiaMultipleNetworkPoliciesForExcludedNamespace(t *testing.T) {
	for _, dnpName := range defaultNetworkPolicyNames {
		if _, err := f.KubeClientset.NetworkingV1().NetworkPolicies("kube-system").Get(dnpName, meta_v1.GetOptions{}); err == nil {
			t.Fatal("Default network policy should not be found for excluded namespace")
		}
	}
}

func networkPoliciesAreEqual(np1, np2 *networkingv1.NetworkPolicy) bool {
	np1Spec, _ := np1.Spec.Marshal()
	np2Spec, _ := np2.Spec.Marshal()
	return bytes.Equal(np1Spec, np2Spec)
}
