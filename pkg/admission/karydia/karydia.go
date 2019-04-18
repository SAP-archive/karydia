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
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/karydia/karydia/pkg/k8sutil"
	"github.com/karydia/karydia/pkg/k8sutil/scheme"
)

var resourcePod = metav1.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}
var kindPod = metav1.GroupVersionKind{Group: "", Version: "v1", Kind: "Pod"}

type KarydiaAdmission struct {
	logger        *logrus.Logger
	kubeClientset *kubernetes.Clientset
}

type Config struct {
	KubeClientset *kubernetes.Clientset
}

func New(config *Config) (*KarydiaAdmission, error) {
	logger := logrus.New()
	logger.Level = logrus.InfoLevel

	return &KarydiaAdmission{
		logger:        logger,
		kubeClientset: config.KubeClientset,
	}, nil
}

func (k *KarydiaAdmission) Admit(ar v1beta1.AdmissionReview, mutationAllowed bool) *v1beta1.AdmissionResponse {
	if shouldIgnoreEvent(ar) {
		return k8sutil.AllowAdmissionResponse()
	}

	if ar.Request.Kind == kindPod && ar.Request.Resource == resourcePod {
		return k.AdmitPod(*ar.Request, mutationAllowed)
	}

	return k8sutil.AllowAdmissionResponse()
}

func (k *KarydiaAdmission) AdmitPod(admissionRequest v1beta1.AdmissionRequest, mutationAllowed bool) *v1beta1.AdmissionResponse {
	var patches, validationErrors []string

	raw := admissionRequest.Object.Raw

	pod, err := decodePod(raw)
	if err != nil {
		k.logger.Errorf("failed to decode object: %v", err)
		return k8sutil.ErrToAdmissionResponse(err)
	}

	namespace, err := k.getNamespaceFromAdmissionRequest(admissionRequest)
	if err != nil {
		k.logger.Errorf("%v", err)
		return k8sutil.ErrToAdmissionResponse(err)
	}

	automountServiceAccountToken, annotated := namespace.ObjectMeta.Annotations["karydia.gardener.cloud/automountServiceAccountToken"]
	if annotated {
		patches = mutatePodServiceAccountToken(*pod, automountServiceAccountToken, patches)
		validationErrors = validatePodServiceAccountToken(*pod, automountServiceAccountToken, validationErrors)
	}

	seccompProfile, annotated := namespace.ObjectMeta.Annotations["karydia.gardener.cloud/seccompProfile"]
	if annotated {
		patches, validationErrors = admitSeccompProfile(*pod, seccompProfile, mutationAllowed, patches, validationErrors)
	}

	return admitResponse(patches, validationErrors)
}

func validatePodServiceAccountToken(pod corev1.Pod, nsAnnotation string, validationErrors []string) []string {
	if nsAnnotation == "forbidden" {
		if automountServiceAccountTokenUndefined(&pod) {
			validationErrors = append(validationErrors, "automount of service account not allowed")
		}
	} else if nsAnnotation == "non-default" || nsAnnotation == "remove-default" {
		if automountServiceAccountTokenUndefined(&pod) && pod.Spec.ServiceAccountName == "default" {
			validationErrors = append(validationErrors, "automount of service account 'default' not allowed")
		}
	}
	return validationErrors
}

func mutatePodServiceAccountToken(pod corev1.Pod, nsAnnotation string, patches []string) []string {
	if nsAnnotation == "remove-default" {
		if automountServiceAccountTokenUndefined(&pod) && pod.Spec.ServiceAccountName == "default" {
			patches = append(patches, fmt.Sprintf(`{"op": "add", "path": "/spec/automountServiceAccountToken", "value": %s}`, "false"))
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
	return patches
}

func admitSeccompProfile(pod corev1.Pod, seccompProfile string, mutationAllowed bool, patches []string, validationErrors []string) ([]string, []string) {
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
	return patches, validationErrors
}

func automountServiceAccountTokenUndefined(pod *corev1.Pod) bool {
	return pod.Spec.AutomountServiceAccountToken == nil
}

func (k *KarydiaAdmission) getNamespaceFromAdmissionRequest(ar v1beta1.AdmissionRequest) (*v1.Namespace, error) {
	namespaceRequest := ar.Namespace
	if namespaceRequest == "" {
		e := fmt.Errorf("received request with empty namespace")
		return nil, e
	}
	namespace, err := k.kubeClientset.CoreV1().Namespaces().Get(namespaceRequest, metav1.GetOptions{})
	if err != nil {
		e := fmt.Errorf("failed to determine pod's namespace: %v", err)
		return nil, e
	}
	return namespace, nil
}

func admitResponse(patches []string, validationErrors []string) *v1beta1.AdmissionResponse {
	if len(validationErrors) > 0 {
		return k8sutil.ValidationErrorAdmissionResponse(validationErrors)
	}
	return k8sutil.MutatingAdmissionResponse(patches)
}

func decodePod(raw []byte) (*corev1.Pod, error) {
	pod := &corev1.Pod{}
	deserializer := scheme.Codecs.UniversalDeserializer()
	if _, _, err := deserializer.Decode(raw, nil, pod); err != nil {
		return nil, err
	}
	return pod, nil
}

func shouldIgnoreEvent(ar v1beta1.AdmissionReview) bool {
	// Right now we only care about 'CREATE' events. Needs to be
	// updated depending on the kind of admission requests that
	// `KarydiaAdmission` should handle in this package.
	// https://github.com/kubernetes/api/blob/kubernetes-1.12.2/admission/v1beta1/types.go#L118-L127
	if ar.Request.Operation != v1beta1.Create {
		return true
	}
	return false
}
