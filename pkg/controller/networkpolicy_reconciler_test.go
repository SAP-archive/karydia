/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"bytes"
	"testing"
	"time"

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

	kubeclient *k8sfake.Clientset

	// Objects to put in the store.
	networkPolicy []*networkingv1.NetworkPolicy
	namespace     []*coreV1.Namespace

	// Objects from here preloaded into NewSimpleFake.
	kubeobjects []runtime.Object

	defaultNetworkPolicy networkingv1.NetworkPolicy

	namespaceExclude []string
}

func newFixture(t *testing.T) *fixture {
	f := &fixture{}
	f.t = t
	f.kubeobjects = []runtime.Object{}
	f.defaultNetworkPolicy = networkingv1.NetworkPolicy{}
	f.defaultNetworkPolicy.Name = "karydia-default-network-policy"
	f.defaultNetworkPolicy.Spec = networkingv1.NetworkPolicySpec{
		PolicyTypes: []networkingv1.PolicyType{},
	}
	f.namespaceExclude = []string{"kube-system", "unittestexclude"}
	return f
}

func (f *fixture) newReconciler() (*NetworkpolicyReconciler, kubeinformers.SharedInformerFactory) {

	f.kubeclient = k8sfake.NewSimpleClientset(f.kubeobjects...)

	k8sI := kubeinformers.NewSharedInformerFactory(f.kubeclient, noResyncPeriodFunc())

	reconciler := NewNetworkpolicyReconciler(f.kubeclient, k8sI.Networking().V1().NetworkPolicies(), k8sI.Core().V1().Namespaces(), &f.defaultNetworkPolicy, f.namespaceExclude)

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
		f.t.Errorf("error syncing networkpolicy: %v", err)
	}
}

func (f *fixture) runNamespaceAdd(namespace string) {

	reconciler, k8sI := f.newReconciler()

	stopCh := make(chan struct{})
	defer close(stopCh)
	k8sI.Start(stopCh)

	err := reconciler.syncNamespaceHandler(namespace)
	if err != nil {
		f.t.Errorf("error syncing foo: %v", err)
	}
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

func TestReconcileNetworkPolicyUpate(t *testing.T) {
	f := newFixture(t)
	newNetworkPolicy := &networkingv1.NetworkPolicy{}
	newNetworkPolicy.Name = "karydia-default-network-policy"
	newNetworkPolicy.Namespace = "default"
	newNetworkPolicy.Spec = networkingv1.NetworkPolicySpec{
		PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeIngress}}

	f.networkPolicy = append(f.networkPolicy, newNetworkPolicy)
	f.kubeobjects = append(f.kubeobjects, newNetworkPolicy)

	f.runReconcile(getKey(newNetworkPolicy, t))

	reconciledPolicy, err := f.kubeclient.NetworkingV1().NetworkPolicies(newNetworkPolicy.Namespace).Get(newNetworkPolicy.Name, meta_v1.GetOptions{})
	if err != nil {
		t.Errorf("No error expected")
	} else if !networkPoliciesAreEqual(&f.defaultNetworkPolicy, reconciledPolicy) {
		t.Errorf("No reconcilation happened")
	}
}

func TestReconcileNetworkPolicyDelete(t *testing.T) {
	f := newFixture(t)
	newNetworkPolicy := &networkingv1.NetworkPolicy{}
	newNetworkPolicy.Name = "karydia-default-network-policy"
	newNetworkPolicy.Namespace = "default"

	f.runReconcile(getKey(newNetworkPolicy, t))

	reconciledPolicy, err := f.kubeclient.NetworkingV1().NetworkPolicies(newNetworkPolicy.Namespace).Get(newNetworkPolicy.Name, meta_v1.GetOptions{})
	if err != nil {
		t.Errorf("No error expected")
	} else if !networkPoliciesAreEqual(&f.defaultNetworkPolicy, reconciledPolicy) {
		t.Errorf("No reconcilation happened")
	}
}

func TestReconcileNetworkPolicyCreateNamespace(t *testing.T) {
	f := newFixture(t)
	newNamespace := &coreV1.Namespace{}
	newNamespace.Name = "unittest"

	f.namespace = append(f.namespace, newNamespace)
	f.kubeobjects = append(f.kubeobjects, newNamespace)

	f.runNamespaceAdd(newNamespace.Name)
	reconciledPolicy, err := f.kubeclient.NetworkingV1().NetworkPolicies(newNamespace.Name).Get(f.defaultNetworkPolicy.Name, meta_v1.GetOptions{})
	if err != nil {
		t.Errorf("No error expected")
	} else if !networkPoliciesAreEqual(&f.defaultNetworkPolicy, reconciledPolicy) {
		t.Errorf("No reconcilation happened")
	}
}

func TestReconcileNetworkPolicyCreateExcludedNamespace(t *testing.T) {
	f := newFixture(t)
	newNamespace := &coreV1.Namespace{}
	newNamespace.Name = "unittestexclude"

	f.namespace = append(f.namespace, newNamespace)
	f.kubeobjects = append(f.kubeobjects, newNamespace)

	f.runNamespaceAdd(newNamespace.Name)
	reconciledPolicy, _ := f.kubeclient.NetworkingV1().NetworkPolicies(newNamespace.Name).Get(f.defaultNetworkPolicy.Name, meta_v1.GetOptions{})
	if reconciledPolicy != nil {
		t.Errorf("Reconcilation happened - default network policy created for excluded namespace")
	}
}
