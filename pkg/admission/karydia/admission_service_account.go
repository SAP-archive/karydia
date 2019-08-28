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
	"strings"

	"k8s.io/api/admission/v1beta1"
	corev1 "k8s.io/api/core/v1"

	"github.com/karydia/karydia/pkg/k8sutil"
	"github.com/karydia/karydia/pkg/k8sutil/scheme"
)

func (k *KarydiaAdmission) mutateServiceAccount(sAcc *corev1.ServiceAccount, ns *corev1.Namespace) *v1beta1.AdmissionResponse {
	var patches Patches

	setting := k.getAutomountServiceAccountTokenSetting(ns)
	if setting.value != "" {
		patches = mutateServiceAccountTokenMount(*sAcc, setting, patches)
	}

	return k8sutil.MutatingAdmissionResponse(patches.toBytes())
}

func (k *KarydiaAdmission) validateServiceAccount(sAcc *corev1.ServiceAccount, ns *corev1.Namespace) *v1beta1.AdmissionResponse {
	var validationErrors []string

	setting := k.getAutomountServiceAccountTokenSetting(ns)
	if setting.value != "" {
		validationErrors = validateServiceAccountTokenMount(*sAcc, setting, validationErrors)
	}

	return k8sutil.ValidatingAdmissionResponse(validationErrors)
}

func (k *KarydiaAdmission) getAutomountServiceAccountTokenSetting(ns *corev1.Namespace) Setting {
	automountServiceAccountToken, annotated := ns.ObjectMeta.Annotations["karydia.gardener.cloud/automountServiceAccountToken"]
	src := "namespace"
	if !annotated || (k.karydiaConfig != nil && k.karydiaConfig.Spec.Enforcement == true) {
		automountServiceAccountToken = k.karydiaConfig.Spec.AutomountServiceAccountToken
		src = "config"
	}
	return Setting{value: automountServiceAccountToken, src: src}
}

func validateServiceAccountTokenMount(sAcc corev1.ServiceAccount, setting Setting, validationErrors []string) []string {
	if setting.value == "change-default" {
		if automountServiceAccountTokenUndefined(&sAcc) && sAcc.Name == "default" {
			validationErrors = append(validationErrors, "implicit automount of default service account token not allowed")
		}
	} else if setting.value == "change-all" {
		if automountServiceAccountTokenUndefined(&sAcc) {
			validationErrors = append(validationErrors, "implicit automount of service account token not allowed")
		}
	}
	return validationErrors
}

func mutateServiceAccountTokenMount(sAcc corev1.ServiceAccount, setting Setting, patches Patches) Patches {
	if setting.value == "change-default" {
		if automountServiceAccountTokenUndefined(&sAcc) && sAcc.Name == "default" {
			patches.operations = append(patches.operations, patchOperation{Op: "add", Path: "/automountServiceAccountToken", Value: false})
			patches.operations = append(patches.operations, annotateServiceAccount(sAcc, "karydia.gardener.cloud/automountServiceAccountToken.internal", setting.src+"/"+setting.value))
		}
	} else if setting.value == "change-all" {
		if automountServiceAccountTokenUndefined(&sAcc) {
			patches.operations = append(patches.operations, patchOperation{Op: "add", Path: "/automountServiceAccountToken", Value: false})
			patches.operations = append(patches.operations, annotateServiceAccount(sAcc, "karydia.gardener.cloud/automountServiceAccountToken.internal", setting.src+"/"+setting.value))
		}
	}
	return patches
}

func automountServiceAccountTokenUndefined(sAcc *corev1.ServiceAccount) bool {
	return sAcc.AutomountServiceAccountToken == nil
}

/* Utility functions to decode raw resources into objects */
func decodeServiceAccount(raw []byte) (*corev1.ServiceAccount, error) {
	sAcc := &corev1.ServiceAccount{}
	deserializer := scheme.Codecs.UniversalDeserializer()
	if _, _, err := deserializer.Decode(raw, nil, sAcc); err != nil {
		return nil, err
	}
	return sAcc, nil
}

func annotateServiceAccount(resource corev1.ServiceAccount, key string, value string) patchOperation {
	if len(resource.ObjectMeta.Annotations) == 0 {
		return patchOperation{
			Op:   "add",
			Path: "/metadata/annotations",
			Value: map[string]string{
				key: value,
			},
		}
	}
	return patchOperation{
		Op:    "add",
		Path:  "/metadata/annotations/" + strings.Replace(key, "/", "~1", -1),
		Value: value,
	}
}
