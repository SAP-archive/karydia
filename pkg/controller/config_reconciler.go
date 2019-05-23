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
	"fmt"
	"github.com/karydia/karydia/pkg/apis/karydia/v1alpha1"
	"github.com/karydia/karydia/pkg/client/clientset/versioned"
	v1alpha12 "github.com/karydia/karydia/pkg/client/informers/externalversions/karydia/v1alpha1"
	v1alpha13 "github.com/karydia/karydia/pkg/client/listers/karydia/v1alpha1"
	"reflect"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog"
)

// reconciler (controller) struct
type ConfigReconciler struct {
	config		v1alpha1.KarydiaConfig
	controllers	[]ControllerInterface

	// clientset for own API group
	clientset	versioned.Interface
	lister		v1alpha13.KarydiaConfigLister
	synced		cache.InformerSynced
	// rate limited work queue
	// This is used to queue work to be processed instead of performing it as
	// soon as a change happens. This means we can ensure we only process a
	// fixed amount of resources at a time, and makes it easy to ensure we are
	// never processing the same item simultaneously in two different workers.
	workqueue	workqueue.RateLimitingInterface
}

// reconciler (controller) setup
func NewConfigReconciler (
	karydiaConfig			v1alpha1.KarydiaConfig,
	karydiaControllers		[]ControllerInterface,
	karydiaClientset		versioned.Interface,
	karydiaConfigInformer	v1alpha12.KarydiaConfigInformer,
	) *ConfigReconciler {
	reconciler := &ConfigReconciler{
		config:      karydiaConfig,
		controllers: karydiaControllers,
		clientset:   karydiaClientset,
		lister:      karydiaConfigInformer.Lister(),
		synced:      karydiaConfigInformer.Informer().HasSynced,
		workqueue:   workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "Config"),
	}

	klog.Info("Setting up event handler")
	// set up an event handler for when resources change
	karydiaConfigInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: func(old, new interface{}) {
			newConfig := new.(*v1alpha1.KarydiaConfig)
			oldConfig := old.(*v1alpha1.KarydiaConfig)
			if newConfig.ResourceVersion == oldConfig.ResourceVersion {
				// periodic resync will send update events
				// Two different versions of the same custom resource will always have different RVs.
				return
			}
			reconciler.enqueueConfig(new)
		},
		DeleteFunc: reconciler.enqueueConfig,
	})

	return reconciler
}

// set up event handlers for types we are interested in, syncing informer caches
// and starting workers
// It will block until the channel is closed, at which point it will shutdown
// the workqueue and wait for workers to finish processing their current work
// items.
func (reconciler *ConfigReconciler) Run(threadiness int, stopCh <-chan struct{}) error {
	defer utilruntime.HandleCrash()
	defer reconciler.workqueue.ShutDown()

	klog.Info("Starting karydia config reconciler")

	// wait for caches to be synced before starting workers
	klog.Info("Waiting for informer cache to sync")
	if ok := cache.WaitForCacheSync(stopCh, reconciler.synced); !ok {
		return fmt.Errorf("failed to wait for cache to sync")
	}

	// launch workers to process resources
	klog.Info("Starting worker")
	for i := 0; i < threadiness; i++ {
		go wait.Until(reconciler.runConfigWorker, time.Second, stopCh)
	}

	klog.Info("Started worker")
	<-stopCh
	klog.Info("Shutting down workers")

	return nil
}

// long-running function that will continually call process-next-item function
// in order to read and process message on workqueue
func (reconciler *ConfigReconciler) runConfigWorker() {
	for reconciler.processNextConfigWorkItem() {
	}
}

// read single work item off workqueue and attempt to process it
func (reconciler *ConfigReconciler) processNextConfigWorkItem() bool {
	obj, shutdown := reconciler.workqueue.Get()

	if shutdown {
		return false
	}

	// wrap block in func to defer workqueue.Done
	err := func(obj interface{}) error {
		// call workqueue.Done to inform workqueue about processing of item
		// has finished
		// We also must remember to call workqueue.Forget if we do not want
		// this work item being re-queued. For example, we do not call
		// Forget if a transient error occurs, instead the item is put back
		// on the workqueue and attempted again after a back-off period.
		defer reconciler.workqueue.Done(obj)
		var key string
		var ok bool

		// expect strings to come off workqueue
		// These are of the form namespace/name.
		// We do this as the delayed nature of the workqueue means the
		// items in the informer cache may actually be more up to date
		// that when the item was initially put onto the workqueue.
		if key, ok = obj.(string); !ok {
			// as item in workqueue is actually invalid, we call
			// Forget here else we'd go into a loop of attempting to
			// process a work item that is invalid
			reconciler.workqueue.Forget(obj)
			utilruntime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", obj))
			return nil
		}

		// run sync handler
		if err := reconciler.syncConfigHandler(key); err != nil {
			// put item back on workqueue to handle any transient errors
			reconciler.workqueue.AddRateLimited(key)
			return fmt.Errorf("error syncing '%s': %s, requeuing", key, err.Error())
		}

		// if no error occurs we Forget this item so it does not get
		// queued again until another change happens
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

// sync handler compares actual with desired state, and attempts to
// converge both
func (reconciler *ConfigReconciler) syncConfigHandler(key string) error {
	// convert namespace/name string into distinct (namespace and) name
	_, configName, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}

	// if no global config is set Forget this item
	if reconciler.config.Name == "" {
		utilruntime.HandleError(fmt.Errorf("No config set"))
		return nil
	}

	// if global config name equals item config name proceed
	// processing
	if configName == reconciler.config.Name {
		// get resource with (namespace/)name
		config, err := reconciler.lister.Get(configName)
		if err != nil {
			// (re)create resource from memory if it no longer exists
			if errors.IsNotFound(err) {
				utilruntime.HandleError(fmt.Errorf("karydia config '%s' no longer exists", key))
				if err := reconciler.createConfig(); err != nil {
					klog.Errorf("failed to recreate karydia config: %v", err)
					return err
				}
				return nil
			}
			return err
		} else {
			klog.Infof("Found karydia config %s", config.Name)
			// compare new config with the one in memory
			if reconciler.reconcileIsNeeded(*config) {
				// update config in memory with new one
				if err := reconciler.UpdateConfig(*config); err != nil {
					klog.Errorf("failed to update karydia config: %v", err)
					return err
				}
			}
		}
	}
	return nil
}

// take resource and convert it into namespace/name string which is
// then put onto work queue
func (reconciler *ConfigReconciler) enqueueConfig(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		utilruntime.HandleError(err)
		return
	}
	reconciler.workqueue.Add(key)
}

// check if desired and actual configs are equal
func (reconciler *ConfigReconciler) reconcileIsNeeded(desiredConfig v1alpha1.KarydiaConfig) bool {
	actualConfig := reconciler.config
	if reflect.DeepEqual(desiredConfig.Spec, actualConfig.Spec) {
		return false
	}
	return true
}

// update actual config
func (reconciler *ConfigReconciler) UpdateConfig(karydiaConfig v1alpha1.KarydiaConfig) error {
	reconciler.config = karydiaConfig
	for _, controller := range reconciler.controllers {
		if err := controller.UpdateConfig(karydiaConfig); err != nil {
			utilruntime.HandleError(err)
			return err
		}
	}
	klog.Infof("KarydiaConfig Name: %s\n", karydiaConfig.Name)
	klog.Infof("KarydiaConfig AutomountServiceAccountToken: %s\n", karydiaConfig.Spec.AutomountServiceAccountToken)
	klog.Infof("KarydiaConfig SeccompProfile: %s\n", karydiaConfig.Spec.SeccompProfile)
	klog.Infof("KarydiaConfig NetworkPolicy: %s\n", karydiaConfig.Spec.NetworkPolicy)
	return nil
}

// create config
func (reconciler *ConfigReconciler) createConfig() error {
	desiredConfig := reconciler.config.DeepCopy()
	if _, err := reconciler.clientset.KarydiaV1alpha1().KarydiaConfigs().Create(desiredConfig); err != nil {
		utilruntime.HandleError(err)
		return err
	}
	return nil
}
