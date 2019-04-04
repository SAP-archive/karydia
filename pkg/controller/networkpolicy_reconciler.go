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
	"fmt"
	"time"

	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	networkpolicyInformer "k8s.io/client-go/informers/networking/v1"
	"k8s.io/client-go/kubernetes"
	kubelistersNetworkingv1 "k8s.io/client-go/listers/networking/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog"
)

const controllerAgentName = "networkpolicy_reconciler"

const (
	// SuccessSynced is used as part of the Event 'reason' when a Foo is synced
	SuccessSynced = "Synced"
	// ErrResourceExists is used as part of the Event 'reason' when a Foo fails
	// to sync due to a Deployment of the same name already existing.
	ErrResourceExists = "ErrResourceExists"

	// MessageResourceExists is the message used for Events when a resource
	// fails to sync due to a Deployment already existing
	MessageResourceExists = "Resource %q already exists and is not managed by Foo"
	// MessageResourceSynced is the message used for an Event fired when a Foo
	// is synced successfully
	MessageResourceSynced = "Karydia Networkpolicy synced successfully"
)

// Controller is the controller implementation for Foo resources
type NetworkpolicyReconciler struct {
	defaultNetworkPolicy *networkingv1.NetworkPolicy

	// kubeclientset is a standard kubernetes clientset
	kubeclientset         kubernetes.Interface
	networkPolicyLister   kubelistersNetworkingv1.NetworkPolicyLister
	networkPoliciesSynced cache.InformerSynced

	// workqueue is a rate limited work queue. This is used to queue work to be
	// processed instead of performing it as soon as a change happens. This
	// means we can ensure we only process a fixed amount of resources at a
	// time, and makes it easy to ensure we are never processing the same item
	// simultaneously in two different workers.
	workqueue workqueue.RateLimitingInterface
	// recorder is an event recorder for recording Event resources to the
	// Kubernetes API.
	recorder record.EventRecorder
}

// NewController returns a new sample controller
func NewNetworkpolicyReconciler(
	kubeclientset kubernetes.Interface,
	networkpolicyInformer networkpolicyInformer.NetworkPolicyInformer,
	defaultNetworkPolicy *networkingv1.NetworkPolicy) *NetworkpolicyReconciler {

	reconciler := &NetworkpolicyReconciler{

		kubeclientset:         kubeclientset,
		networkPolicyLister:   networkpolicyInformer.Lister(),
		networkPoliciesSynced: networkpolicyInformer.Informer().HasSynced,
		workqueue:             workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "Networkpolicies"),
		defaultNetworkPolicy:  defaultNetworkPolicy,
	}

	klog.Info("Setting up event handlers")
	// Set up an event handler for when Foo resources change
	networkpolicyInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: func(old, new interface{}) {
			newNetworkPolicy := new.(*networkingv1.NetworkPolicy)
			oldNetworkPolicy := old.(*networkingv1.NetworkPolicy)
			if newNetworkPolicy.ResourceVersion == oldNetworkPolicy.ResourceVersion || oldNetworkPolicy.GetName() != reconciler.defaultNetworkPolicy.GetName() {
				// Periodic resync will send update events for all known Deployments.
				// Two different versions of the same Deployment will always have different RVs.
				return
			}
			reconciler.enqueueNetworkPolicy(new)
		},
		DeleteFunc: reconciler.enqueueNetworkPolicy,
	})

	return reconciler
}

// Run will set up the event handlers for types we are interested in, as well
// as syncing informer caches and starting workers. It will block until stopCh
// is closed, at which point it will shutdown the workqueue and wait for
// workers to finish processing their current work items.
func (reconciler *NetworkpolicyReconciler) Run(threadiness int, stopCh <-chan struct{}) error {
	defer utilruntime.HandleCrash()
	defer reconciler.workqueue.ShutDown()

	// Start the informer factories to begin populating the informer caches
	klog.Info("Starting Foo controller")

	// Wait for the caches to be synced before starting workers
	klog.Info("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(stopCh, reconciler.networkPoliciesSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	klog.Info("Starting workers")
	// Launch two workers to process Foo resources
	for i := 0; i < threadiness; i++ {
		go wait.Until(reconciler.runWorker, time.Second, stopCh)
	}

	klog.Info("Started workers")
	<-stopCh
	klog.Info("Shutting down workers")

	return nil
}

// runWorker is a long-running function that will continually call the
// processNextWorkItem function in order to read and process a message on the
// workqueue.
func (reconciler *NetworkpolicyReconciler) runWorker() {
	for reconciler.processNextWorkItem() {
	}
}

// processNextWorkItem will read a single work item off the workqueue and
// attempt to process it, by calling the syncHandler.
func (reconciler *NetworkpolicyReconciler) processNextWorkItem() bool {
	obj, shutdown := reconciler.workqueue.Get()

	if shutdown {
		return false
	}

	// We wrap this block in a func so we can defer c.workqueue.Done.
	err := func(obj interface{}) error {
		// We call Done here so the workqueue knows we have finished
		// processing this item. We also must remember to call Forget if we
		// do not want this work item being re-queued. For example, we do
		// not call Forget if a transient error occurs, instead the item is
		// put back on the workqueue and attempted again after a back-off
		// period.
		defer reconciler.workqueue.Done(obj)
		var key string
		var ok bool
		// We expect strings to come off the workqueue. These are of the
		// form namespace/name. We do this as the delayed nature of the
		// workqueue means the items in the informer cache may actually be
		// more up to date that when the item was initially put onto the
		// workqueue.
		if key, ok = obj.(string); !ok {
			// As the item in the workqueue is actually invalid, we call
			// Forget here else we'd go into a loop of attempting to
			// process a work item that is invalid.
			reconciler.workqueue.Forget(obj)
			utilruntime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", obj))
			return nil
		}
		// Run the syncHandler, passing it the namespace/name string of the
		// Foo resource to be synced.
		if err := reconciler.syncHandler(key); err != nil {
			// Put the item back on the workqueue to handle any transient errors.
			reconciler.workqueue.AddRateLimited(key)
			return fmt.Errorf("error syncing '%s': %s, requeuing", key, err.Error())
		}
		// Finally, if no error occurs we Forget this item so it does not
		// get queued again until another change happens.
		reconciler.workqueue.Forget(obj)
		klog.Infof("Successfully synced '%s'", key)
		return nil
	}(obj)

	if err != nil {
		utilruntime.HandleError(err)
		return true
	}

	return true
}

// syncHandler compares the actual state with the desired, and attempts to
// converge the two. It then updates the Status block of the Foo resource
// with the current status of the resource.
func (reconciler *NetworkpolicyReconciler) syncHandler(key string) error {
	// Convert the namespace/name string into a distinct namespace and name
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}

	// Get the Foo resource with this namespace/name
	networkPolicy, err := reconciler.networkPolicyLister.NetworkPolicies(namespace).Get(name)

	if err != nil {
		// The Foo resource may no longer exist, in which case we stop
		// processing.
		if errors.IsNotFound(err) {
			utilruntime.HandleError(fmt.Errorf("networkpolicy '%s' in work queue no longer exists", key))
			// CHeck if name == defaultNetworkPolicyName, if yes recreate it
			return nil
		}

		klog.Infof("Found networkpolicy %s", networkPolicy.Name)

		return err
	}
	return nil
}

// enqueueFoo takes a Foo resource and converts it into a namespace/name
// string which is then put onto the work queue. This method should *not* be
// passed resources of any type other than Foo.
func (reconciler *NetworkpolicyReconciler) enqueueNetworkPolicy(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		utilruntime.HandleError(err)
		return
	}
	reconciler.workqueue.AddRateLimited(key)
}

// func (reconciler *NetworkpolicyReconciler) defaultNetworkPolicyExists(namespace string) bool {
// 	if _, err := reconciler.kubeClientset.NetworkingV1().NetworkPolicies(namespace).Get(reconciler.defaultNetworkPolicy.GetName(), meta_v1.GetOptions{}); err == nil {
// 		return true
// 	} else {
// 		return false
// 	}

// }

// func (reconciler *NetworkpolicyReconciler) updateDefaultNetworkPolicy(namespace string) error {
// 	desiredPolicy := reconciler.defaultNetworkPolicy.DeepCopy()
// 	desiredPolicy.ObjectMeta.Namespace = namespace

// 	if actualPolicy, err := reconciler.kubeClientset.NetworkingV1().NetworkPolicies(namespace).Get(desiredPolicy.GetName(), meta_v1.GetOptions{}); err == nil {
// 		actualSpec, _ := actualPolicy.Spec.Marshal()
// 		desiredSpec, _ := desiredPolicy.Spec.Marshal()
// 		if bytes.Equal(actualSpec, desiredSpec) {
// 			return nil
// 		} else if _, err := reconciler.kubeClientset.NetworkingV1().NetworkPolicies(namespace).Update(desiredPolicy); err != nil {
// 			return err
// 		}
// 	}
// 	return nil

// }

// func (reconciler *NetworkpolicyReconciler) createDefaultNetworkPolicy(namespace string) error {
// 	desiredPolicy := reconciler.defaultNetworkPolicy.DeepCopy()
// 	desiredPolicy.ObjectMeta.Namespace = namespace

// 	if _, err := reconciler.kubeClientset.NetworkingV1().NetworkPolicies(namespace).Create(desiredPolicy); err != nil {
// 		return err
// 	}
// 	return nil

// }
