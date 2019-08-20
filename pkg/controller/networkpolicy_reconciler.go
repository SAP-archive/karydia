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
	"fmt"
	"github.com/karydia/karydia/pkg/logger"
	"time"

	"github.com/karydia/karydia/pkg/apis/karydia/v1alpha1"
	"github.com/karydia/karydia/pkg/client/clientset/versioned"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	v1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	namespaceInformer "k8s.io/client-go/informers/core/v1"
	networkpolicyInformer "k8s.io/client-go/informers/networking/v1"
	"k8s.io/client-go/kubernetes"
	kubelistersv1 "k8s.io/client-go/listers/core/v1"
	kubelistersNetworkingv1 "k8s.io/client-go/listers/networking/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

const controllerAgentName = "networkpolicy_reconciler"

type NetworkpolicyReconciler struct {
	log                          *logger.Logger
	defaultNetworkPolicyName     string
	defaultNetworkPolicies       map[string]*networkingv1.NetworkPolicy
	defaultNetworkPolicyExcludes []string

	kubeclientset          kubernetes.Interface
	karydiaClientset       versioned.Interface
	networkPolicyLister    kubelistersNetworkingv1.NetworkPolicyLister
	networkPoliciesSynced  cache.InformerSynced
	namespacesLister       kubelistersv1.NamespaceLister
	namespacesSynced       cache.InformerSynced
	networkPolicyworkqueue workqueue.RateLimitingInterface
	namespaceWorkqueue     workqueue.RateLimitingInterface
}

type Setting struct {
	value string
	src   string
}

func (reconciler *NetworkpolicyReconciler) UpdateConfig(karydiaConfig v1alpha1.KarydiaConfig) error {
	reconciler.defaultNetworkPolicyName = karydiaConfig.Spec.NetworkPolicy
	return nil
}

func NewNetworkpolicyReconciler(
	kubeclientset kubernetes.Interface,
	karydiaClientset versioned.Interface,
	networkpolicyInformer networkpolicyInformer.NetworkPolicyInformer, namespaceInformer namespaceInformer.NamespaceInformer,
	defaultNetworkPolicies map[string]*networkingv1.NetworkPolicy, defaultNetworkPolicyName string, defaultNetworkPolicyExcludes []string) *NetworkpolicyReconciler {

	reconciler := &NetworkpolicyReconciler{
		log:                          logger.NewComponentLogger(logger.GetCallersFilename()),
		defaultNetworkPolicyName:     defaultNetworkPolicyName,
		kubeclientset:                kubeclientset,
		karydiaClientset:             karydiaClientset,
		networkPolicyLister:          networkpolicyInformer.Lister(),
		networkPoliciesSynced:        networkpolicyInformer.Informer().HasSynced,
		namespacesLister:             namespaceInformer.Lister(),
		namespacesSynced:             namespaceInformer.Informer().HasSynced,
		namespaceWorkqueue:           workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "Namespaces"),
		networkPolicyworkqueue:       workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "Networkpolicies"),
		defaultNetworkPolicies:       defaultNetworkPolicies,
		defaultNetworkPolicyExcludes: defaultNetworkPolicyExcludes,
	}

	reconciler.log.Infoln("Setting up event handlers")
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
		UpdateFunc: func(old, new interface{}) {
			newNamespace := new.(*corev1.Namespace)
			oldNamespace := old.(*corev1.Namespace)
			if newNamespace.ResourceVersion == oldNamespace.ResourceVersion {
				return
			}
			reconciler.enqueueNamespace(new)
		},
	})

	return reconciler
}

func (reconciler *NetworkpolicyReconciler) Run(threadiness int, stopCh <-chan struct{}) error {
	defer reconciler.log.HandleCrash()
	defer reconciler.networkPolicyworkqueue.ShutDown()
	defer reconciler.namespaceWorkqueue.ShutDown()

	reconciler.log.Infoln("Starting karydia network policy reconciler")
	reconciler.log.Infoln("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(stopCh, reconciler.networkPoliciesSynced, reconciler.namespacesSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	reconciler.log.Infoln("Starting workers")

	for i := 0; i < threadiness; i++ {
		go wait.Until(reconciler.runNetworkPolicyWorker, time.Second, stopCh)
		go wait.Until(reconciler.runNamespaceWorker, time.Second, stopCh)
	}

	reconciler.log.Infoln("Started workers")
	<-stopCh
	reconciler.log.Infoln("Shutting down workers")

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
			reconciler.log.Errorf("expected string in workqueue but got %#v", obj)
			return nil
		}

		if err := reconciler.syncNetworkPolicyHandler(key); err != nil {
			reconciler.networkPolicyworkqueue.AddRateLimited(key)
			return fmt.Errorf("error syncing '%s': %s, requeuing", key, err.Error())
		}

		reconciler.networkPolicyworkqueue.Forget(obj)
		reconciler.log.Infof("Successfully synced network policy '%s'", key)
		return nil
	}(obj)

	if err != nil {
		reconciler.log.Errorln(err)
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
			reconciler.log.Errorf("expected string in workqueue but got %#v", obj)
			return nil
		}

		if err := reconciler.syncNamespaceHandler(key); err != nil {
			reconciler.namespaceWorkqueue.AddRateLimited(key)
			return fmt.Errorf("error syncing '%s': %s, requeuing", key, err.Error())
		}

		reconciler.namespaceWorkqueue.Forget(obj)
		reconciler.log.Infof("Successfully synced workitem '%s'", key)
		return nil
	}(obj)

	if err != nil {
		reconciler.log.Errorln(err)
		return true
	}

	return true
}

func (reconciler *NetworkpolicyReconciler) syncNetworkPolicyHandler(key string) error {
	reconciler.log.Infof("Start network policy reconciler (syncNetworkPolicyHandler) for '%s'", key)
	namespaceName, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		reconciler.log.Errorln("invalid resource key:", key)
		return nil
	}
	namespace, err := reconciler.kubeclientset.CoreV1().Namespaces().Get(namespaceName, meta_v1.GetOptions{})
	if err != nil {
		reconciler.log.Errorf("namespace '%s'does not exist", namespaceName)
		return nil
	}

	setting := reconciler.getDefaultNetworkpolicySetting(namespace)
	npName := setting.value

	if _, ok := reconciler.defaultNetworkPolicies[npName]; !ok {
		if err := reconciler.updateBuffer(npName); err != nil {

			reconciler.log.Warnf("Failed to get default network policy '%s'", npName)
			return nil
		}
	}

	if name == reconciler.defaultNetworkPolicies[npName].GetName() {
		networkPolicy, err := reconciler.networkPolicyLister.NetworkPolicies(namespaceName).Get(name)
		if err != nil {
			if errors.IsNotFound(err) {
				reconciler.log.Errorf("networkpolicy '%s' in work queue no longer exists", key)
				if err := reconciler.createDefaultNetworkPolicy(namespaceName, setting); err != nil {
					reconciler.log.Errorln("failed to recreate default network policy:", err)
					return fmt.Errorf("error syncing '%s': %v", namespaceName, err)
				}
				return nil
			}
			return err
		} else {
			reconciler.log.Infof("Found networkpolicy '%s'", key)
			if reconciler.reconcileIsNeeded(networkPolicy, name) {
				if err := reconciler.updateDefaultNetworkPolicy(namespaceName, name); err != nil {
					reconciler.log.Errorln("failed to update default network policy:", err)
					return fmt.Errorf("error syncing '%s': %v", key, err)
				}
			} else if err != nil {
				return fmt.Errorf("error syncing buffer '%s': %v", key, err)
			}
		}
	}
	return nil
}

func (reconciler *NetworkpolicyReconciler) syncNamespaceHandler(key string) error {
	reconciler.log.Infof("Start network policy reconciler for namespace '%s'", key)
	namespace, err := reconciler.namespacesLister.Get(key)
	if err != nil {
		if errors.IsNotFound(err) {
			reconciler.log.Errorf("namespace '%s' in work queue no longer exists", key)
			return nil
		}
	}
	reconciler.log.Infof("Found namespace '%s' in queue", key)
	for _, ns := range reconciler.defaultNetworkPolicyExcludes {
		if key == ns {
			reconciler.log.Infof("Not creating default network policy in '%s' - namespace excluded", key)
			return nil
		}
	}

	setting := reconciler.getDefaultNetworkpolicySetting(namespace)
	npName := setting.value

	if _, ok := reconciler.defaultNetworkPolicies[npName]; !ok {
		if err := reconciler.updateBuffer(npName); err != nil {

			reconciler.log.Warnf("Failed to get default network policy '%s'", npName)
			return nil
		}
	}

	networkPolicy, err := reconciler.networkPolicyLister.NetworkPolicies(key).Get(npName)
	if err != nil {
		if errors.IsNotFound(err) {
			if err := reconciler.createDefaultNetworkPolicy(key, setting); err != nil {
				reconciler.log.Errorln("failed to create default network policy:", err)
				return fmt.Errorf("error syncing '%s': %v", key, err)
			}
			reconciler.log.Infof("Successfully synced namespace '%s'", key)
			return nil
		}
		return err
	} else {
		reconciler.log.Infof("Found networkpolicy '%s/%s'", key, npName)
		if reconciler.reconcileIsNeeded(networkPolicy, npName) {
			if err := reconciler.updateDefaultNetworkPolicy(key, npName); err != nil {
				reconciler.log.Errorln("failed to update default network policy:", err)
				return fmt.Errorf("error syncing '%s': %v", key, err)
			}
		} else if err != nil {
			return fmt.Errorf("error syncing buffer '%s': %v", key, err)
		}
	}
	return nil
}

func (reconciler *NetworkpolicyReconciler) getDefaultNetworkpolicySetting(namespace *corev1.Namespace) Setting {

	npName := reconciler.defaultNetworkPolicyName
	src := "config"
	if defaultNetworkPolicyAnnotation, ok := namespace.ObjectMeta.Annotations["karydia.gardener.cloud/networkPolicy"]; ok {
		reconciler.log.Infof("Found annotation, use network policy '%s'", defaultNetworkPolicyAnnotation)
		npName = defaultNetworkPolicyAnnotation
		src = "namespace"
	}
	return Setting{value: npName, src: src}
}

func (reconciler *NetworkpolicyReconciler) enqueueNetworkPolicy(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		reconciler.log.Errorln(err)
		return
	}
	reconciler.networkPolicyworkqueue.Add(key)
}
func (reconciler *NetworkpolicyReconciler) enqueueNamespace(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		reconciler.log.Errorln(err)
		return
	}
	reconciler.namespaceWorkqueue.Add(key)
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

func (reconciler *NetworkpolicyReconciler) updateBuffer(networkpolicyName string) error {

	karydiaNetworkPolicy, err := reconciler.karydiaClientset.KarydiaV1alpha1().KarydiaNetworkPolicies().Get(networkpolicyName, meta_v1.GetOptions{})
	if err != nil {
		reconciler.log.Errorf("Failed to get karydia network policy '%s'", networkpolicyName)
		return fmt.Errorf("Failed to get karydia network policy '%s'", networkpolicyName)
	}

	var policy networkingv1.NetworkPolicy
	policy.Name = networkpolicyName
	policy.Spec = *karydiaNetworkPolicy.Spec.DeepCopy()
	reconciler.defaultNetworkPolicies[networkpolicyName] = &policy
	reconciler.log.Infof("Network policy '%s' loaded into buffer. New Buffer length: %v", policy.GetName(), len(reconciler.defaultNetworkPolicies))

	return nil
}

func (reconciler *NetworkpolicyReconciler) createDefaultNetworkPolicy(namespace string, setting Setting) error {
	networkpolicyName := setting.value
	desiredPolicy := reconciler.defaultNetworkPolicies[networkpolicyName].DeepCopy()
	desiredPolicy.ObjectMeta.Namespace = namespace

	annotations := make(map[string]string)
	annotations["karydia.gardener.cloud/networkPolicy.internal"] = setting.src + "/" + setting.value
	desiredPolicy.ObjectMeta.SetAnnotations(annotations)

	reconciler.log.Infof("Deep copy of network policy with name '%s'", desiredPolicy.GetName())
	if _, err := reconciler.kubeclientset.NetworkingV1().NetworkPolicies(namespace).Create(desiredPolicy); err != nil {
		reconciler.log.Errorf("Network policy creation failed. Name specified: '%s'; Actual name: '%s'", networkpolicyName, desiredPolicy.GetName())
		return err
	}
	if err := reconciler.deleteMultipleDefaultNetworkPolicies(namespace, networkpolicyName); err != nil {
		reconciler.log.Errorln("Failed to delete network policy", networkpolicyName)
		return err
	}
	return nil
}

func (reconciler *NetworkpolicyReconciler) deleteMultipleDefaultNetworkPolicies(namespace string, networkpolicyName string) error {
	for npName := range reconciler.defaultNetworkPolicies {
		if npName != networkpolicyName {
			if _, err := reconciler.kubeclientset.NetworkingV1().NetworkPolicies(namespace).Get(npName, meta_v1.GetOptions{}); err == nil {
				err := reconciler.kubeclientset.NetworkingV1().NetworkPolicies(namespace).Delete(npName, &meta_v1.DeleteOptions{})
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}
