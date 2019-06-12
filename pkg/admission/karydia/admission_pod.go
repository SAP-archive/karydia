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
	"strings"

	"k8s.io/api/admission/v1beta1"
	corev1 "k8s.io/api/core/v1"

	"github.com/karydia/karydia/pkg/k8sutil"
	"github.com/karydia/karydia/pkg/k8sutil/scheme"
)

func (k *KarydiaAdmission) mutatePod(pod *corev1.Pod, ns *corev1.Namespace) *v1beta1.AdmissionResponse {
	var patches Patches

	setting := k.getSeccompProfileSetting(ns)
	if setting.value != "" {
		patches = mutatePodSeccompProfile(*pod, setting, patches)
	}
	setting = k.getSecurityContextSetting(ns)
	if setting.value != "" {
		patches = mutatePodSecurityContext(*pod, setting, patches)
	}
	return k8sutil.MutatingAdmissionResponse(patches.toBytes())
}

func (k *KarydiaAdmission) validatePod(pod *corev1.Pod, ns *corev1.Namespace) *v1beta1.AdmissionResponse {
	var validationErrors []string

	setting := k.getSeccompProfileSetting(ns)
	if setting.value != "" {
		validationErrors = validatePodSeccompProfile(*pod, setting, validationErrors)
	}

	return k8sutil.ValidatingAdmissionResponse(validationErrors)
}

func (k *KarydiaAdmission) getSeccompProfileSetting(ns *corev1.Namespace) Setting {
	seccompProfile, annotated := ns.ObjectMeta.Annotations["karydia.gardener.cloud/seccompProfile"]
	src := "namespace"
	if !annotated {
		seccompProfile = k.karydiaConfig.Spec.SeccompProfile
		src = "config"
	}
	return Setting{value: seccompProfile, src: src}
}

func (k *KarydiaAdmission) getSecurityContextSetting(ns *corev1.Namespace) Setting {
	seccompProfile, annotated := ns.ObjectMeta.Annotations["karydia.gardener.cloud/podSecurityContext"]
	src := "namespace"
	if !annotated {
		seccompProfile = k.karydiaConfig.Spec.PodSecurityContext
		src = "config"
	}
	return Setting{value: seccompProfile, src: src}
}

func validatePodSeccompProfile(pod corev1.Pod, setting Setting, validationErrors []string) []string {
	_, ok := pod.ObjectMeta.Annotations["seccomp.security.alpha.kubernetes.io/pod"]
	if !ok {
		validationErrorMsg := fmt.Sprintf("seccomp profile ('seccomp.security.alpha.kubernetes.io/pod') must be '%s'", setting.value)
		validationErrors = append(validationErrors, validationErrorMsg)
	}
	return validationErrors
}

func mutatePodSeccompProfile(pod corev1.Pod, setting Setting, patches Patches) Patches {
	_, ok := pod.ObjectMeta.Annotations["seccomp.security.alpha.kubernetes.io/pod"]
	if !ok {
		annotatePod(pod, &patches, "seccomp.security.alpha.kubernetes.io/pod", setting.value)
		annotatePod(pod, &patches, "karydia.gardener.cloud/seccompProfile.internal", setting.src+"/"+setting.value)
	}
	return patches
}

func mutatePodSecurityContext(pod corev1.Pod, setting Setting, patches Patches) Patches {
	var uid int64 = 65534
	var gid int64 = 65534
	setPodSecurityContext(pod, &patches, &uid, &gid)
	annotatePod(pod, &patches, "karydia.gardener.cloud/podSecurityContext.internal", setting.src+"/"+setting.value)

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

func annotatePod(resource corev1.Pod, patches *Patches, key string, value string) {
	if len(resource.ObjectMeta.Annotations) == 0 && !patches.annotated {
		patches.operations = append(patches.operations, patchOperation{
			Op:   "add",
			Path: "/metadata/annotations",
			Value: map[string]string{
				key: value,
			},
		})
		patches.annotated = true
	} else {
		patches.operations = append(patches.operations, patchOperation{
			Op:    "add",
			Path:  "/metadata/annotations/" + strings.Replace(key, "/", "~1", -1),
			Value: value,
		})
	}
}

func setPodSecurityContext(resource corev1.Pod, patches *Patches, uid *int64, gid *int64) {
	if resource.Spec.SecurityContext == nil {
		patches.operations = append(patches.operations, patchOperation{
			Op:   "add",
			Path: "/spec/securitycontext",
			Value: corev1.SecurityContext{
				RunAsUser:  uid,
				RunAsGroup: gid,
			},
		})
	}
}
