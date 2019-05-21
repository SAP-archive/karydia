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
	"context"
	"fmt"
	"github.com/karydia/karydia/pkg/apis/karydia/v1alpha1"
	"github.com/karydia/karydia/pkg/client/clientset/versioned/fake"
	v1alpha13 "github.com/karydia/karydia/pkg/client/clientset/versioned/typed/karydia/v1alpha1"
	"github.com/karydia/karydia/pkg/client/informers/externalversions"
	v1alpha12 "github.com/karydia/karydia/pkg/client/informers/externalversions/karydia/v1alpha1"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"testing"
	"time"
)

type testSettings struct {
	t                     *testing.T
	clientset             *fake.Clientset
	configInformer        v1alpha12.KarydiaConfigInformer
	controllers           []ControllerInterface
	configWorker          v1alpha13.KarydiaConfigInterface
	sharedInformerFactory externalversions.SharedInformerFactory
	waitTimeoutSeconds    time.Duration
}
func newTestSettings(t *testing.T, kubeObjects []runtime.Object, karydiaControllers []ControllerInterface) *testSettings {
	clientset := fake.NewSimpleClientset(kubeObjects...)
	sharedInformerFactory := externalversions.NewSharedInformerFactory(clientset, noResyncPeriodFunc())
	return &testSettings{
		t:                     t,
		clientset:             clientset,
		configInformer:        sharedInformerFactory.Karydia().V1alpha1().KarydiaConfigs(),
		controllers:           karydiaControllers,
		configWorker:          clientset.KarydiaV1alpha1().KarydiaConfigs(),
		sharedInformerFactory: sharedInformerFactory,
		waitTimeoutSeconds:    10 * time.Second,
	}
}

type testConfigParams struct {
	automountServiceAccountToken	string
	seccompProfile					string
	networkPolicy					string
}
type testConfig struct {
	t		*testing.T
	config	v1alpha1.KarydiaConfig
}
func newTestConfig(t *testing.T, resourceVersion string, params testConfigParams) *testConfig {
	configName := "testConfig"
	return &testConfig{
		t: t,
		config: v1alpha1.KarydiaConfig{
			ObjectMeta: meta_v1.ObjectMeta{
				Name: configName,
				ResourceVersion: resourceVersion,
			},
			Spec: v1alpha1.KarydiaConfigSpec{
				AutomountServiceAccountToken: params.automountServiceAccountToken,
				SeccompProfile: params.seccompProfile,
				NetworkPolicy: params.networkPolicy,
			},
		},
	}
}

type testControllerInterface interface {
	UpdateConfig(karydiaConfig v1alpha1.KarydiaConfig) error
	isUpdated() bool
}
type testController struct {
	name				string
	updated				bool
	updateError			error
}
func (c *testController) UpdateConfig(karydiaConfig v1alpha1.KarydiaConfig) error {
	if c.updateError != nil {
		c.updated = false
		return c.updateError
	}
	c.updated = true
	return nil
}
func (c *testController) isUpdated() bool {
	return c.updated
}

func TestNewConfigReconciler(t *testing.T) {
	// setup
	assert := assert.New(t)
	c := newTestConfig(t, "1", testConfigParams{
		automountServiceAccountToken:	"testAutomountServiceAccountToken",
		seccompProfile:					"testSeccompProfile",
		networkPolicy:					"testNetworkPolicy",
	})
	s := newTestSettings(t, []runtime.Object{}, []ControllerInterface{new(testController)})

	// create reconciler and check values
	r := NewConfigReconciler(c.config, s.controllers, s.clientset, s.configInformer)
	assert.Equal(c.config, r.config)
	assert.Equal(s.controllers, r.controllers)
	assert.Equal(s.clientset, r.clientset)
	assert.Equal(s.configInformer.Lister(), r.lister)

	// change expected values to make sure no pointers are used for config
	c.config.ResourceVersion = "2"
	assert.NotEqual(c.config, r.config)
	// undo change and check again
	c.config.ResourceVersion = "1"
	assert.Equal(c.config, r.config)

	// change expected values to make sure no pointers are used for config interfaces
	s.controllers = append(s.controllers, new(testController))
	assert.NotEqual(s.controllers, r.controllers)
	// undo change and check again
	s.controllers = s.controllers[:1]
	assert.Equal(s.controllers, r.controllers)
}

func TestConfigReconciler_syncConfigHandlerWithoutConfigElement(t *testing.T) {
	// setup
	assert := assert.New(t)
	c := newTestConfig(t, "1", testConfigParams{
		automountServiceAccountToken:	"testAutomountServiceAccountToken",
		seccompProfile:					"testSeccompProfile",
		networkPolicy:					"testNetworkPolicy",
	})
	s := newTestSettings(t, []runtime.Object{}, []ControllerInterface{})
	r := NewConfigReconciler(c.config, s.controllers, s.clientset, s.configInformer)

	// valid resource key namespace/name
	assert.NoError(r.syncConfigHandler("a/" + c.config.Name))
	assert.NoError(s.configWorker.Delete(c.config.Name, &metav1.DeleteOptions{}))
	// valid resource key /name
	assert.NoError(r.syncConfigHandler("/" + c.config.Name))
	assert.NoError(s.configWorker.Delete(c.config.Name, &metav1.DeleteOptions{}))
	// valid resource key name
	assert.NoError(r.syncConfigHandler(c.config.Name))
	// valid resource key name again
	assert.Error(r.syncConfigHandler(c.config.Name))
	assert.NoError(s.configWorker.Delete(c.config.Name, &metav1.DeleteOptions{}))
	// invalid resource key
	assert.NoError(r.syncConfigHandler("a/b/c"))

	// no global config name
	r.config.Name = ""
	assert.NoError(r.syncConfigHandler(c.config.Namespace + "/" + c.config.Name))
	// no global config
	r.config = *new(v1alpha1.KarydiaConfig)
	assert.NoError(r.syncConfigHandler(c.config.Namespace + "/" + c.config.Name))
}

func TestConfigReconciler_enqueueConfig(t *testing.T) {
	// setup
	assert := assert.New(t)
	c := newTestConfig(t, "1", testConfigParams{
		automountServiceAccountToken:	"testAutomountServiceAccountToken",
		seccompProfile:					"testSeccompProfile",
		networkPolicy:					"testNetworkPolicy",
	})
	s := newTestSettings(t, []runtime.Object{}, []ControllerInterface{})
	r := NewConfigReconciler(c.config, s.controllers, s.clientset, s.configInformer)

	// empty workqueue
	assert.Equal(0, r.workqueue.Len())
	// add wrong element
	r.enqueueConfig(new(runtime.Object))
	assert.Equal(0, r.workqueue.Len())
	// add right element
	e0 := new(metav1.ObjectMeta)
	e0.Name = "testElement0"
	r.enqueueConfig(e0)
	assert.Equal(1, r.workqueue.Len())
	// add another right element
	e1 := new(metav1.ObjectMeta)
	e1.Name = "testElement1"
	r.enqueueConfig(e1)
	assert.Equal(2, r.workqueue.Len())
}

func TestConfigReconciler_reconcileIsNeeded(t *testing.T) {
	// setup
	assert := assert.New(t)
	c := newTestConfig(t, "1", testConfigParams{
		automountServiceAccountToken:	"testAutomountServiceAccountToken",
		seccompProfile:					"testSeccompProfile",
		networkPolicy:					"testNetworkPolicy",
	})
	s := newTestSettings(t, []runtime.Object{}, []ControllerInterface{})
	r := NewConfigReconciler(c.config, s.controllers, s.clientset, s.configInformer)

	// equal configs
	assert.Equal(c.config, r.config)
	assert.False(r.reconcileIsNeeded(c.config))
	// different configs
	c.config.Spec.SeccompProfile = "newValue"
	assert.NotEqual(c.config, r.config)
	assert.True(r.reconcileIsNeeded(c.config))
	// different configs with empty config
	c.config = *new(v1alpha1.KarydiaConfig)
	assert.NotEqual(c.config, r.config)
	assert.True(r.reconcileIsNeeded(c.config))
}

func TestConfigReconciler_UpdateConfigWithoutControllers(t *testing.T) {
	// setup
	assert := assert.New(t)
	c := newTestConfig(t, "1", testConfigParams{
		automountServiceAccountToken:	"testAutomountServiceAccountToken",
		seccompProfile:					"testSeccompProfile",
		networkPolicy:					"testNetworkPolicy",
	})
	s := newTestSettings(t, []runtime.Object{}, []ControllerInterface{})
	r := NewConfigReconciler(c.config, s.controllers, s.clientset, s.configInformer)

	// equal configs
	assert.Equal(c.config, r.config)
	assert.NoError(r.UpdateConfig(c.config))
	assert.Equal(c.config, r.config)
	// different configs
	c.config.Name = "newName"
	assert.NotEqual(c.config, r.config)
	assert.NoError(r.UpdateConfig(c.config))
	assert.Equal(c.config, r.config)
	// different configs with empty config
	c.config = *new(v1alpha1.KarydiaConfig)
	assert.NotEqual(c.config, r.config)
	assert.NoError(r.UpdateConfig(c.config))
	assert.Equal(c.config, r.config)
}

func TestConfigReconciler_UpdateConfigWithController(t *testing.T) {
	// setup
	assert := assert.New(t)
	c := newTestConfig(t, "1", testConfigParams{
		automountServiceAccountToken:	"testAutomountServiceAccountToken",
		seccompProfile:					"testSeccompProfile",
		networkPolicy:					"testNetworkPolicy",
	})
	s := newTestSettings(t, []runtime.Object{}, []ControllerInterface{new(testController)})
	r := NewConfigReconciler(c.config, s.controllers, s.clientset, s.configInformer)
	controller := s.controllers[0].(*testController)

	// equal configs
	assert.Equal(c.config, r.config)
	assert.False(controller.updated)
	assert.NoError(r.UpdateConfig(c.config))
	assert.Equal(c.config, r.config)
	assert.True(controller.updated)
	// different configs
	controller.updated = false
	c.config.Name = "newName"
	assert.NotEqual(c.config, r.config)
	assert.False(controller.updated)
	assert.NoError(r.UpdateConfig(c.config))
	assert.Equal(c.config, r.config)
	assert.True(controller.updated)
	// different configs again but with already updated flag
	c.config.Name = "newSpecialName"
	assert.NotEqual(c.config, r.config)
	assert.True(controller.updated)
	assert.NoError(r.UpdateConfig(c.config))
	assert.Equal(c.config, r.config)
	assert.True(controller.updated)
	// different configs with empty config
	controller.updated = false
	c.config = *new(v1alpha1.KarydiaConfig)
	assert.NotEqual(c.config, r.config)
	assert.False(controller.updated)
	assert.NoError(r.UpdateConfig(c.config))
	assert.Equal(c.config, r.config)
	assert.True(controller.updated)
}

func TestConfigReconciler_UpdateConfigWithControllers(t *testing.T) {
	// setup
	assert := assert.New(t)
	c := newTestConfig(t, "1", testConfigParams{
		automountServiceAccountToken:	"testAutomountServiceAccountToken",
		seccompProfile:					"testSeccompProfile",
		networkPolicy:					"testNetworkPolicy",
	})
	controller0 := testController{name: "testController0"}
	controller1 := testController{name: "testController1"}
	s := newTestSettings(t, []runtime.Object{}, []ControllerInterface{&controller0, &controller1})
	r := NewConfigReconciler(c.config, s.controllers, s.clientset, s.configInformer)

	// equal configs
	assert.Equal(c.config, r.config)
	assert.False(controller0.updated)
	assert.False(controller1.updated)
	assert.NoError(r.UpdateConfig(c.config))
	assert.Equal(c.config, r.config)
	assert.True(controller0.updated)
	assert.True(controller1.updated)
	// different configs
	controller0.updated = false
	controller1.updated = false
	c.config.Name = "newName"
	assert.NotEqual(c.config, r.config)
	assert.False(controller0.updated)
	assert.False(controller1.updated)
	assert.NoError(r.UpdateConfig(c.config))
	assert.Equal(c.config, r.config)
	assert.True(controller0.updated)
	assert.True(controller1.updated)
	// different configs again but with already updated flag
	c.config.Name = "newSpecialName"
	assert.NotEqual(c.config, r.config)
	assert.True(controller0.updated)
	assert.True(controller1.updated)
	assert.NoError(r.UpdateConfig(c.config))
	assert.Equal(c.config, r.config)
	assert.True(controller0.updated)
	assert.True(controller1.updated)
	// different configs with empty config
	controller0.updated = false
	controller1.updated = false
	c.config = *new(v1alpha1.KarydiaConfig)
	assert.NotEqual(c.config, r.config)
	assert.False(controller0.updated)
	assert.False(controller1.updated)
	assert.NoError(r.UpdateConfig(c.config))
	assert.Equal(c.config, r.config)
	assert.True(controller0.updated)
	assert.True(controller1.updated)
}

func TestConfigReconciler_UpdateConfigWithControllersAndError(t *testing.T) {
	// setup
	assert := assert.New(t)
	c := newTestConfig(t, "1", testConfigParams{
		automountServiceAccountToken:	"testAutomountServiceAccountToken",
		seccompProfile:					"testSeccompProfile",
		networkPolicy:					"testNetworkPolicy",
	})
	controller0 := testController{name: "testController0"}
	controller1 := testController{name: "testController1", updateError: fmt.Errorf("test config update error")}
	s := newTestSettings(t, []runtime.Object{}, []ControllerInterface{&controller0, &controller1})
	r := NewConfigReconciler(c.config, s.controllers, s.clientset, s.configInformer)

	// equal configs
	assert.Equal(c.config, r.config)
	assert.False(controller0.updated)
	assert.False(controller1.updated)
	assert.EqualError(r.UpdateConfig(c.config), fmt.Sprint(controller1.updateError))
	assert.Equal(c.config, r.config)
	assert.True(controller0.updated)
	assert.False(controller1.updated)
	// different configs
	controller0.updated = false
	controller1.updated = false
	c.config.Name = "newName"
	assert.NotEqual(c.config, r.config)
	assert.False(controller0.updated)
	assert.False(controller1.updated)
	assert.EqualError(r.UpdateConfig(c.config), fmt.Sprint(controller1.updateError))
	assert.Equal(c.config, r.config)
	assert.True(controller0.updated)
	assert.False(controller1.updated)
	// different configs again but with already updated flag
	c.config.Name = "newSpecialName"
	assert.NotEqual(c.config, r.config)
	assert.True(controller0.updated)
	assert.False(controller1.updated)
	assert.EqualError(r.UpdateConfig(c.config), fmt.Sprint(controller1.updateError))
	assert.Equal(c.config, r.config)
	assert.True(controller0.updated)
	assert.False(controller1.updated)
	// different configs with empty config
	controller0.updated = false
	controller1.updated = false
	c.config = *new(v1alpha1.KarydiaConfig)
	assert.NotEqual(c.config, r.config)
	assert.False(controller0.updated)
	assert.False(controller1.updated)
	assert.EqualError(r.UpdateConfig(c.config), fmt.Sprint(controller1.updateError))
	assert.Equal(c.config, r.config)
	assert.True(controller0.updated)
	assert.False(controller1.updated)
}

func TestConfigReconciler_createConfig(t *testing.T) {
	// setup
	assert := assert.New(t)
	c := newTestConfig(t, "1", testConfigParams{
		automountServiceAccountToken:	"testAutomountServiceAccountToken",
		seccompProfile:					"testSeccompProfile",
		networkPolicy:					"testNetworkPolicy",
	})
	s := newTestSettings(t, []runtime.Object{}, []ControllerInterface{})
	r := NewConfigReconciler(c.config, s.controllers, s.clientset, s.configInformer)

	// no config deployed
	_, err := s.configWorker.Get(c.config.Name, metav1.GetOptions{})
	assert.True(errors.IsNotFound(err))
	// create config
	assert.NoError(r.createConfig())
	e, _ := s.configWorker.Get(c.config.Name, metav1.GetOptions{})
	assert.Equal(&c.config, e)
	// retry config creation
	assert.True(errors.IsAlreadyExists(r.createConfig()))
}

func TestConfigReconciler_RunWithoutStartConfig(t *testing.T) {
	// setup
	assert := assert.New(t)
	c := newTestConfig(t, "1", testConfigParams{
		automountServiceAccountToken:	"testAutomountServiceAccountToken",
		seccompProfile:					"testSeccompProfile",
		networkPolicy:					"testNetworkPolicy",
	})
	newC := newTestConfig(t, "2", testConfigParams{
		automountServiceAccountToken:	"testAutomountServiceAccountToken",
		seccompProfile:					"newValue",
		networkPolicy:					"testNetworkPolicy",
	})
	s := newTestSettings(t, []runtime.Object{}, []ControllerInterface{new(testController)})
	r := NewConfigReconciler(c.config, s.controllers, s.clientset, s.configInformer)
	controller := s.controllers[0].(*testController)

	// start channels
	ctx, cancelCtx := context.WithCancel(context.Background())
	defer cancelCtx()

	// start reconciler
	go func() {
		s.sharedInformerFactory.Start(ctx.Done())
		r.Run(1, ctx.Done())
	}()

	// no config deployed
	_, err := s.configWorker.Get(c.config.Name, metav1.GetOptions{})
	assert.True(errors.IsNotFound(err))
	s.sharedInformerFactory.WaitForCacheSync(ctx.Done())
	_, err = r.lister.Get(c.config.Name)
	assert.True(errors.IsNotFound(err))

	// try to update config
	_, err = s.configWorker.Update(&newC.config)
	assert.True(errors.IsNotFound(err))
	// but without cache update because operation failed
	s.sharedInformerFactory.WaitForCacheSync(ctx.Done())
	_, err = r.lister.Get(c.config.Name)
	assert.True(errors.IsNotFound(err))
	// and without informing methods because operation failed
	assert.Equal(c.config, r.config)
	assert.False(controller.updated)

	// try to delete config
	assert.True(errors.IsNotFound(s.configWorker.Delete(c.config.Name, &metav1.DeleteOptions{})))
	// but without cache update because operation failed
	s.sharedInformerFactory.WaitForCacheSync(ctx.Done())
	_, err = r.lister.Get(c.config.Name)
	assert.True(errors.IsNotFound(err))
	// and without informing methods because operation failed
	assert.Equal(c.config, r.config)
	assert.False(controller.updated)

	// create config
	_, err = s.configWorker.Create(&newC.config)
	assert.NoError(err)
	// with cache update
	s.sharedInformerFactory.WaitForCacheSync(ctx.Done())
	if err := wait.PollImmediate(1 * time.Millisecond, s.waitTimeoutSeconds, func()(bool, error) {
		_, err := r.lister.Get(c.config.Name)
		if errors.IsNotFound(err) {
			return false, nil
		}
		return true, nil
	}); err != nil {
		t.FailNow()
	}
	e, _ := r.lister.Get(c.config.Name)
	assert.Equal(&newC.config, e)
	// but without informing methods because operation not watched
	assert.Equal(c.config, r.config)
	assert.False(controller.updated)

	// stop channels
	cancelCtx()
}

func TestConfigReconciler_RunWithStartConfig(t *testing.T) {
	// setup
	assert := assert.New(t)
	c := newTestConfig(t, "1", testConfigParams{
		automountServiceAccountToken:	"testAutomountServiceAccountToken",
		seccompProfile:					"testSeccompProfile",
		networkPolicy:					"testNetworkPolicy",
	})
	newC := newTestConfig(t, "1", testConfigParams{
		automountServiceAccountToken:	"testAutomountServiceAccountToken",
		seccompProfile:					"testSeccompProfile",
		networkPolicy:					"newValue",
	})
	differentC := newTestConfig(t, "2", testConfigParams{
		automountServiceAccountToken:	"testAutomountServiceAccountToken",
		seccompProfile:					"testSeccompProfile",
		networkPolicy:					"differentValue",
	})
	s := newTestSettings(t, []runtime.Object{&c.config}, []ControllerInterface{new(testController)})
	r := NewConfigReconciler(c.config, s.controllers, s.clientset, s.configInformer)
	controller := s.controllers[0].(*testController)

	// start channels
	ctx, cancelCtx := context.WithCancel(context.Background())
	defer cancelCtx()

	// start reconciler
	go func() {
		s.sharedInformerFactory.Start(ctx.Done())
		r.Run(1, ctx.Done())
	}()

	// config deployed by default
	_, err := s.configWorker.Get(c.config.Name, metav1.GetOptions{})
	assert.NoError(err)

	// try to create config again
	_, err = s.configWorker.Create(&newC.config)
	assert.True(errors.IsAlreadyExists(err))
	// but without cache update because operation failed
	s.sharedInformerFactory.WaitForCacheSync(ctx.Done())
	if err := wait.PollImmediate(1 * time.Millisecond, s.waitTimeoutSeconds, func()(bool, error) {
		_, err := r.lister.Get(c.config.Name)
		if errors.IsNotFound(err) {
			return false, nil
		}
		return true, nil
	}); err != nil {
		t.FailNow()
	}
	e, _ := r.lister.Get(c.config.Name)
	assert.Equal(&c.config, e)
	// and without informing methods because operation failed and not watched
	assert.Equal(c.config, r.config)
	assert.False(controller.updated)

	// try to update config
	_, err = s.configWorker.Update(&newC.config)
	assert.NoError(err)
	// with cache update
	s.sharedInformerFactory.WaitForCacheSync(ctx.Done())
	e, _ = r.lister.Get(c.config.Name)
	assert.Equal(&newC.config, e)
	// but without informing methods because resource versions are equal
	assert.Equal(c.config, r.config)
	assert.False(controller.updated)

	// try to update config again with different resource versions
	_, err = s.configWorker.Update(&differentC.config)
	assert.NoError(err)
	// with cache update
	s.sharedInformerFactory.WaitForCacheSync(ctx.Done())
	e, _ = r.lister.Get(c.config.Name)
	assert.Equal(&differentC.config, e)
	// and informing methods
	assert.Equal(differentC.config, r.config)
	assert.True(controller.updated)

	// try to update config again with different resource versions but equal configs
	differentC.config.ResourceVersion = "3"
	controller.updated = false
	_, err = s.configWorker.Update(&differentC.config)
	assert.NoError(err)
	// with cache update
	s.sharedInformerFactory.WaitForCacheSync(ctx.Done())
	e, _ = r.lister.Get(c.config.Name)
	assert.Equal(&differentC.config, e)
	// and informing methods
	assert.NotEqual(differentC.config, r.config)
	assert.Equal(differentC.config.Spec, r.config.Spec)
	assert.False(controller.updated)

	// try to delete config
	differentC.config.ResourceVersion = "2"
	assert.NoError(s.configWorker.Delete(c.config.Name, &metav1.DeleteOptions{}))
	// with informing methods
	s.sharedInformerFactory.WaitForCacheSync(ctx.Done())
	e, _ = s.configWorker.Get(c.config.Name, metav1.GetOptions{})
	assert.Equal(&differentC.config, e)
	assert.Equal(differentC.config, r.config)
	assert.False(controller.updated)
	// and re-synced cache
	s.sharedInformerFactory.WaitForCacheSync(ctx.Done())
	if err := wait.PollImmediate(1 * time.Millisecond, s.waitTimeoutSeconds, func()(bool, error) {
		_, err := r.lister.Get(c.config.Name)
		if errors.IsNotFound(err) {
			return false, nil
		}
		return true, nil
	}); err != nil {
		t.FailNow()
	}
	e, _ = r.lister.Get(c.config.Name)
	assert.Equal(&differentC.config, e)

	// stop channels
	cancelCtx()
}

func TestConfigReconciler_RunWithStartConfigAndWorkingWithDifferentConfig(t *testing.T) {
	// setup
	assert := assert.New(t)
	c := newTestConfig(t, "1", testConfigParams{
		automountServiceAccountToken:	"testAutomountServiceAccountToken",
		seccompProfile:					"testSeccompProfile",
		networkPolicy:					"testNetworkPolicy",
	})
	differentC := newTestConfig(t, "2", testConfigParams{
		automountServiceAccountToken:	"differentAutomountServiceAccountToken",
		seccompProfile:					"differentSeccompProfile",
		networkPolicy:					"differentNetworkPolicy",
	})
	differentC.config.Name = "differentName"
	s := newTestSettings(t, []runtime.Object{&c.config}, []ControllerInterface{new(testController)})
	r := NewConfigReconciler(c.config, s.controllers, s.clientset, s.configInformer)
	controller := s.controllers[0].(*testController)

	// start channels
	ctx, cancelCtx := context.WithCancel(context.Background())
	defer cancelCtx()

	// start reconciler
	go func() {
		s.sharedInformerFactory.Start(ctx.Done())
		r.Run(1, ctx.Done())
	}()

	// config deployed by default
	_, err := s.configWorker.Get(c.config.Name, metav1.GetOptions{})
	assert.NoError(err)
	_, err = s.configWorker.Get(differentC.config.Name, metav1.GetOptions{})
	assert.True(errors.IsNotFound(err))

	// create different config
	_, err = s.configWorker.Create(&differentC.config)
	assert.NoError(err)
	e, err := s.configWorker.Get(differentC.config.Name, metav1.GetOptions{})
	assert.NoError(err)
	assert.Equal(&differentC.config, e)
	s.sharedInformerFactory.WaitForCacheSync(ctx.Done())
	if err := wait.PollImmediate(1 * time.Millisecond, s.waitTimeoutSeconds, func()(bool, error) {
		_, err := r.lister.Get(c.config.Name)
		if errors.IsNotFound(err) {
			return false, nil
		}
		return true, nil
	}); err != nil {
		t.FailNow()
	}
	if err := wait.PollImmediate(1 * time.Millisecond, s.waitTimeoutSeconds, func()(bool, error) {
		_, err := r.lister.Get(differentC.config.Name)
		if errors.IsNotFound(err) {
			return false, nil
		}
		return true, nil
	}); err != nil {
		t.FailNow()
	}
	e, _ = r.lister.Get(differentC.config.Name)
	assert.Equal(&differentC.config, e)
	// but without cache update because it is a different config
	e, _ = r.lister.Get(c.config.Name)
	assert.Equal(&c.config, e)
	// and without informing methods because it is a different config and operation not watched
	assert.Equal(c.config, r.config)
	assert.False(controller.updated)

	// update config
	differentC.config.ResourceVersion = "3"
	differentC.config.Spec.AutomountServiceAccountToken = "changedValue"
	_, err = s.configWorker.Update(&differentC.config)
	assert.NoError(err)
	e, err = s.configWorker.Get(differentC.config.Name, metav1.GetOptions{})
	assert.NoError(err)
	assert.Equal(&differentC.config, e)
	s.sharedInformerFactory.WaitForCacheSync(ctx.Done())
	e, _ = r.lister.Get(differentC.config.Name)
	assert.Equal(&differentC.config, e)
	// but without cache update because it is a different config
	e, _ = r.lister.Get(c.config.Name)
	assert.Equal(&c.config, e)
	// and without informing methods because it is a different config
	assert.Equal(c.config, r.config)
	assert.False(controller.updated)

	// delete config
	assert.NoError(s.configWorker.Delete(differentC.config.Name, &metav1.DeleteOptions{}))
	_, err = s.configWorker.Get(differentC.config.Name, metav1.GetOptions{})
	assert.True(errors.IsNotFound(err))
	s.sharedInformerFactory.WaitForCacheSync(ctx.Done())
	_, err = r.lister.Get(differentC.config.Name)
	assert.True(errors.IsNotFound(err))
	// but without cache update because it is a different config
	e, _ = r.lister.Get(c.config.Name)
	assert.Equal(&c.config, e)
	// and without informing methods because it is a different config
	assert.Equal(c.config, r.config)
	assert.False(controller.updated)

	// stop channels
	cancelCtx()
}
