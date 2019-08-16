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

package karydia

import (
	"encoding/json"
	"testing"

	jsonpatch "github.com/evanphx/json-patch"
	corev1 "k8s.io/api/core/v1"
)

/* Mutating and Validating Webhook
 * Changes the default of automountServiceToken for the default service accounts
 * from true to false when undefined.
 * kubectl annotate ns default karydia.gardener.cloud/automountServiceAccountToken=change-default
 */
func TestServiceAccountChangeDefaultAnnotationDefaultServiceAccountMountUndefined(t *testing.T) {
	var patches Patches
	var validationErrors []string

	sAcc := corev1.ServiceAccount{}
	sAcc.Name = "default"

	setting := Setting{value: "change-default", src: "namespace"}
	patches = mutateServiceAccountTokenMount(sAcc, setting, patches)
	if len(patches.operations) != 2 {
		t.Error("expected 2 patches but got:", patches.operations)
	}
	mutatedServiceAccount, err := patchServiceAccount(sAcc, patches)
	if err != nil {
		t.Error("failed to apply patches:", err)
	}
	// Zero validation errors expected for mutated pod
	validationErrors = validateServiceAccountTokenMount(mutatedServiceAccount, setting, validationErrors)
	if len(validationErrors) != 0 {
		t.Error("expected 0 validationErrors but got:", validationErrors)
	}
	validationErrors = []string{}
	// One validation error expected for initial pod
	validationErrors = validateServiceAccountTokenMount(sAcc, setting, validationErrors)
	if len(validationErrors) != 1 {
		t.Error("expected 1 validationErrors but got:", validationErrors)
	}
}

func TestServiceAccountChangeDefaultAnnotationDefaultServiceAccountMountFalse(t *testing.T) {
	var patches Patches
	var validationErrors []string
	var automount = false

	sAcc := corev1.ServiceAccount{}
	sAcc.Name = "default"
	sAcc.AutomountServiceAccountToken = &(automount)

	setting := Setting{value: "change-default", src: "namespace"}

	patches = mutateServiceAccountTokenMount(sAcc, setting, patches)
	if len(patches.operations) != 0 {
		t.Error("expected 0 patches but got:", patches.operations)
	}
	mutatedServiceAccount, err := patchServiceAccount(sAcc, patches)
	if err != nil {
		t.Error("failed to apply patches:", err)
	}
	// Zero validation errors expected for mutated pod
	validationErrors = validateServiceAccountTokenMount(mutatedServiceAccount, setting, validationErrors)
	if len(validationErrors) != 0 {
		t.Error("expected 0 validationErrors but got:", validationErrors)
	}
	validationErrors = []string{}
	// Zero validation errors expected for mutated pod
	validationErrors = validateServiceAccountTokenMount(sAcc, setting, validationErrors)
	if len(validationErrors) != 0 {
		t.Error("expected 0 validationErrors but got:", validationErrors)
	}
}

func TestServiceAccountChangeDefaultAnnotationDefaultServiceAccountMountTrue(t *testing.T) {
	var patches Patches
	var validationErrors []string
	var automount = true

	sAcc := corev1.ServiceAccount{}
	sAcc.Name = "default"
	sAcc.AutomountServiceAccountToken = &(automount)

	setting := Setting{value: "change-default", src: "namespace"}

	patches = mutateServiceAccountTokenMount(sAcc, setting, patches)
	if len(patches.operations) != 0 {
		t.Error("expected 0 patches but got:", patches.operations)
	}
	mutatedServiceAccount, err := patchServiceAccount(sAcc, patches)
	if err != nil {
		t.Error("failed to apply patches:", err)
	}
	// Zero validation errors expected for mutated pod
	validationErrors = validateServiceAccountTokenMount(mutatedServiceAccount, setting, validationErrors)
	if len(validationErrors) != 0 {
		t.Error("expected 0 validationErrors but got:", validationErrors)
	}
	validationErrors = []string{}
	// Zero validation errors expected for mutated pod
	validationErrors = validateServiceAccountTokenMount(sAcc, setting, validationErrors)
	if len(validationErrors) != 0 {
		t.Error("expected 0 validationErrors but got:", validationErrors)
	}
}

func TestServiceAccountChangeDefaultAnnotationSpecificServiceAccountMountUndefined(t *testing.T) {
	var patches Patches
	var validationErrors []string

	sAcc := corev1.ServiceAccount{}
	sAcc.Name = "specific"

	setting := Setting{value: "change-default", src: "namespace"}

	patches = mutateServiceAccountTokenMount(sAcc, setting, patches)
	if len(patches.operations) != 0 {
		t.Error("expected 0 patches but got:", patches.operations)
	}
	mutatedServiceAccount, err := patchServiceAccount(sAcc, patches)
	if err != nil {
		t.Error("failed to apply patches:", err)
	}
	// Zero validation errors expected for mutated pod
	validationErrors = validateServiceAccountTokenMount(mutatedServiceAccount, setting, validationErrors)
	if len(validationErrors) != 0 {
		t.Error("expected 0 validationErrors but got:", validationErrors)
	}
	validationErrors = []string{}
	// Zero validation errors expected for mutated pod
	validationErrors = validateServiceAccountTokenMount(sAcc, setting, validationErrors)
	if len(validationErrors) != 0 {
		t.Error("expected 0 validationErrors but got:", validationErrors)
	}
}
func TestServiceAccountRandomAnnotationDefaultServiceAccountMountUndefined(t *testing.T) {
	var patches Patches
	var validationErrors []string

	sAcc := corev1.ServiceAccount{}
	sAcc.Name = "default"

	setting := Setting{value: "random", src: "namespace"}

	patches = mutateServiceAccountTokenMount(sAcc, setting, patches)
	if len(patches.operations) != 0 {
		t.Error("expected 0 patches but got:", patches.operations)
	}
	mutatedServiceAccount, err := patchServiceAccount(sAcc, patches)
	if err != nil {
		t.Error("failed to apply patches:", err)
	}
	// Zero validation errors expected for mutated pod
	validationErrors = validateServiceAccountTokenMount(mutatedServiceAccount, setting, validationErrors)
	if len(validationErrors) != 0 {
		t.Error("expected 0 validationErrors but got:", validationErrors)
	}
	validationErrors = []string{}
	// Zero validation errors expected for mutated pod
	validationErrors = validateServiceAccountTokenMount(sAcc, setting, validationErrors)
	if len(validationErrors) != 0 {
		t.Error("expected 0 validationErrors but got:", validationErrors)
	}
}

/* Mutating and Validating Webhook
 * Changes the default of automountServiceToken for all service accounts
 * from true to false when undefined.
 * kubectl annotate ns default karydia.gardener.cloud/automountServiceAccountToken=change-all
 */
func TestServiceAccountChangeAllAnnotationDefaultServiceAccountMountUndefined(t *testing.T) {
	var patches Patches
	var validationErrors []string

	sAcc := corev1.ServiceAccount{}
	sAcc.Name = "default"

	setting := Setting{value: "change-all", src: "namespace"}

	patches = mutateServiceAccountTokenMount(sAcc, setting, patches)
	if len(patches.operations) != 2 {
		t.Error("expected 2 patches but got:", patches.operations)
	}
	mutatedServiceAccount, err := patchServiceAccount(sAcc, patches)
	if err != nil {
		t.Error("failed to apply patches:", err)
	}
	// Zero validation errors expected for mutated pod
	validationErrors = validateServiceAccountTokenMount(mutatedServiceAccount, setting, validationErrors)
	if len(validationErrors) != 0 {
		t.Error("expected 0 validationErrors but got:", validationErrors)
	}
	validationErrors = []string{}
	// One validation error expected for initial pod
	validationErrors = validateServiceAccountTokenMount(sAcc, setting, validationErrors)
	if len(validationErrors) != 1 {
		t.Error("expected 1 validationErrors but got:", validationErrors)
	}
}

func TestServiceAccountChangeAllAnnotationSpecificServiceAccountMountUndefined(t *testing.T) {
	var patches Patches
	var validationErrors []string

	sAcc := corev1.ServiceAccount{}
	sAcc.Name = "specific"

	setting := Setting{value: "change-all", src: "namespace"}

	patches = mutateServiceAccountTokenMount(sAcc, setting, patches)
	if len(patches.operations) != 2 {
		t.Error("expected 2 patches but got:", patches.operations)
	}
	mutatedServiceAccount, err := patchServiceAccount(sAcc, patches)
	if err != nil {
		t.Error("failed to apply patches:", err)
	}
	// Zero validation errors expected for mutated pod
	validationErrors = validateServiceAccountTokenMount(mutatedServiceAccount, setting, validationErrors)
	if len(validationErrors) != 0 {
		t.Error("expected 0 validationErrors but got:", validationErrors)
	}
	validationErrors = []string{}
	// One validation error expected for initial pod
	validationErrors = validateServiceAccountTokenMount(sAcc, setting, validationErrors)
	if len(validationErrors) != 1 {
		t.Error("expected 1 validationErrors but got:", validationErrors)
	}
}

func TestServiceAccountChangeAllAnnotationSpecificServiceAccountMountFalse(t *testing.T) {
	var patches Patches
	var validationErrors []string
	var automount = false

	sAcc := corev1.ServiceAccount{}
	sAcc.Name = "specific"
	sAcc.AutomountServiceAccountToken = &(automount)

	setting := Setting{value: "change-all", src: "namespace"}

	patches = mutateServiceAccountTokenMount(sAcc, setting, patches)
	if len(patches.operations) != 0 {
		t.Error("expected 0 patches but got:", patches.operations)
	}
	mutatedServiceAccount, err := patchServiceAccount(sAcc, patches)
	if err != nil {
		t.Error("failed to apply patches:", err)
	}
	// Zero validation errors expected for mutated pod
	validationErrors = validateServiceAccountTokenMount(mutatedServiceAccount, setting, validationErrors)
	if len(validationErrors) != 0 {
		t.Error("expected 0 validationErrors but got:", validationErrors)
	}
	validationErrors = []string{}
	// Zero validation errors expected for mutated pod
	validationErrors = validateServiceAccountTokenMount(sAcc, setting, validationErrors)
	if len(validationErrors) != 0 {
		t.Error("expected 0 validationErrors but got:", validationErrors)
	}
}

func TestServiceAccountChangeAllAnnotationspecificServiceAccountMountTrue(t *testing.T) {
	var patches Patches
	var validationErrors []string
	var automount = true

	sAcc := corev1.ServiceAccount{}
	sAcc.Name = "specific"
	sAcc.AutomountServiceAccountToken = &(automount)

	setting := Setting{value: "change-all", src: "namespace"}

	patches = mutateServiceAccountTokenMount(sAcc, setting, patches)
	if len(patches.operations) != 0 {
		t.Error("expected 0 patches but got:", patches.operations)
	}
	mutatedServiceAccount, err := patchServiceAccount(sAcc, patches)
	if err != nil {
		t.Error("failed to apply patches:", err)
	}
	// Zero validation errors expected for mutated pod
	validationErrors = validateServiceAccountTokenMount(mutatedServiceAccount, setting, validationErrors)
	if len(validationErrors) != 0 {
		t.Error("expected 0 validationErrors but got:", validationErrors)
	}
	validationErrors = []string{}
	// Zero validation errors expected for mutated pod
	validationErrors = validateServiceAccountTokenMount(sAcc, setting, validationErrors)
	if len(validationErrors) != 0 {
		t.Error("expected 0 validationErrors but got:", validationErrors)
	}
}

func TestServiceAccountRandomAnnotationSpecificServiceAccountMountUndefined(t *testing.T) {
	var patches Patches
	var validationErrors []string

	sAcc := corev1.ServiceAccount{}
	sAcc.Name = "specific"

	setting := Setting{value: "random", src: "namespace"}

	patches = mutateServiceAccountTokenMount(sAcc, setting, patches)
	if len(patches.operations) != 0 {
		t.Error("expected 0 patches but got:", patches.operations)
	}
	mutatedServiceAccount, err := patchServiceAccount(sAcc, patches)
	if err != nil {
		t.Error("failed to apply patches:", err)
	}
	// Zero validation errors expected for mutated pod
	validationErrors = validateServiceAccountTokenMount(mutatedServiceAccount, setting, validationErrors)
	if len(validationErrors) != 0 {
		t.Error("expected 0 validationErrors but got:", validationErrors)
	}
	validationErrors = []string{}
	// Zero validation errors expected for mutated pod
	validationErrors = validateServiceAccountTokenMount(sAcc, setting, validationErrors)
	if len(validationErrors) != 0 {
		t.Error("expected 0 validationErrors but got:", validationErrors)
	}
}

/* Helper functions */
func patchPod(pod corev1.Pod, patches Patches) (corev1.Pod, error) {
	var podJSON []byte
	podJSON, err := json.Marshal(&pod)
	if err != nil {
		return pod, err
	}

	patchObj, err := jsonpatch.DecodePatch(patches.toBytes())
	if err != nil {
		return pod, err
	}
	podPatchedJSON, err := patchObj.Apply(podJSON)
	if err != nil {
		return pod, err
	}

	var podPatched corev1.Pod
	json.Unmarshal(podPatchedJSON, &podPatched)

	return podPatched, nil
}

func patchServiceAccount(sAcc corev1.ServiceAccount, patches Patches) (corev1.ServiceAccount, error) {
	var podJSON []byte
	podJSON, err := json.Marshal(&sAcc)
	if err != nil {
		return sAcc, err
	}

	patchObj, err := jsonpatch.DecodePatch(patches.toBytes())
	if err != nil {
		return sAcc, err
	}
	sAccPatchedJSON, err := patchObj.Apply(podJSON)
	if err != nil {
		return sAcc, err
	}

	var sAccPatched corev1.ServiceAccount
	json.Unmarshal(sAccPatchedJSON, &sAccPatched)

	return sAccPatched, nil
}
