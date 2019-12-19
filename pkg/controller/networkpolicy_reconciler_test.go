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
	alwaysReady               = func() bool { return true }
	noResyncPeriodFunc        = func() time.Duration { return 0 }
	defaultNetworkPolicyNames = [3]string{"karydia-default-network-policy-l1", "karydia-default-network-policy-l2", "karydia-default-network-policy-l3"}
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
	f.objects = []runtime.Object{}
	f.defaultNetworkPolicies = make(map[string]*networkingv1.NetworkPolicy, 3)

	defaultNetworkPolicy := networkingv1.NetworkPolicy{}
	defaultNetworkPolicy.Name = defaultNetworkPolicyNames[0]
	defaultNetworkPolicy.Spec = networkingv1.NetworkPolicySpec{
		PolicyTypes: []networkingv1.PolicyType{},
	}

	f.defaultNetworkPolicies[defaultNetworkPolicyNames[0]] = &defaultNetworkPolicy

	defaultNetworkPolicyL2 := networkingv1.NetworkPolicy{}
	defaultNetworkPolicyL2.Name = defaultNetworkPolicyNames[1]
	defaultNetworkPolicyL2.Spec = networkingv1.NetworkPolicySpec{
		PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeEgress},
	}

	f.defaultNetworkPolicies[defaultNetworkPolicyNames[1]] = &defaultNetworkPolicyL2

	defaultNetworkPolicyL3 := networkingv1.NetworkPolicy{}
	defaultNetworkPolicyL3.Name = defaultNetworkPolicyNames[2]
	defaultNetworkPolicyL3.Spec = networkingv1.NetworkPolicySpec{
		PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeIngress},
	}

	f.defaultNetworkPolicies[defaultNetworkPolicyNames[2]] = &defaultNetworkPolicyL3

	f.namespaceExclude = []string{"kube-system", "unittestexclude"}
	return f
}

func (f *fixture) newReconciler(defaultNetworkPolicyName string) (*NetworkpolicyReconciler, kubeinformers.SharedInformerFactory) {

	f.kubeclient = k8sfake.NewSimpleClientset(f.kubeobjects...)
	f.karydiaClient = fake.NewSimpleClientset(f.objects...)

	k8sI := kubeinformers.NewSharedInformerFactory(f.kubeclient, noResyncPeriodFunc())

	reconciler := NewNetworkpolicyReconciler(f.kubeclient, f.karydiaClient, k8sI.Networking().V1().NetworkPolicies(), k8sI.Core().V1().Namespaces(), f.defaultNetworkPolicies, false, defaultNetworkPolicyName, f.namespaceExclude)

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

func (f *fixture) runReconcile(defaultNetworkPolicyName string, networkPolicyName string) {

	reconciler, k8sI := f.newReconciler(defaultNetworkPolicyName)
	stopCh := make(chan struct{})
	defer close(stopCh)
	k8sI.Start(stopCh)

	err := reconciler.syncNetworkPolicyHandler(networkPolicyName)
	if err != nil {
		f.t.Error("error syncing networkpolicy:", err)
	}
}

func (f *fixture) runNamespaceAdd(defaultNetworkPolicyName string, namespace string) {

	reconciler, k8sI := f.newReconciler(defaultNetworkPolicyName)

	stopCh := make(chan struct{})
	defer close(stopCh)
	k8sI.Start(stopCh)

	err := reconciler.syncNamespaceHandler(namespace)
	if err != nil {
		f.t.Error("error syncing foo:", err)
	}
}

func (f *fixture) runNamespaceUpdate(defaultNetworkPolicyName string, namespace string) {

	f.runNamespaceAdd(defaultNetworkPolicyName, namespace)
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

func networkpolicyReconciler_UpdateConfig(t *testing.T) {
	assert := assert.New(t)
	f := newFixture(t)
	defaultNetworkPolicyName := "default"
	reconciler, _ := f.newReconciler(defaultNetworkPolicyName)

	newNetworkpolicyName := "newName"

	newConfig := v1alpha1.KarydiaConfig{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:            "testConfig",
			ResourceVersion: "1",
		},
		Spec: v1alpha1.KarydiaConfigSpec{
			AutomountServiceAccountToken: "testASAT",
			SeccompProfile:               "testSP",
			NetworkPolicies:              newNetworkpolicyName,
		},
	}

	// first check for different config values
	assert.NotEqual(newNetworkpolicyName, reconciler.defaultNetworkPolicyNames, "config values shouldn't be equal before updated")
	// try update with right network policy name
	assert.NoError(reconciler.UpdateConfig(newConfig), "config update should succeed because of right network policy name")
	// check for equal config values
	assert.Equal(newNetworkpolicyName, reconciler.defaultNetworkPolicyNames, "config values should be equal after succeeded update")

}

func TestReconcileNetworkPolicyUpdate(t *testing.T) {
	namespace := &coreV1.Namespace{}
	namespace.Name = "default"

	f := newFixture(t)
	newNetworkPolicy := &networkingv1.NetworkPolicy{}
	newNetworkPolicy.Name = defaultNetworkPolicyNames[0]
	newNetworkPolicy.Namespace = "default"
	newNetworkPolicy.Spec = networkingv1.NetworkPolicySpec{
		PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeIngress}}

	f.networkPolicy = append(f.networkPolicy, newNetworkPolicy)
	f.kubeobjects = append(f.kubeobjects, newNetworkPolicy)
	f.kubeobjects = append(f.kubeobjects, namespace)

	f.runReconcile(defaultNetworkPolicyNames[0], getKey(newNetworkPolicy, t))

	reconciledPolicy, err := f.kubeclient.NetworkingV1().NetworkPolicies(newNetworkPolicy.Namespace).Get(newNetworkPolicy.Name, meta_v1.GetOptions{})

	if err != nil {
		t.Error("No error expected")
	} else if !networkPoliciesAreEqual(f.defaultNetworkPolicies[defaultNetworkPolicyNames[0]], reconciledPolicy) {
		t.Error("No reconcilation happened")
	}
}

func TestReconcileMultipleNetworkPoliciesUpdate(t *testing.T) {
	dnpNames := defaultNetworkPolicyNames[0] + ";" + defaultNetworkPolicyNames[1] + ";" + defaultNetworkPolicyNames[2]

	f := newFixture(t)

	// reset "default" namespace
	namespace := &coreV1.Namespace{}
	namespace.Name = "default"

	f.namespace = append(f.namespace, namespace)
	f.kubeobjects = append(f.kubeobjects, namespace)

	reconciler, k8sI := f.newReconciler(dnpNames)
	stopCh := make(chan struct{})
	defer close(stopCh)
	k8sI.Start(stopCh)

	err := reconciler.syncNamespaceHandler(namespace.Name)
	if err != nil {
		f.t.Error("error syncing foo:", err)
	}

	// update single network policy
	newNetworkPolicy := &networkingv1.NetworkPolicy{}
	newNetworkPolicy.Name = defaultNetworkPolicyNames[0]
	newNetworkPolicy.Namespace = "default"
	newNetworkPolicy.Spec = networkingv1.NetworkPolicySpec{
		PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeIngress}}

	f.networkPolicy = append(f.networkPolicy, newNetworkPolicy)
	f.kubeobjects = append(f.kubeobjects, newNetworkPolicy)
	f.kubeobjects = append(f.kubeobjects, namespace)

	err = reconciler.syncNetworkPolicyHandler(getKey(newNetworkPolicy, t))
	if err != nil {
		f.t.Error("error syncing networkpolicy:", err)
	}

	// check if single network policy changed and the others stayed unchanged
	reconciledPolicy, err := f.kubeclient.NetworkingV1().NetworkPolicies(newNetworkPolicy.Namespace).Get(newNetworkPolicy.Name, meta_v1.GetOptions{})

	if err != nil {
		t.Error("No error expected", err)
	} else if !networkPoliciesAreEqual(f.defaultNetworkPolicies[defaultNetworkPolicyNames[0]], reconciledPolicy) {
		t.Error("No reconcilation happened")
	}

	for i := 1; i <= 2; i++ {
		currentPolicy, err := f.kubeclient.NetworkingV1().NetworkPolicies(newNetworkPolicy.Namespace).Get(defaultNetworkPolicyNames[i], meta_v1.GetOptions{})
		if err != nil {
			t.Error("No error expected (for current policies). Error: ", err)
		} else if !networkPoliciesAreEqual(f.defaultNetworkPolicies[defaultNetworkPolicyNames[i]], currentPolicy) {
			t.Error("Other network policies changed that shouldn't")
		}
	}
}

func TestReconcileNetworkPolicyDelete(t *testing.T) {
	namespace := &coreV1.Namespace{}
	namespace.Name = "default"
	f := newFixture(t)
	assert := assert.New(t)
	newNetworkPolicy := &networkingv1.NetworkPolicy{}
	newNetworkPolicy.Name = defaultNetworkPolicyNames[0]
	newNetworkPolicy.Namespace = "default"

	f.kubeobjects = append(f.kubeobjects, namespace)

	f.runReconcile(defaultNetworkPolicyNames[0], getKey(newNetworkPolicy, t))

	reconciledPolicy, err := f.kubeclient.NetworkingV1().NetworkPolicies(newNetworkPolicy.Namespace).Get(newNetworkPolicy.Name, meta_v1.GetOptions{})
	if err != nil {
		t.Error("No error expected")
	} else if !networkPoliciesAreEqual(f.defaultNetworkPolicies[defaultNetworkPolicyNames[0]], reconciledPolicy) {
		t.Error("No reconcilation happened")
	}
	assert.Equal(len(reconciledPolicy.ObjectMeta.Annotations), 1, "network policy should contain internal karydia annotation")
	assert.Contains(reconciledPolicy.ObjectMeta.Annotations["karydia.gardener.cloud/networkPolicy.internal"], "config")
}

func TestReconcileMultipleNetworkPoliciesDelete(t *testing.T) {
	dnpNames := defaultNetworkPolicyNames[0] + ";" + defaultNetworkPolicyNames[1] + ";" + defaultNetworkPolicyNames[2]

	f := newFixture(t)
	assert := assert.New(t)

	// reset "default" namespace
	namespace := &coreV1.Namespace{}
	namespace.Name = "default"

	f.namespace = append(f.namespace, namespace)
	f.kubeobjects = append(f.kubeobjects, namespace)

	reconciler, k8sI := f.newReconciler(dnpNames)
	stopCh := make(chan struct{})
	defer close(stopCh)
	k8sI.Start(stopCh)

	err := reconciler.syncNamespaceHandler(namespace.Name)
	if err != nil {
		f.t.Error("error syncing foo:", err)
	}

	// delete single network policy
	newNetworkPolicy := &networkingv1.NetworkPolicy{}
	newNetworkPolicy.Name = defaultNetworkPolicyNames[0]
	newNetworkPolicy.Namespace = "default"

	f.kubeobjects = append(f.kubeobjects, namespace)

	err = reconciler.syncNetworkPolicyHandler(getKey(newNetworkPolicy, t))
	if err != nil {
		f.t.Error("error syncing networkpolicy:", err)
	}

	// check if single network policy was deleted and the others stayed unchanged
	reconciledPolicy, err := f.kubeclient.NetworkingV1().NetworkPolicies(newNetworkPolicy.Namespace).Get(newNetworkPolicy.Name, meta_v1.GetOptions{})
	if err != nil {
		t.Error("No error expected")
	} else if !networkPoliciesAreEqual(f.defaultNetworkPolicies[defaultNetworkPolicyNames[0]], reconciledPolicy) {
		t.Error("No reconcilation happened")
	}

	assert.Equal(len(reconciledPolicy.ObjectMeta.Annotations), 1, "network policy should contain internal karydia annotation")
	assert.Contains(reconciledPolicy.ObjectMeta.Annotations["karydia.gardener.cloud/networkPolicy.internal"], "config")

	for i := 1; i <= 2; i++ {
		currentPolicy, err := f.kubeclient.NetworkingV1().NetworkPolicies(newNetworkPolicy.Namespace).Get(defaultNetworkPolicyNames[i], meta_v1.GetOptions{})
		if err != nil {
			t.Error("No error expected", err)
		} else if !networkPoliciesAreEqual(f.defaultNetworkPolicies[defaultNetworkPolicyNames[i]], currentPolicy) {
			t.Error("Other network policies changed that shouldn't")
		}
	}
}

func TestReconcileNetworkPolicyWithExisting(t *testing.T) {
	namespace := &coreV1.Namespace{}
	namespace.Name = "default"
	f := newFixture(t)
	newNetworkPolicy := &networkingv1.NetworkPolicy{}
	newNetworkPolicy.Name = defaultNetworkPolicyNames[0]
	newNetworkPolicy.Namespace = "default"

	f.kubeobjects = append(f.kubeobjects, namespace)

	f.runReconcile(defaultNetworkPolicyNames[0], getKey(newNetworkPolicy, t))

	reconciledPolicy, err := f.kubeclient.NetworkingV1().NetworkPolicies(newNetworkPolicy.Namespace).Get(newNetworkPolicy.Name, meta_v1.GetOptions{})
	if err != nil {
		t.Error("No error expected")
	} else if !networkPoliciesAreEqual(f.defaultNetworkPolicies[defaultNetworkPolicyNames[0]], reconciledPolicy) {
		t.Error("No reconcilation happened")
	}

	f.defaultNetworkPolicies = make(map[string]*networkingv1.NetworkPolicy, 3)
	f.runReconcile(defaultNetworkPolicyNames[0], getKey(newNetworkPolicy, t))
}

func TestReconcileMultipleNetworkPoliciesWithExisting(t *testing.T) {
	dnpNames := defaultNetworkPolicyNames[0] + ";" + defaultNetworkPolicyNames[1] + ";" + defaultNetworkPolicyNames[2]

	f := newFixture(t)

	// reset "default" namespace
	namespace := &coreV1.Namespace{}
	namespace.Name = "default"

	f.namespace = append(f.namespace, namespace)
	f.kubeobjects = append(f.kubeobjects, namespace)

	reconciler, k8sI := f.newReconciler(dnpNames)
	stopCh := make(chan struct{})
	defer close(stopCh)
	k8sI.Start(stopCh)

	err := reconciler.syncNamespaceHandler(namespace.Name)
	if err != nil {
		f.t.Error("error syncing foo:", err)
	}

	// delete single network policy
	newNetworkPolicy := &networkingv1.NetworkPolicy{}
	newNetworkPolicy.Name = defaultNetworkPolicyNames[0]
	newNetworkPolicy.Namespace = "default"

	f.kubeobjects = append(f.kubeobjects, namespace)

	err = reconciler.syncNetworkPolicyHandler(getKey(newNetworkPolicy, t))
	if err != nil {
		f.t.Error("error syncing networkpolicy:", err)
	}

	// check if single network policy changed and the others stayed unchanged
	reconciledPolicy, err := f.kubeclient.NetworkingV1().NetworkPolicies(newNetworkPolicy.Namespace).Get(newNetworkPolicy.Name, meta_v1.GetOptions{})
	if err != nil {
		t.Error("No error expected")
	} else if !networkPoliciesAreEqual(f.defaultNetworkPolicies[defaultNetworkPolicyNames[0]], reconciledPolicy) {
		t.Error("No reconcilation happened")
	}

	for i := 1; i <= 2; i++ {
		currentPolicy, err := f.kubeclient.NetworkingV1().NetworkPolicies(newNetworkPolicy.Namespace).Get(defaultNetworkPolicyNames[i], meta_v1.GetOptions{})
		if err != nil {
			t.Error("No error expected")
		} else if !networkPoliciesAreEqual(f.defaultNetworkPolicies[defaultNetworkPolicyNames[i]], currentPolicy) {
			t.Error("Other network policies changed that shouldn't")
		}
	}

	f.defaultNetworkPolicies = make(map[string]*networkingv1.NetworkPolicy, 3)

	err = reconciler.syncNetworkPolicyHandler(getKey(newNetworkPolicy, t))
	if err != nil {
		f.t.Error("error syncing networkpolicy:", err)
	}
}

func TestReconcileNetworkPolicyCreateNamespace(t *testing.T) {
	f := newFixture(t)
	assert := assert.New(t)
	newNamespace := &coreV1.Namespace{}
	newNamespace.Name = "unittest"

	f.namespace = append(f.namespace, newNamespace)
	f.kubeobjects = append(f.kubeobjects, newNamespace)

	f.runNamespaceAdd(defaultNetworkPolicyNames[0], newNamespace.Name)
	reconciledPolicy, err := f.kubeclient.NetworkingV1().NetworkPolicies(newNamespace.Name).Get(f.defaultNetworkPolicies[defaultNetworkPolicyNames[0]].Name, meta_v1.GetOptions{})
	if err != nil {
		t.Error("No error expected")
	} else if !networkPoliciesAreEqual(f.defaultNetworkPolicies[defaultNetworkPolicyNames[0]], reconciledPolicy) {
		t.Error("No reconcilation happened")
	}
	assert.Equal(len(reconciledPolicy.ObjectMeta.Annotations), 1, "network policy should contain internal karydia annotation")
	assert.Contains(reconciledPolicy.ObjectMeta.Annotations["karydia.gardener.cloud/networkPolicy.internal"], "config")
}

func TestReconcileMultipleNetworkPoliciesCreateNamespace(t *testing.T) {
	dnpNames := defaultNetworkPolicyNames[0] + ";" + defaultNetworkPolicyNames[1] + ";" + defaultNetworkPolicyNames[2]

	f := newFixture(t)
	assert := assert.New(t)
	newNamespace := &coreV1.Namespace{}
	newNamespace.Name = "unittest"

	f.namespace = append(f.namespace, newNamespace)
	f.kubeobjects = append(f.kubeobjects, newNamespace)

	f.runNamespaceAdd(dnpNames, newNamespace.Name)

	for i := 0; i <= 2; i++ {
		reconciledPolicy, err := f.kubeclient.NetworkingV1().NetworkPolicies(newNamespace.Name).Get(f.defaultNetworkPolicies[defaultNetworkPolicyNames[i]].Name, meta_v1.GetOptions{})
		if err != nil {
			t.Error("No error expected")
		} else if !networkPoliciesAreEqual(f.defaultNetworkPolicies[defaultNetworkPolicyNames[i]], reconciledPolicy) {
			t.Error("No reconcilation happened")
		}
		assert.Equal(len(reconciledPolicy.ObjectMeta.Annotations), 1, "network policy should contain internal karydia annotation")
		assert.Contains(reconciledPolicy.ObjectMeta.Annotations["karydia.gardener.cloud/networkPolicy.internal"], "config")
	}
}

func TestReconcileNetworkPolicyCreateExcludedNamespace(t *testing.T) {
	f := newFixture(t)
	newNamespace := &coreV1.Namespace{}
	newNamespace.Name = "unittestexclude"

	f.namespace = append(f.namespace, newNamespace)
	f.kubeobjects = append(f.kubeobjects, newNamespace)

	f.runNamespaceAdd(defaultNetworkPolicyNames[0], newNamespace.Name)
	reconciledPolicy, _ := f.kubeclient.NetworkingV1().NetworkPolicies(newNamespace.Name).Get(f.defaultNetworkPolicies[defaultNetworkPolicyNames[0]].Name, meta_v1.GetOptions{})
	if reconciledPolicy != nil {
		t.Error("Reconcilation happened - default network policy created for excluded namespace")
	}
}

func TestReconcileMultipleNetworkPoliciesCreateExcludedNamespace(t *testing.T) {
	dnpNames := defaultNetworkPolicyNames[0] + ";" + defaultNetworkPolicyNames[1] + ";" + defaultNetworkPolicyNames[2]

	f := newFixture(t)
	newNamespace := &coreV1.Namespace{}
	newNamespace.Name = "unittestexclude"

	f.namespace = append(f.namespace, newNamespace)
	f.kubeobjects = append(f.kubeobjects, newNamespace)

	f.runNamespaceAdd(dnpNames, newNamespace.Name)

	for i := 0; i <= 2; i++ {
		reconciledPolicy, _ := f.kubeclient.NetworkingV1().NetworkPolicies(newNamespace.Name).Get(f.defaultNetworkPolicies[defaultNetworkPolicyNames[0]].Name, meta_v1.GetOptions{})
		if reconciledPolicy != nil {
			t.Error("Reconcilation happened - default network policy created for excluded namespace")
		}
	}
}

func TestReconcileNetworkPolicyCreateNamespaceWithAnnotation(t *testing.T) {
	f := newFixture(t)
	assert := assert.New(t)
	newNamespace := &coreV1.Namespace{}
	newNamespace.Name = "unittest"

	annotations := make(map[string]string)
	annotations["karydia.gardener.cloud/networkPolicy"] = defaultNetworkPolicyNames[1]
	newNamespace.ObjectMeta.SetAnnotations(annotations)

	f.namespace = append(f.namespace, newNamespace)
	f.kubeobjects = append(f.kubeobjects, newNamespace)

	f.runNamespaceAdd(defaultNetworkPolicyNames[0], newNamespace.Name)

	reconciledPolicy, err := f.kubeclient.NetworkingV1().NetworkPolicies(newNamespace.Name).Get(f.defaultNetworkPolicies[defaultNetworkPolicyNames[0]].Name, meta_v1.GetOptions{})
	if reconciledPolicy != nil {
		t.Error("Default network policy should not be found")
	}

	reconciledPolicy, err = f.kubeclient.NetworkingV1().NetworkPolicies(newNamespace.Name).Get(f.defaultNetworkPolicies[defaultNetworkPolicyNames[1]].Name, meta_v1.GetOptions{})
	if err != nil {
		t.Error("No error expected")
	} else if !networkPoliciesAreEqual(f.defaultNetworkPolicies[defaultNetworkPolicyNames[1]], reconciledPolicy) {
		t.Error("No reconcilation happened")
	}
	assert.Equal(len(reconciledPolicy.ObjectMeta.Annotations), 1, "network policy should contain internal karydia annotation")
	assert.Contains(reconciledPolicy.ObjectMeta.Annotations["karydia.gardener.cloud/networkPolicy.internal"], "namespace")
}

func TestReconcileMultipleNetworkPoliciesCreateNamespaceWithAnnotation(t *testing.T) {
	dnpNames := defaultNetworkPolicyNames[0] + ";" + defaultNetworkPolicyNames[1] + ";" + defaultNetworkPolicyNames[2]

	f := newFixture(t)
	assert := assert.New(t)
	newNamespace := &coreV1.Namespace{}
	newNamespace.Name = "unittest"

	annotations := make(map[string]string)
	annotations["karydia.gardener.cloud/networkPolicy"] = defaultNetworkPolicyNames[2]
	newNamespace.ObjectMeta.SetAnnotations(annotations)

	f.namespace = append(f.namespace, newNamespace)
	f.kubeobjects = append(f.kubeobjects, newNamespace)

	f.runNamespaceAdd(dnpNames, newNamespace.Name)

	for i := 0; i <= 1; i++ {
		reconciledPolicy, _ := f.kubeclient.NetworkingV1().NetworkPolicies(newNamespace.Name).Get(f.defaultNetworkPolicies[defaultNetworkPolicyNames[i]].Name, meta_v1.GetOptions{})
		if reconciledPolicy != nil {
			t.Error("Default network policy should not be found")
		}
	}

	reconciledPolicy, err := f.kubeclient.NetworkingV1().NetworkPolicies(newNamespace.Name).Get(f.defaultNetworkPolicies[defaultNetworkPolicyNames[2]].Name, meta_v1.GetOptions{})
	if err != nil {
		t.Error("No error expected")
	} else if !networkPoliciesAreEqual(f.defaultNetworkPolicies[defaultNetworkPolicyNames[2]], reconciledPolicy) {
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

	f.runNamespaceAdd(defaultNetworkPolicyNames[0], newNamespace.Name)
	reconciledPolicy, err := f.kubeclient.NetworkingV1().NetworkPolicies(newNamespace.Name).Get(f.defaultNetworkPolicies[defaultNetworkPolicyNames[0]].Name, meta_v1.GetOptions{})
	if err != nil {
		t.Error("No error expected")
	} else if !networkPoliciesAreEqual(f.defaultNetworkPolicies[defaultNetworkPolicyNames[0]], reconciledPolicy) {
		t.Error("No reconcilation happened")
	}

	annotations := make(map[string]string)
	annotations["karydia.gardener.cloud/networkPolicy"] = defaultNetworkPolicyNames[1]
	newNamespace.ObjectMeta.SetAnnotations(annotations)
	f.runNamespaceUpdate(defaultNetworkPolicyNames[0], newNamespace.Name)

	reconciledPolicy, err = f.kubeclient.NetworkingV1().NetworkPolicies(newNamespace.Name).Get(f.defaultNetworkPolicies[defaultNetworkPolicyNames[0]].Name, meta_v1.GetOptions{})
	if reconciledPolicy != nil {
		t.Error("Default network policy should not be found")
	}

	reconciledPolicy, err = f.kubeclient.NetworkingV1().NetworkPolicies(newNamespace.Name).Get(f.defaultNetworkPolicies[defaultNetworkPolicyNames[1]].Name, meta_v1.GetOptions{})
	if err != nil {
		t.Error("No error expected")
	} else if !networkPoliciesAreEqual(f.defaultNetworkPolicies[defaultNetworkPolicyNames[1]], reconciledPolicy) {
		t.Error("No reconcilation happened")
	}
}

func TestReconcileMultipleNetworkPoliciesUpdatedNamespace(t *testing.T) {
	dnpNames := defaultNetworkPolicyNames[0] + ";" + defaultNetworkPolicyNames[1] + ";" + defaultNetworkPolicyNames[2]

	f := newFixture(t)
	newNamespace := &coreV1.Namespace{}
	newNamespace.Name = "unittest"

	f.namespace = append(f.namespace, newNamespace)
	f.kubeobjects = append(f.kubeobjects, newNamespace)

	f.runNamespaceAdd(dnpNames, newNamespace.Name)

	for i := 0; i <= 2; i++ {
		reconciledPolicy, err := f.kubeclient.NetworkingV1().NetworkPolicies(newNamespace.Name).Get(f.defaultNetworkPolicies[defaultNetworkPolicyNames[i]].Name, meta_v1.GetOptions{})
		if err != nil {
			t.Error("No error expected")
		} else if !networkPoliciesAreEqual(f.defaultNetworkPolicies[defaultNetworkPolicyNames[i]], reconciledPolicy) {
			t.Error("No reconcilation happened")
		}
	}

	annotations := make(map[string]string)
	annotations["karydia.gardener.cloud/networkPolicy"] = defaultNetworkPolicyNames[2]
	newNamespace.ObjectMeta.SetAnnotations(annotations)
	f.runNamespaceUpdate(dnpNames, newNamespace.Name)

	for i := 0; i <= 1; i++ {
		reconciledPolicy, _ := f.kubeclient.NetworkingV1().NetworkPolicies(newNamespace.Name).Get(f.defaultNetworkPolicies[defaultNetworkPolicyNames[i]].Name, meta_v1.GetOptions{})
		if reconciledPolicy != nil {
			t.Error("Default network policy should not be found")
		}
	}

	reconciledPolicy, err := f.kubeclient.NetworkingV1().NetworkPolicies(newNamespace.Name).Get(f.defaultNetworkPolicies[defaultNetworkPolicyNames[2]].Name, meta_v1.GetOptions{})
	if err != nil {
		t.Error("No error expected")
	} else if !networkPoliciesAreEqual(f.defaultNetworkPolicies[defaultNetworkPolicyNames[2]], reconciledPolicy) {
		t.Error("No reconcilation happened")
	}
}
