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
	"fmt"
	"time"

	networkingv1 "k8s.io/api/networking/v1"
	v1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
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
	defaultNetworkPolicy         *networkingv1.NetworkPolicy
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
	defaultNetworkPolicy *networkingv1.NetworkPolicy, defaultNetworkPolicyExcludes []string) *NetworkpolicyReconciler {

	reconciler := &NetworkpolicyReconciler{

		kubeclientset:                kubeclientset,
		networkPolicyLister:          networkpolicyInformer.Lister(),
		networkPoliciesSynced:        networkpolicyInformer.Informer().HasSynced,
		namespacesLister:             namespaceInformer.Lister(),
		namespacesSynced:             namespaceInformer.Informer().HasSynced,
		namespaceWorkqueue:           workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "Namespaces"),
		networkPolicyworkqueue:       workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "Networkpolicies"),
		defaultNetworkPolicy:         defaultNetworkPolicy,
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
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}
	if reconciler.defaultNetworkPolicy == nil {
		klog.Warning("No default network policy set")
		reconciler.namespaceWorkqueue.Forget(key)
		return nil
	}

	if name == reconciler.defaultNetworkPolicy.GetName() {

		networkPolicy, err := reconciler.networkPolicyLister.NetworkPolicies(namespace).Get(name)
		if err != nil {
			if errors.IsNotFound(err) {
				utilruntime.HandleError(fmt.Errorf("networkpolicy '%s' in work queue no longer exists", key))
				if err := reconciler.createDefaultNetworkPolicy(namespace); err != nil {
					klog.Errorf("failed to recreate default network policy: %v", err)
					return fmt.Errorf("error syncing %q: %v", namespace, err)
				}
				return nil
			}
		} else {
			klog.Infof("Found networkpolicy %s/%s", namespace, networkPolicy.Name)
			if reconciler.reconcileIsNeeded(networkPolicy) {
				if err := reconciler.updateDefaultNetworkPolicy(namespace); err != nil {
					klog.Errorf("failed to update default network policy: %v", err)
					return fmt.Errorf("error syncing %q: %v", namespace, err)
				}
			}
		}
	}
	return nil
}
func (reconciler *NetworkpolicyReconciler) syncNamespaceHandler(key string) error {

	_, err := reconciler.namespacesLister.Get(key)

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

	if reconciler.defaultNetworkPolicy == nil {
		klog.Warning("No default network policy set")
		reconciler.namespaceWorkqueue.Forget(key)
		return nil
	}

	if err := reconciler.createDefaultNetworkPolicy(key); err != nil {
		klog.Errorf("failed to create default network policy: %v", err)
		return fmt.Errorf("error syncing %q: %v", key, err)
	}

	reconciler.namespaceWorkqueue.Forget(key)
	klog.Infof("Successfully synced %q", key)
	return nil
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

func (reconciler *NetworkpolicyReconciler) reconcileIsNeeded(actualPolicy *v1.NetworkPolicy) bool {
	desiredPolicy := reconciler.defaultNetworkPolicy.DeepCopy()
	actualSpec, _ := actualPolicy.Spec.Marshal()
	desiredSpec, _ := desiredPolicy.Spec.Marshal()
	if bytes.Equal(actualSpec, desiredSpec) {
		return false
	}
	return true

}

func (reconciler *NetworkpolicyReconciler) updateDefaultNetworkPolicy(namespace string) error {
	desiredPolicy := reconciler.defaultNetworkPolicy.DeepCopy()
	desiredPolicy.Namespace = namespace
	if _, err := reconciler.kubeclientset.NetworkingV1().NetworkPolicies(namespace).Update(desiredPolicy); err != nil {
		return err
	}
	return nil
}

func (reconciler *NetworkpolicyReconciler) createDefaultNetworkPolicy(namespace string) error {
	desiredPolicy := reconciler.defaultNetworkPolicy.DeepCopy()
	desiredPolicy.ObjectMeta.Namespace = namespace

	if _, err := reconciler.kubeclientset.NetworkingV1().NetworkPolicies(namespace).Create(desiredPolicy); err != nil {
		return err
	}
	return nil

}
