// Copyright 2019 Copyright (c) 2019 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file.
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

package karydiasecuritypolicy

import (
	"fmt"
	"strings"

	"k8s.io/api/admission/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/karydia/karydia/pkg/apis/karydia/v1alpha1"
	"github.com/karydia/karydia/pkg/k8sutil"
	"github.com/karydia/karydia/pkg/k8sutil/scheme"
)

func (k *KarydiaSecurityPolicyAdmission) computeSecurityContextPod(ar v1beta1.AdmissionReview, mutationAllowed bool, policies []*v1alpha1.KarydiaSecurityPolicy) *v1beta1.AdmissionResponse {
	if ar.Request.Operation != v1beta1.Create {
		return &v1beta1.AdmissionResponse{
			Allowed: true,
		}
	}

	raw := ar.Request.Object.Raw
	pod := corev1.Pod{}
	deserializer := scheme.Codecs.UniversalDeserializer()
	if _, _, err := deserializer.Decode(raw, nil, &pod); err != nil {
		return k8sutil.ErrToAdmissionResponse(err)
	}

	// b, _ := json.MarshalIndent(pod, "", "  ")
	// fmt.Printf("%s", b)

	validationErrors := make(map[string][]string)

	var acceptedWithPatches *v1beta1.AdmissionResponse

	for _, policy := range policies {
		policyName := policy.ObjectMeta.Name

		patches, violations := validatePod(policy, &pod)
		if len(violations) > 0 {
			validationErrors[policyName] = append(validationErrors[policyName], violations...)
			continue
		}

		if len(patches) > 0 && !mutationAllowed {
			continue
		}

		if len(patches) == 0 {
			// Valid without mutation
			return &v1beta1.AdmissionResponse{
				Allowed: true,
			}
		}

		if mutationAllowed && acceptedWithPatches == nil {
			patchesStr := strings.Join(patches, ",")
			patchType := v1beta1.PatchTypeJSONPatch

			acceptedWithPatches = &v1beta1.AdmissionResponse{
				Allowed:   true,
				Patch:     []byte(fmt.Sprintf("[%s]", patchesStr)),
				PatchType: &patchType,
			}

			// Continue to see if other policy allows the
			// pod without modification, which would be
			// preferred
		}
	}

	if acceptedWithPatches != nil {
		return acceptedWithPatches
	}

	return &v1beta1.AdmissionResponse{
		Allowed: false,
		Result: &metav1.Status{
			Message: fmt.Sprintf("%+v", validationErrors),
		},
	}
}

// validatePod returns patches and validation errors. If any
// validation error is returned, the patches are not considered.
func validatePod(policy *v1alpha1.KarydiaSecurityPolicy, pod *corev1.Pod) ([]string, []string) {
	var patches, validationErrors []string

	if policy.Spec.Pod.AutomountServiceAccountToken == "forbidden" {
		if doesAutomountServiceAccountToken(pod) {
			//validationErrors = append(validationErrors, "automount of service account not allowed")
			patches = append(patches, fmt.Sprintf(`{"op": "add", "path": "/spec/automountServiceAccountToken", "value": "%s"}`, "false"))
		}
	} else if policy.Spec.Pod.AutomountServiceAccountToken == "non-default" {
		if doesAutomountServiceAccountToken(pod) && pod.Spec.ServiceAccountName == "default" {
			//validationErrors = append(validationErrors, "automount of service account 'default' not allowed")
			patches = append(patches, fmt.Sprintf(`{"op": "add", "path": "/spec/automountServiceAccountToken", "value": "%s"}`, "false"))
		}
	}

	seccompProfile := policy.Spec.Pod.SeccompProfile
	if seccompProfile != "" {
		seccompPod, ok := pod.ObjectMeta.Annotations["seccomp.security.alpha.kubernetes.io/pod"]
		if !ok {
			patches = append(patches, fmt.Sprintf(`{"op": "add", "path": "/metadata/annotations/%s", "value": "%s"}`, "seccomp.security.alpha.kubernetes.io~1pod", seccompProfile))
		} else if seccompPod != seccompProfile {
			validationErrorMsg := fmt.Sprintf("seccomp profile ('seccomp.security.alpha.kubernetes.io/pod') must be '%s'", seccompProfile)
			validationErrors = append(validationErrors, validationErrorMsg)
		}
	}

	return patches, validationErrors
}

func doesAutomountServiceAccountToken(pod *corev1.Pod) bool {
	return pod.Spec.AutomountServiceAccountToken == nil || *pod.Spec.AutomountServiceAccountToken
}
