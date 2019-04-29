// Copyright 2019 Copyright (c) 2019 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file.
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
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	v1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	namespaceInformer "k8s.io/client-go/informers/core/v1"
	networkpolicyInformer "k8s.io/client-go/informers/networking/v1"
	"k8s.io/client-go/kubernetes"
	kubelistersv1 "k8s.io/client-go/listers/core/v1"
	kubelistersNetworkingv1 "k8s.io/client-go/listers/networking/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog"
)

const controllerAgentName = "networkpolicy_reconciler"

type NetworkpolicyReconciler struct {
	defaultNetworkPolicyName     string
	defaultNetworkPolicies       map[string]*networkingv1.NetworkPolicy
	defaultNetworkPolicyExcludes []string

	kubeclientset          kubernetes.Interface
	networkPolicyLister    kubelistersNetworkingv1.NetworkPolicyLister
	networkPoliciesSynced  cache.InformerSynced
	namespacesLister       kubelistersv1.NamespaceLister
	namespacesSynced       cache.InformerSynced
	networkPolicyworkqueue workqueue.RateLimitingInterface
	namespaceWorkqueue     workqueue.RateLimitingInterface
}

func NewNetworkpolicyReconciler(
	kubeclientset kubernetes.Interface,
	networkpolicyInformer networkpolicyInformer.NetworkPolicyInformer, namespaceInformer namespaceInformer.NamespaceInformer,
	defaultNetworkPolicies map[string]*networkingv1.NetworkPolicy, defaultNetworkPolicyName string, defaultNetworkPolicyExcludes []string) *NetworkpolicyReconciler {

	reconciler := &NetworkpolicyReconciler{
		defaultNetworkPolicyName:     defaultNetworkPolicyName,
		kubeclientset:                kubeclientset,
		networkPolicyLister:          networkpolicyInformer.Lister(),
		networkPoliciesSynced:        networkpolicyInformer.Informer().HasSynced,
		namespacesLister:             namespaceInformer.Lister(),
		namespacesSynced:             namespaceInformer.Informer().HasSynced,
		namespaceWorkqueue:           workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "Namespaces"),
		networkPolicyworkqueue:       workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "Networkpolicies"),
		defaultNetworkPolicies:       defaultNetworkPolicies,
		defaultNetworkPolicyExcludes: defaultNetworkPolicyExcludes,
	}

	klog.Info("Setting up event handlers")
	networkpolicyInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: func(old, new interface{}) {
			newNetworkPolicy := new.(*networkingv1.NetworkPolicy)
			oldNetworkPolicy := old.(*networkingv1.NetworkPolicy)
			if newNetworkPolicy.ResourceVersion == oldNetworkPolicy.ResourceVersion {
				return
			}
			reconciler.enqueueNetworkPolicy(new)
		},
		DeleteFunc: reconciler.enqueueNetworkPolicy,
	})

	namespaceInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: reconciler.enqueueNamespace,
	})

	return reconciler
}

func (reconciler *NetworkpolicyReconciler) Run(threadiness int, stopCh <-chan struct{}) error {
	defer utilruntime.HandleCrash()
	defer reconciler.networkPolicyworkqueue.ShutDown()
	defer reconciler.namespaceWorkqueue.ShutDown()

	klog.Info("Starting karydia network policy reconciler")
	klog.Info("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(stopCh, reconciler.networkPoliciesSynced, reconciler.namespacesSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	klog.Info("Starting workers")

	for i := 0; i < threadiness; i++ {
		go wait.Until(reconciler.runNetworkPolicyWorker, time.Second, stopCh)
		go wait.Until(reconciler.runNamespaceWorker, time.Second, stopCh)
	}

	klog.Info("Started workers")
	<-stopCh
	klog.Info("Shutting down workers")

	return nil
}

func (reconciler *NetworkpolicyReconciler) runNetworkPolicyWorker() {
	for reconciler.processNextNetworkPolicyWorkItem() {
	}
}
func (reconciler *NetworkpolicyReconciler) runNamespaceWorker() {
	for reconciler.processNextNamespaceWorkItem() {
	}
}

func (reconciler *NetworkpolicyReconciler) processNextNetworkPolicyWorkItem() bool {
	obj, shutdown := reconciler.networkPolicyworkqueue.Get()

	if shutdown {
		return false
	}

	err := func(obj interface{}) error {
		defer reconciler.networkPolicyworkqueue.Done(obj)
		var key string
		var ok bool

		if key, ok = obj.(string); !ok {
			reconciler.networkPolicyworkqueue.Forget(obj)
			utilruntime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", obj))
			return nil
		}

		if err := reconciler.syncNetworkPolicyHandler(key); err != nil {

			reconciler.networkPolicyworkqueue.AddRateLimited(key)
			return fmt.Errorf("error syncing '%s': %s, requeuing", key, err.Error())
		}

		reconciler.networkPolicyworkqueue.Forget(obj)
		klog.Infof("Successfully synced '%s'", key)
		return nil
	}(obj)

	if err != nil {
		utilruntime.HandleError(err)
		return true
	}

	return true
}

func (reconciler *NetworkpolicyReconciler) processNextNamespaceWorkItem() bool {
	obj, shutdown := reconciler.namespaceWorkqueue.Get()

	if shutdown {
		return false
	}

	err := func(obj interface{}) error {
		defer reconciler.namespaceWorkqueue.Done(obj)
		var key string
		var ok bool

		if key, ok = obj.(string); !ok {

			reconciler.namespaceWorkqueue.Forget(obj)
			utilruntime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", obj))
			return nil
		}

		if err := reconciler.syncNamespaceHandler(key); err != nil {
			reconciler.namespaceWorkqueue.AddRateLimited(key)
			return fmt.Errorf("error syncing '%s': %s, requeuing", key, err.Error())
		}

		reconciler.namespaceWorkqueue.Forget(obj)
		klog.Infof("Successfully synced '%s'", key)
		return nil
	}(obj)

	if err != nil {
		utilruntime.HandleError(err)
		return true
	}

	return true
}

func (reconciler *NetworkpolicyReconciler) syncNetworkPolicyHandler(key string) error {
	namespaceName, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}
	namespace, err := reconciler.kubeclientset.CoreV1().Namespaces().Get(namespaceName, meta_v1.GetOptions{})
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("namespace '%s'does not exist", namespaceName))
		return nil
	}

	npName, err := reconciler.getDefaultNetworkpolicyName(namespace)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("error getting default network policy: %s", err))
		return nil
	}

	if _, ok := reconciler.defaultNetworkPolicies[npName]; !ok {
		klog.Warning("No default network policy set")
		reconciler.namespaceWorkqueue.Forget(key)
		return nil
	}

	if name == reconciler.defaultNetworkPolicies[npName].GetName() {

		networkPolicy, err := reconciler.networkPolicyLister.NetworkPolicies(namespaceName).Get(name)
		if err != nil {
			if errors.IsNotFound(err) {
				utilruntime.HandleError(fmt.Errorf("networkpolicy '%s' in work queue no longer exists", key))
				if err := reconciler.createDefaultNetworkPolicy(namespaceName, npName); err != nil {
					klog.Errorf("failed to recreate default network policy: %v", err)
					return fmt.Errorf("error syncing %q: %v", namespaceName, err)
				}
				return nil
			}
		} else {
			klog.Infof("Found networkpolicy %s/%s", namespaceName, networkPolicy.Name)
			if reconciler.reconcileIsNeeded(networkPolicy, npName) {
				if err := reconciler.updateDefaultNetworkPolicy(namespaceName, npName); err != nil {
					klog.Errorf("failed to update default network policy: %v", err)
					return fmt.Errorf("error syncing %q: %v", namespaceName, err)
				}
			}
		}
	}
	return nil
}
func (reconciler *NetworkpolicyReconciler) syncNamespaceHandler(key string) error {
	namespace, err := reconciler.namespacesLister.Get(key)
	if err != nil {
		if errors.IsNotFound(err) {
			utilruntime.HandleError(fmt.Errorf("namespace '%s' in work queue no longer exists", key))
			return nil
		}
	}
	klog.Infof("Found namespace %s", key)
	for _, ns := range reconciler.defaultNetworkPolicyExcludes {
		if key == ns {
			klog.Infof("Not creating default network policy in %q - namespace excluded", key)
			reconciler.namespaceWorkqueue.Forget(key)
			return nil
		}
	}

	npName, err := reconciler.getDefaultNetworkpolicyName(namespace)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("error getting default network policy: %s", err))
		return nil
	}

	if _, ok := reconciler.defaultNetworkPolicies[npName]; !ok {
		klog.Warning("No default network policy set")
		reconciler.namespaceWorkqueue.Forget(key)
		return nil
	}

	if err := reconciler.createDefaultNetworkPolicy(key, npName); err != nil {
		klog.Errorf("failed to create default network policy: %v", err)
		return fmt.Errorf("error syncing %q: %v", key, err)
	}

	reconciler.namespaceWorkqueue.Forget(key)
	klog.Infof("Successfully synced %q", key)
	return nil
}

func (reconciler *NetworkpolicyReconciler) getDefaultNetworkpolicyName(namespace *corev1.Namespace) (name string, err error) {

	npName := reconciler.defaultNetworkPolicyName

	if defaultNetworkPolicyAnnotation, ok := namespace.ObjectMeta.Annotations["karydia.gardener.cloud/networkPolicy"]; ok {
		npName = defaultNetworkPolicyAnnotation
	}

	return npName, err
}

func (reconciler *NetworkpolicyReconciler) enqueueNetworkPolicy(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		utilruntime.HandleError(err)
		return
	}
	reconciler.networkPolicyworkqueue.AddRateLimited(key)
}
func (reconciler *NetworkpolicyReconciler) enqueueNamespace(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		utilruntime.HandleError(err)
		return
	}
	reconciler.namespaceWorkqueue.AddRateLimited(key)
}

func (reconciler *NetworkpolicyReconciler) reconcileIsNeeded(actualPolicy *v1.NetworkPolicy, networkpolicyName string) bool {
	desiredPolicy := reconciler.defaultNetworkPolicies[networkpolicyName].DeepCopy()
	actualSpec, _ := actualPolicy.Spec.Marshal()
	desiredSpec, _ := desiredPolicy.Spec.Marshal()
	if bytes.Equal(actualSpec, desiredSpec) {
		return false
	}
	return true

}

func (reconciler *NetworkpolicyReconciler) updateDefaultNetworkPolicy(namespace string, networkpolicyName string) error {
	desiredPolicy := reconciler.defaultNetworkPolicies[networkpolicyName].DeepCopy()
	desiredPolicy.Namespace = namespace
	if _, err := reconciler.kubeclientset.NetworkingV1().NetworkPolicies(namespace).Update(desiredPolicy); err != nil {
		return err
	}
	return nil
}

func (reconciler *NetworkpolicyReconciler) createDefaultNetworkPolicy(namespace string, networkpolicyName string) error {
	desiredPolicy := reconciler.defaultNetworkPolicies[networkpolicyName].DeepCopy()
	desiredPolicy.ObjectMeta.Namespace = namespace

	if _, err := reconciler.kubeclientset.NetworkingV1().NetworkPolicies(namespace).Create(desiredPolicy); err != nil {
		return err
	}
	return nil

}
