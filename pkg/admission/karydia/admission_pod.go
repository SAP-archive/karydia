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

func (k *KarydiaAdmission) mutatePod(pod *corev1.Pod, ns *corev1.Namespace) *v1beta1.AdmissionResponse {
	var patches patchOperations

	seccompProfile := k.getSeccompProfileSetting(ns)
	if seccompProfile != "" {
		patches = mutatePodSeccompProfile(*pod, seccompProfile, patches)
	}
	return k8sutil.MutatingAdmissionResponse(patches.toBytes())
}

func (k *KarydiaAdmission) validatePod(pod *corev1.Pod, ns *corev1.Namespace) *v1beta1.AdmissionResponse {
	var validationErrors []string

	seccompProfile := k.getSeccompProfileSetting(ns)
	if seccompProfile != "" {
		validationErrors = validatePodSeccompProfile(*pod, seccompProfile, validationErrors)
	}

	return k8sutil.ValidatingAdmissionResponse(validationErrors)
}

func (k *KarydiaAdmission) getSeccompProfileSetting(ns *corev1.Namespace) string {
	seccompProfile, annotated := ns.ObjectMeta.Annotations["karydia.gardener.cloud/seccompProfile"]
	if !annotated {
		seccompProfile = k.karydiaConfig.Spec.SeccompProfile
	}
	return seccompProfile
}

func validatePodSeccompProfile(pod corev1.Pod, nsAnnotation string, validationErrors []string) []string {
	_, ok := pod.ObjectMeta.Annotations["seccomp.security.alpha.kubernetes.io/pod"]
	if !ok {
		validationErrorMsg := fmt.Sprintf("seccomp profile ('seccomp.security.alpha.kubernetes.io/pod') must be '%s'", nsAnnotation)
		validationErrors = append(validationErrors, validationErrorMsg)
	}
	return validationErrors
}

func mutatePodSeccompProfile(pod corev1.Pod, nsAnnotation string, patches patchOperations) patchOperations {
	_, ok := pod.ObjectMeta.Annotations["seccomp.security.alpha.kubernetes.io/pod"]
	if !ok {
		if len(pod.ObjectMeta.Annotations) == 0 {
			// If no annotation object exists yet, we have
			// to create it. Otherwise we will encounter
			// the following error:
			// 'jsonpatch add operation does not apply: doc is missing path: "/metadata/annotations..."'
			patches = append(patches, patchOperation{
				Op:   "add",
				Path: "/metadata/annotations",
				Value: map[string]string{
					"seccomp.security.alpha.kubernetes.io/pod": nsAnnotation,
				},
			})
		} else {
			patches = append(patches, patchOperation{
				Op:    "add",
				Path:  "/metadata/annotations/seccomp.security.alpha.kubernetes.io~1pod",
				Value: nsAnnotation,
			})
		}
	}
	return patches
}

/* Utility functions to decode raw resources into objects */
func decodePod(raw []byte) (*corev1.Pod, error) {
	pod := &corev1.Pod{}
	deserializer := scheme.Codecs.UniversalDeserializer()
	if _, _, err := deserializer.Decode(raw, nil, pod); err != nil {
		return nil, err
	}
	return pod, nil
}
