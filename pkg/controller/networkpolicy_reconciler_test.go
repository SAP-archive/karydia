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

package controller

import (
	"bytes"
	"testing"
	"time"

	"github.com/karydia/karydia/pkg/apis/karydia/v1alpha1"
	"github.com/karydia/karydia/pkg/client/clientset/versioned/fake"
	"github.com/stretchr/testify/assert"
	networkingv1 "k8s.io/api/networking/v1"

	coreV1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kubeinformers "k8s.io/client-go/informers"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"
)

var (
	alwaysReady        = func() bool { return true }
	noResyncPeriodFunc = func() time.Duration { return 0 }
)

type fixture struct {
	t *testing.T

	kubeclient    *k8sfake.Clientset
	karydiaClient *fake.Clientset

	// Objects to put in the store.
	networkPolicy []*networkingv1.NetworkPolicy
	namespace     []*coreV1.Namespace

	// Objects from here preloaded into NewSimpleFake.
	kubeobjects []runtime.Object
	objects     []runtime.Object

	defaultNetworkPolicies map[string]*networkingv1.NetworkPolicy

	namespaceExclude []string
}

func newFixture(t *testing.T) *fixture {
	f := &fixture{}
	f.t = t
	f.kubeobjects = []runtime.Object{}
	f.defaultNetworkPolicies = make(map[string]*networkingv1.NetworkPolicy, 5)

	// ********** Old Network Policy **********
	defaultNetworkPolicy := networkingv1.NetworkPolicy{}
	defaultNetworkPolicy.Name = "karydia-default-network-policy"
	defaultNetworkPolicy.Spec = networkingv1.NetworkPolicySpec{
		PolicyTypes: []networkingv1.PolicyType{},
	}

	f.defaultNetworkPolicies["karydia-default-network-policy"] = &defaultNetworkPolicy

	defaultNetworkPolicy2 := networkingv1.NetworkPolicy{}
	defaultNetworkPolicy2.Name = "karydia-default-network-policy-l2"
	defaultNetworkPolicy2.Spec = networkingv1.NetworkPolicySpec{
		PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeEgress},
	}

	f.defaultNetworkPolicies["karydia-default-network-policy-l2"] = &defaultNetworkPolicy2

	// ********** New Network Policy **********
	defaultNetworkPolicyL1 := networkingv1.NetworkPolicy{}
	defaultNetworkPolicyL1.Name = "network_policy_1"
	defaultNetworkPolicyL1.Spec = networkingv1.NetworkPolicySpec{
		PolicyTypes: []networkingv1.PolicyType{},
	}

	f.defaultNetworkPolicies["network_policy_1"] = &defaultNetworkPolicyL1

	defaultNetworkPolicyL2 := networkingv1.NetworkPolicy{}
	defaultNetworkPolicyL2.Name = "network_policy_2"
	defaultNetworkPolicyL2.Spec = networkingv1.NetworkPolicySpec{
		PolicyTypes: []networkingv1.PolicyType{},
	}

	f.defaultNetworkPolicies["network_policy_2"] = &defaultNetworkPolicyL2

	defaultNetworkPolicyL3 := networkingv1.NetworkPolicy{}
	defaultNetworkPolicyL3.Name = "network_policy_3"

	networkPolicyEgressRule := []networkingv1.NetworkPolicyEgressRule{
		networkingv1.NetworkPolicyEgressRule{
			To: []networkingv1.NetworkPolicyPeer{
				networkingv1.NetworkPolicyPeer{
					NamespaceSelector: &meta_v1.LabelSelector{
						MatchLabels: map[string]string{
							"project": "default",
						},
					},
				},
			},
		},
	}

	defaultNetworkPolicyL3.Spec = networkingv1.NetworkPolicySpec{
		PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeEgress},
		Egress:      networkPolicyEgressRule,
	}

	f.defaultNetworkPolicies["network_policy_3"] = &defaultNetworkPolicyL3

	f.namespaceExclude = []string{"kube-system", "unittestexclude"}
	return f
}

func (f *fixture) newReconciler() (*NetworkpolicyReconciler, kubeinformers.SharedInformerFactory) {

	f.kubeclient = k8sfake.NewSimpleClientset(f.kubeobjects...)
	f.karydiaClient = fake.NewSimpleClientset(f.objects...)

	k8sI := kubeinformers.NewSharedInformerFactory(f.kubeclient, noResyncPeriodFunc())

	reconciler := NewNetworkpolicyReconciler(f.kubeclient, f.karydiaClient, k8sI.Networking().V1().NetworkPolicies(), k8sI.Core().V1().Namespaces(), f.defaultNetworkPolicies, false, "karydia-default-network-policy", f.namespaceExclude)

	reconciler.networkPoliciesSynced = alwaysReady
	reconciler.namespacesSynced = alwaysReady

	for _, d := range f.networkPolicy {
		k8sI.Networking().V1().NetworkPolicies().Informer().GetIndexer().Add(d)
	}

	for _, d := range f.namespace {
		k8sI.Core().V1().Namespaces().Informer().GetIndexer().Add(d)
	}

	return reconciler, k8sI
}

func (f *fixture) runReconcile(networkPolicyName string) {

	reconciler, k8sI := f.newReconciler()
	stopCh := make(chan struct{})
	defer close(stopCh)
	k8sI.Start(stopCh)

	err := reconciler.syncNetworkPolicyHandler(networkPolicyName)
	if err != nil {
		f.t.Error("error syncing networkpolicy:", err)
	}
}

func (f *fixture) runNamespaceAdd(namespace string) {

	reconciler, k8sI := f.newReconciler()

	stopCh := make(chan struct{})
	defer close(stopCh)
	k8sI.Start(stopCh)

	err := reconciler.syncNamespaceHandler(namespace)
	if err != nil {
		f.t.Error("error syncing foo:", err)
	}
}
func (f *fixture) runNamespaceUpdate(namespace string) {

	f.runNamespaceAdd(namespace)
}

func getKey(networkpolicy *networkingv1.NetworkPolicy, t *testing.T) string {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(networkpolicy)
	if err != nil {
		t.Errorf("Unexpected error getting key for foo %v: %v", networkpolicy.Name, err)
		return ""
	}
	return key
}

func networkPoliciesAreEqual(defaultNetworkPolicy, networkPolicy *networkingv1.NetworkPolicy) bool {
	actualSpec, _ := networkPolicy.Spec.Marshal()
	desiredSpec, _ := defaultNetworkPolicy.Spec.Marshal()
	return bytes.Equal(actualSpec, desiredSpec)
}

func TestNetworkpolicyReconciler_UpdateConfig(t *testing.T) {
	assert := assert.New(t)
	f := newFixture(t)
	reconciler, _ := f.newReconciler()
	reconciler.defaultNetworkPolicyName = "default"

	newNetworkpolicyName := "newName"

	newConfig := v1alpha1.KarydiaConfig{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:            "testConfig",
			ResourceVersion: "1",
		},
		Spec: v1alpha1.KarydiaConfigSpec{
			AutomountServiceAccountToken: "testASAT",
			SeccompProfile:               "testSP",
			NetworkPolicy:                newNetworkpolicyName,
		},
	}

	// first check for different config values
	assert.NotEqual(newNetworkpolicyName, reconciler.defaultNetworkPolicyName, "config values shouldn't be equal before updated")
	// try update with right network policy name
	assert.NoError(reconciler.UpdateConfig(newConfig), "config update should succeed because of right network policy name")
	// check for equal config values
	assert.Equal(newNetworkpolicyName, reconciler.defaultNetworkPolicyName, "config values should be equal after succeeded update")

}

func TestReconcileNetworkPolicyUpate(t *testing.T) {
	namespace := &coreV1.Namespace{}
	namespace.Name = "default"

	f := newFixture(t)
	newNetworkPolicy := &networkingv1.NetworkPolicy{}
	newNetworkPolicy.Name = "karydia-default-network-policy"
	newNetworkPolicy.Namespace = "default"
	newNetworkPolicy.Spec = networkingv1.NetworkPolicySpec{
		PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeIngress}}

	f.networkPolicy = append(f.networkPolicy, newNetworkPolicy)
	f.kubeobjects = append(f.kubeobjects, newNetworkPolicy)
	f.kubeobjects = append(f.kubeobjects, namespace)

	f.runReconcile(getKey(newNetworkPolicy, t))

	reconciledPolicy, err := f.kubeclient.NetworkingV1().NetworkPolicies(newNetworkPolicy.Namespace).Get(newNetworkPolicy.Name, meta_v1.GetOptions{})
	if err != nil {
		t.Error("No error expected")
	} else if !networkPoliciesAreEqual(f.defaultNetworkPolicies["karydia-default-network-policy"], reconciledPolicy) {
		t.Error("No reconcilation happened")
	}

}

func TestReconcileNetworkPolicyDelete(t *testing.T) {
	namespace := &coreV1.Namespace{}
	namespace.Name = "default"
	f := newFixture(t)
	assert := assert.New(t)
	newNetworkPolicy := &networkingv1.NetworkPolicy{}
	newNetworkPolicy.Name = "karydia-default-network-policy"
	newNetworkPolicy.Namespace = "default"

	f.kubeobjects = append(f.kubeobjects, namespace)

	f.runReconcile(getKey(newNetworkPolicy, t))

	reconciledPolicy, err := f.kubeclient.NetworkingV1().NetworkPolicies(newNetworkPolicy.Namespace).Get(newNetworkPolicy.Name, meta_v1.GetOptions{})
	if err != nil {
		t.Error("No error expected")
	} else if !networkPoliciesAreEqual(f.defaultNetworkPolicies["karydia-default-network-policy"], reconciledPolicy) {
		t.Error("No reconcilation happened")
	}
	assert.Equal(len(reconciledPolicy.ObjectMeta.Annotations), 1, "network policy should contain internal karydia annotation")
	assert.Contains(reconciledPolicy.ObjectMeta.Annotations["karydia.gardener.cloud/networkPolicy.internal"], "config")
}

func TestReconcileNetworkPolicyWithExisting(t *testing.T) {
	namespace := &coreV1.Namespace{}
	namespace.Name = "default"
	f := newFixture(t)
	newNetworkPolicy := &networkingv1.NetworkPolicy{}
	newNetworkPolicy.Name = "karydia-default-network-policy"
	newNetworkPolicy.Namespace = "default"

	f.kubeobjects = append(f.kubeobjects, namespace)

	f.runReconcile(getKey(newNetworkPolicy, t))

	reconciledPolicy, err := f.kubeclient.NetworkingV1().NetworkPolicies(newNetworkPolicy.Namespace).Get(newNetworkPolicy.Name, meta_v1.GetOptions{})
	if err != nil {
		t.Error("No error expected")
	} else if !networkPoliciesAreEqual(f.defaultNetworkPolicies["karydia-default-network-policy"], reconciledPolicy) {
		t.Error("No reconcilation happened")
	}

	f.defaultNetworkPolicies = make(map[string]*networkingv1.NetworkPolicy, 2)
	f.runReconcile(getKey(newNetworkPolicy, t))
}

func TestReconcileNetworkPolicyCreateNamespace(t *testing.T) {
	f := newFixture(t)
	assert := assert.New(t)
	newNamespace := &coreV1.Namespace{}
	newNamespace.Name = "unittest"

	f.namespace = append(f.namespace, newNamespace)
	f.kubeobjects = append(f.kubeobjects, newNamespace)

	f.runNamespaceAdd(newNamespace.Name)
	reconciledPolicy, err := f.kubeclient.NetworkingV1().NetworkPolicies(newNamespace.Name).Get(f.defaultNetworkPolicies["karydia-default-network-policy"].Name, meta_v1.GetOptions{})
	if err != nil {
		t.Error("No error expected")
	} else if !networkPoliciesAreEqual(f.defaultNetworkPolicies["karydia-default-network-policy"], reconciledPolicy) {
		t.Error("No reconcilation happened")
	}
	assert.Equal(len(reconciledPolicy.ObjectMeta.Annotations), 1, "network policy should contain internal karydia annotation")
	assert.Contains(reconciledPolicy.ObjectMeta.Annotations["karydia.gardener.cloud/networkPolicy.internal"], "config")
}

func TestReconcileNetworkPolicyCreateExcludedNamespace(t *testing.T) {
	f := newFixture(t)
	newNamespace := &coreV1.Namespace{}
	newNamespace.Name = "unittestexclude"

	f.namespace = append(f.namespace, newNamespace)
	f.kubeobjects = append(f.kubeobjects, newNamespace)

	f.runNamespaceAdd(newNamespace.Name)
	reconciledPolicy, _ := f.kubeclient.NetworkingV1().NetworkPolicies(newNamespace.Name).Get(f.defaultNetworkPolicies["karydia-default-network-policy"].Name, meta_v1.GetOptions{})
	if reconciledPolicy != nil {
		t.Error("Reconcilation happened - default network policy created for excluded namespace")
	}
}

func TestReconcileNetworkPolicyCreateNamespaceWithAnnotation(t *testing.T) {
	f := newFixture(t)
	assert := assert.New(t)
	newNamespace := &coreV1.Namespace{}
	newNamespace.Name = "unittest"

	annotations := make(map[string]string)
	annotations["karydia.gardener.cloud/networkPolicy"] = "karydia-default-network-policy-l2"
	newNamespace.ObjectMeta.SetAnnotations(annotations)

	f.namespace = append(f.namespace, newNamespace)
	f.kubeobjects = append(f.kubeobjects, newNamespace)

	f.runNamespaceAdd(newNamespace.Name)

	reconciledPolicy, err := f.kubeclient.NetworkingV1().NetworkPolicies(newNamespace.Name).Get(f.defaultNetworkPolicies["karydia-default-network-policy"].Name, meta_v1.GetOptions{})
	if reconciledPolicy != nil {
		t.Error("Default network policy should not be found")
	}

	reconciledPolicy, err = f.kubeclient.NetworkingV1().NetworkPolicies(newNamespace.Name).Get(f.defaultNetworkPolicies["karydia-default-network-policy-l2"].Name, meta_v1.GetOptions{})
	if err != nil {
		t.Error("No error expected")
	} else if !networkPoliciesAreEqual(f.defaultNetworkPolicies["karydia-default-network-policy-l2"], reconciledPolicy) {
		t.Error("No reconcilation happened")
	}
	assert.Equal(len(reconciledPolicy.ObjectMeta.Annotations), 1, "network policy should contain internal karydia annotation")
	assert.Contains(reconciledPolicy.ObjectMeta.Annotations["karydia.gardener.cloud/networkPolicy.internal"], "namespace")
}

func TestReconcileNetworkPolicyUpdatedNamespace(t *testing.T) {
	f := newFixture(t)
	newNamespace := &coreV1.Namespace{}
	newNamespace.Name = "unittest"

	f.namespace = append(f.namespace, newNamespace)
	f.kubeobjects = append(f.kubeobjects, newNamespace)

	f.runNamespaceAdd(newNamespace.Name)
	reconciledPolicy, err := f.kubeclient.NetworkingV1().NetworkPolicies(newNamespace.Name).Get(f.defaultNetworkPolicies["karydia-default-network-policy"].Name, meta_v1.GetOptions{})
	if err != nil {
		t.Error("No error expected")
	} else if !networkPoliciesAreEqual(f.defaultNetworkPolicies["karydia-default-network-policy"], reconciledPolicy) {
		t.Error("No reconcilation happened")
	}

	annotations := make(map[string]string)
	annotations["karydia.gardener.cloud/networkPolicy"] = "karydia-default-network-policy-l2"
	newNamespace.ObjectMeta.SetAnnotations(annotations)
	f.runNamespaceUpdate(newNamespace.Name)

	reconciledPolicy, err = f.kubeclient.NetworkingV1().NetworkPolicies(newNamespace.Name).Get(f.defaultNetworkPolicies["karydia-default-network-policy"].Name, meta_v1.GetOptions{})
	if reconciledPolicy != nil {
		t.Error("Default network policy should not be found")
	}

	reconciledPolicy, err = f.kubeclient.NetworkingV1().NetworkPolicies(newNamespace.Name).Get(f.defaultNetworkPolicies["karydia-default-network-policy-l2"].Name, meta_v1.GetOptions{})
	if err != nil {
		t.Error("No error expected")
	} else if !networkPoliciesAreEqual(f.defaultNetworkPolicies["karydia-default-network-policy-l2"], reconciledPolicy) {
		t.Error("No reconcilation happened")
	}
}

func TestNetworkPolicyWithNetworkPolicyLevel3ForNamespaceSelector(t *testing.T) {
	f := newFixture(t)
	newNamespace := &coreV1.Namespace{}
	newNamespace.Name = "unittestlevel3"

	annotations := make(map[string]string)
	annotations["karydia.gardener.cloud/networkPolicy"] = "network_policy_3"
	newNamespace.ObjectMeta.SetAnnotations(annotations)

	f.namespace = append(f.namespace, newNamespace)
	f.kubeobjects = append(f.kubeobjects, newNamespace)

	f.runNamespaceAdd(newNamespace.Name)

	reconciledPolicy, err := f.kubeclient.NetworkingV1().NetworkPolicies(newNamespace.Name).Get(f.defaultNetworkPolicies["network_policy_3"].Name, meta_v1.GetOptions{})
	if err != nil {
		t.Errorf("No error expected")
	} else if _, ok := reconciledPolicy.Spec.Egress[0].To[0].NamespaceSelector.MatchLabels["project"]; !ok {
		t.Errorf("No default Namespace Selector")
	} else if value, _ := reconciledPolicy.Spec.Egress[0].To[0].NamespaceSelector.MatchLabels["project"]; value != newNamespace.Name {
		t.Errorf("Namespace is not equals default")
	}

	f.networkPolicy = append(f.networkPolicy, reconciledPolicy)
	f.kubeobjects = append(f.kubeobjects, reconciledPolicy)

	f.runReconcile(getKey(reconciledPolicy, t))

	reconciledPolicy2, err := f.kubeclient.NetworkingV1().NetworkPolicies(reconciledPolicy.Namespace).Get(reconciledPolicy.Name, meta_v1.GetOptions{})
	if err != nil {
		t.Errorf("No error expected")
	} else if value, _ := reconciledPolicy2.Spec.Egress[0].To[0].NamespaceSelector.MatchLabels["project"]; value != newNamespace.Name {
		t.Errorf("No reconcilation happened")
	}
}
