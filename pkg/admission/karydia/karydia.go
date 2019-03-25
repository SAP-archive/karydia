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

package karydia

import (
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
	"k8s.io/api/admission/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/karydia/karydia/pkg/k8sutil"
	"github.com/karydia/karydia/pkg/k8sutil/scheme"
)

var (
	resourcePod = metav1.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}
	kindPod     = metav1.GroupVersionKind{Group: "", Version: "v1", Kind: "Pod"}
)

type KarydiaAdmission struct {
	logger *logrus.Logger

	kubeClientset *kubernetes.Clientset
}

type Config struct {
	KubeClientset *kubernetes.Clientset
}

func New(config *Config) (*KarydiaAdmission, error) {
	logger := logrus.New()
	logger.Level = logrus.InfoLevel

	return &KarydiaAdmission{
		logger: logger,

		kubeClientset: config.KubeClientset,
	}, nil
}

func (k *KarydiaAdmission) Admit(ar v1beta1.AdmissionReview, mutationAllowed bool) *v1beta1.AdmissionResponse {
	if ignore, err := shouldIgnore(ar); err != nil {
		k.logger.Errorf("failed to determine if admission request should be ignored: %v", err)
		return &v1beta1.AdmissionResponse{
			Allowed: true,
		}
	} else if ignore {
		return &v1beta1.AdmissionResponse{
			Allowed: true,
		}
	}

	var response *v1beta1.AdmissionResponse

	if ar.Request.Kind == kindPod && ar.Request.Resource == resourcePod {
		response = k.AdmitPod(ar, mutationAllowed)
	} else {
		response = &v1beta1.AdmissionResponse{
			Allowed: true,
		}
	}

	return response
}

func (k *KarydiaAdmission) AdmitPod(ar v1beta1.AdmissionReview, mutationAllowed bool) *v1beta1.AdmissionResponse {
	var patches, validationErrors []string

	raw := ar.Request.Object.Raw
	pod := corev1.Pod{}
	deserializer := scheme.Codecs.UniversalDeserializer()
	if _, _, err := deserializer.Decode(raw, nil, &pod); err != nil {
		e := fmt.Errorf("failed to decode object: %v", err)
		k.logger.Errorf("%v", e)
		return k8sutil.ErrToAdmissionResponse(e)
	}

	namespaceRequest := ar.Request.Namespace
	if namespaceRequest == "" {
		e := fmt.Errorf("received request with empty namespace")
		k.logger.Errorf("%v", e)
		return k8sutil.ErrToAdmissionResponse(e)
	}
	namespace, err := k.kubeClientset.CoreV1().Namespaces().Get(namespaceRequest, metav1.GetOptions{})
	if err != nil {
		e := fmt.Errorf("failed to determine pod's namespace: %v", err)
		k.logger.Errorf("%v", e)
		return k8sutil.ErrToAdmissionResponse(e)
	}

	automountServiceAccountToken, doCheck := namespace.ObjectMeta.Annotations["karydia.gardener.cloud/automountServiceAccountToken"]
	if doCheck {
		patches, validationErrors = secureAutomountServiceAccountToken(pod, automountServiceAccountToken, patches, validationErrors)
	}

	seccompProfile, doCheck := namespace.ObjectMeta.Annotations["karydia.gardener.cloud/seccompProfile"]
	if doCheck {
		seccompPod, ok := pod.ObjectMeta.Annotations["seccomp.security.alpha.kubernetes.io/pod"]
		if !ok && mutationAllowed {
			if len(pod.ObjectMeta.Annotations) == 0 {
				// If no annotation object exists yet, we have
				// to create it. Otherwise we will encounter
				// the following error:
				// 'jsonpatch add operation does not apply: doc is missing path: "/metadata/annotations..."'
				patches = append(patches, fmt.Sprintf(`{"op": "add", "path": "/metadata/annotations", "value": {"%s": "%s"}}`, "seccomp.security.alpha.kubernetes.io/pod", seccompProfile))
			} else {
				patches = append(patches, fmt.Sprintf(`{"op": "add", "path": "/metadata/annotations/%s", "value": "%s"}`, "seccomp.security.alpha.kubernetes.io~1pod", seccompProfile))
			}
		} else if seccompPod != seccompProfile {
			validationErrorMsg := fmt.Sprintf("seccomp profile ('seccomp.security.alpha.kubernetes.io/pod') must be '%s'", seccompProfile)
			validationErrors = append(validationErrors, validationErrorMsg)
		}
	}

	if len(validationErrors) > 0 {
		return &v1beta1.AdmissionResponse{
			Allowed: false,
			Result: &metav1.Status{
				Message: fmt.Sprintf("%+v", validationErrors),
			},
		}
	}

	response := &v1beta1.AdmissionResponse{
		Allowed: true,
	}

	if len(patches) > 0 {
		patchesStr := strings.Join(patches, ",")
		patchType := v1beta1.PatchTypeJSONPatch

		response.Patch = []byte(fmt.Sprintf("[%s]", patchesStr))
		response.PatchType = &patchType
	}

	return response
}

func secureAutomountServiceAccountToken(pod corev1.Pod, annotation string, patches []string, errors []string) ([]string, []string) {
	if annotation == "forbidden" {
		if automountServiceAccountTokenUndefined(&pod) {
			errors = append(errors, "automount of service account not allowed")
		}
	} else if annotation == "non-default" {
		if automountServiceAccountTokenUndefined(&pod) && pod.Spec.ServiceAccountName == "default" {
			errors = append(errors, "automount of service account 'default' not allowed")
		}
	} else if annotation == "remove-default" {
		if automountServiceAccountTokenUndefined(&pod) && pod.Spec.ServiceAccountName == "default" {
			patches = append(patches, fmt.Sprintf(`{"op": "add", "path": "/spec/automountServiceAccountToken", "value": %s}`, "false"))
			patches = append(patches, fmt.Sprintf(`{"op": "remove", "path": "/spec/serviceAccountName"}`))
			for i, v := range pod.Spec.Volumes {
				if strings.HasPrefix(v.Name, "default-token-") {
					patches = append(patches, fmt.Sprintf(`{"op": "remove", "path": "/spec/volumes/%d"}`, i))
				}
			}
			for i, c := range pod.Spec.Containers {
				for j, v := range c.VolumeMounts {
					if strings.HasPrefix(v.Name, "default-token-") {
						patches = append(patches, fmt.Sprintf(`{"op": "remove", "path": "/spec/containers/%d/volumeMounts/%d"}`, i, j))
					}
				}
			}
		}
	}
	fmt.Printf("Got %d patches", len(patches))
	return patches, errors
}

func doesAutomountServiceAccountToken(pod *corev1.Pod) bool {
	return pod.Spec.AutomountServiceAccountToken == nil || *pod.Spec.AutomountServiceAccountToken
}

func automountServiceAccountTokenUndefined(pod *corev1.Pod) bool {
	return pod.Spec.AutomountServiceAccountToken == nil
}

func shouldIgnore(ar v1beta1.AdmissionReview) (bool, error) {
	// Right now we only care about 'CREATE' events. Needs to be
	// updated depending on the kind of admission requests that
	// `KarydiaAdmission` should handle in this package.
	// https://github.com/kubernetes/api/blob/kubernetes-1.12.2/admission/v1beta1/types.go#L118-L127
	if ar.Request.Operation != v1beta1.Create {
		return true, nil
	}
	return false, nil
}
