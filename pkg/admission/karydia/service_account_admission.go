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
	"fmt"

	"k8s.io/api/admission/v1beta1"
	corev1 "k8s.io/api/core/v1"

	"github.com/karydia/karydia/pkg/k8sutil"
	"github.com/karydia/karydia/pkg/k8sutil/scheme"
)

func (k *KarydiaAdmission) mutateServiceAccount(sAcc *corev1.ServiceAccount, ns *corev1.Namespace) *v1beta1.AdmissionResponse {
	var patches []string

	automountServiceAccountToken, annotated := ns.ObjectMeta.Annotations["karydia.gardener.cloud/automountServiceAccountToken"]
	if annotated {
		patches = mutateServiceAccountTokenMount(*sAcc, automountServiceAccountToken, patches)
	}

	return k8sutil.MutatingAdmissionResponse(patches)
}

func (k *KarydiaAdmission) validateServiceAccount(sAcc *corev1.ServiceAccount, ns *corev1.Namespace) *v1beta1.AdmissionResponse {
	var validationErrors []string

	automountServiceAccountToken, annotated := ns.ObjectMeta.Annotations["karydia.gardener.cloud/automountServiceAccountToken"]
	if annotated {
		validationErrors = validateServiceAccountTokenMount(*sAcc, automountServiceAccountToken, validationErrors)
	}

	return k8sutil.ValidatingAdmissionResponse(validationErrors)
}

func validateServiceAccountTokenMount(sAcc corev1.ServiceAccount, nsAnnotation string, validationErrors []string) []string {
	if nsAnnotation == "change-default" {
		if automountServiceAccountTokenUndefined(&sAcc) && sAcc.Name == "default" {
			validationErrors = append(validationErrors, "implicit automount of default service account token not allowed")
		}
	} else if nsAnnotation == "change-all" {
		if automountServiceAccountTokenUndefined(&sAcc) {
			validationErrors = append(validationErrors, "implicit automount of service account token not allowed")
		}
	}
	return validationErrors
}

func mutateServiceAccountTokenMount(sAcc corev1.ServiceAccount, nsAnnotation string, patches []string) []string {
	if nsAnnotation == "change-default" {
		if automountServiceAccountTokenUndefined(&sAcc) && sAcc.Name == "default" {
			patches = append(patches, fmt.Sprintf(`{"op": "add", "path": "/automountServiceAccountToken", "value": %s}`, "false"))
		}
	} else if nsAnnotation == "change-all" {
		if automountServiceAccountTokenUndefined(&sAcc) {
			patches = append(patches, fmt.Sprintf(`{"op": "add", "path": "/automountServiceAccountToken", "value": %s}`, "false"))
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
