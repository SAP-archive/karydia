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

var resourceServiceAccount = metav1.GroupVersionResource{Group: "", Version: "v1", Resource: "serviceaccounts"}
var kindServiceAccount = metav1.GroupVersionKind{Group: "", Version: "v1", Kind: "ServiceAccount"}

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
		raw := ar.Request.Object.Raw

		pod, err := decodePod(raw)
		if err != nil {
			k.logger.Errorf("failed to decode object: %v", err)
			return k8sutil.ErrToAdmissionResponse(err)
		}

		namespace, err := k.getNamespaceFromAdmissionRequest(*ar.Request)
		if err != nil {
			k.logger.Errorf("%v", err)
			return k8sutil.ErrToAdmissionResponse(err)
		}

		if mutationAllowed {
			return k.mutatePod(pod, namespace)
		}
		return k.validatePod(pod, namespace)
	} else if ar.Request.Kind == kindServiceAccount && ar.Request.Resource == resourceServiceAccount {
		raw := ar.Request.Object.Raw

		sAcc, err := decodeServiceAccount(raw)
		if err != nil {
			k.logger.Errorf("failed to decode object: %v", err)
			return k8sutil.ErrToAdmissionResponse(err)
		}

		namespace, err := k.getNamespaceFromAdmissionRequest(*ar.Request)
		if err != nil {
			k.logger.Errorf("%v", err)
			return k8sutil.ErrToAdmissionResponse(err)
		}

		if mutationAllowed {
			return k.mutateServiceAccount(sAcc, namespace)
		}
		return k.validateServiceAccount(sAcc, namespace)
	}

	return k8sutil.AllowAdmissionResponse()
}

func (k *KarydiaAdmission) mutatePod(pod *corev1.Pod, ns *corev1.Namespace) *v1beta1.AdmissionResponse {
	var patches []string

	seccompProfile, annotated := ns.ObjectMeta.Annotations["karydia.gardener.cloud/seccompProfile"]
	if annotated {
		patches = mutatePodSeccompProfile(*pod, seccompProfile, patches)
	}
	return k8sutil.MutatingAdmissionResponse(patches)
}

func (k *KarydiaAdmission) validatePod(pod *corev1.Pod, ns *corev1.Namespace) *v1beta1.AdmissionResponse {
	var validationErrors []string

	seccompProfile, annotated := ns.ObjectMeta.Annotations["karydia.gardener.cloud/seccompProfile"]
	if annotated {
		validationErrors = validatePodSeccompProfile(*pod, seccompProfile, validationErrors)
	}

	return k8sutil.ValidatingAdmissionResponse(validationErrors)
}

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

func validatePodSeccompProfile(pod corev1.Pod, nsAnnotation string, validationErrors []string) []string {
	seccompPod, ok := pod.ObjectMeta.Annotations["seccomp.security.alpha.kubernetes.io/pod"]
	if !ok || seccompPod != nsAnnotation {
		validationErrorMsg := fmt.Sprintf("seccomp profile ('seccomp.security.alpha.kubernetes.io/pod') must be '%s'", nsAnnotation)
		validationErrors = append(validationErrors, validationErrorMsg)
	}
	return validationErrors
}

func mutatePodSeccompProfile(pod corev1.Pod, nsAnnotation string, patches []string) []string {
	_, ok := pod.ObjectMeta.Annotations["seccomp.security.alpha.kubernetes.io/pod"]
	if !ok {
		if len(pod.ObjectMeta.Annotations) == 0 {
			// If no annotation object exists yet, we have
			// to create it. Otherwise we will encounter
			// the following error:
			// 'jsonpatch add operation does not apply: doc is missing path: "/metadata/annotations..."'
			patches = append(patches, fmt.Sprintf(`{"op": "add", "path": "/metadata/annotations", "value": {"%s": "%s"}}`, "seccomp.security.alpha.kubernetes.io/pod", nsAnnotation))
		} else {
			patches = append(patches, fmt.Sprintf(`{"op": "add", "path": "/metadata/annotations/%s", "value": "%s"}`, "seccomp.security.alpha.kubernetes.io~1pod", nsAnnotation))
		}
	}
	return patches
}

func automountServiceAccountTokenUndefined(sAcc *corev1.ServiceAccount) bool {
	return sAcc.AutomountServiceAccountToken == nil
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

/* Utility functions to decode raw resources into objects */
func decodePod(raw []byte) (*corev1.Pod, error) {
	pod := &corev1.Pod{}
	deserializer := scheme.Codecs.UniversalDeserializer()
	if _, _, err := deserializer.Decode(raw, nil, pod); err != nil {
		return nil, err
	}
	return pod, nil
}

func decodeServiceAccount(raw []byte) (*corev1.ServiceAccount, error) {
	sAcc := &corev1.ServiceAccount{}
	deserializer := scheme.Codecs.UniversalDeserializer()
	if _, _, err := deserializer.Decode(raw, nil, sAcc); err != nil {
		return nil, err
	}
	return sAcc, nil
}

func shouldIgnoreEvent(ar v1beta1.AdmissionReview) bool {
	/* Right now we only care about 'CREATE' and 'UPDATE' events.
	   Needs to be updated depending on the kind of admission requests that
	   `KarydiaAdmission` should handle in this package.
	   https://github.com/kubernetes/api/blob/kubernetes-1.12.2/admission/v1beta1/types.go#L118-L127 */
	const Create v1beta1.Operation = "CREATE"
	const Update v1beta1.Operation = "UPDATE"
	operation := ar.Request.Operation
	if operation != Create && operation != Update {
		return true
	}
	return false
}
