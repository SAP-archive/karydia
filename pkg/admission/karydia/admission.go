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
	"fmt"

	"github.com/karydia/karydia/pkg/apis/karydia/v1alpha1"

	"github.com/karydia/karydia/pkg/k8sutil"
	"github.com/sirupsen/logrus"
	"k8s.io/api/admission/v1beta1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var kindPod = metav1.GroupVersionKind{Group: "", Version: "v1", Kind: "Pod"}
var kindServiceAccount = metav1.GroupVersionKind{Group: "", Version: "v1", Kind: "ServiceAccount"}

type KarydiaAdmission struct {
	logger        *logrus.Logger
	kubeClientset kubernetes.Interface
	karydiaConfig *v1alpha1.KarydiaConfig
}

func (k *KarydiaAdmission) UpdateConfig(karydiaConfig v1alpha1.KarydiaConfig) error {
	k.karydiaConfig = &karydiaConfig
	return nil
}

type Config struct {
	KubeClientset kubernetes.Interface
	KarydiaConfig *v1alpha1.KarydiaConfig
}

type Setting struct {
	value string
	src   string
}

type patchOperation struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value,omitempty"`
}

type Patches struct {
	operations []patchOperation
	annotated  bool
}

func New(config *Config) (*KarydiaAdmission, error) {
	logger := logrus.New()
	logger.Level = logrus.InfoLevel

	return &KarydiaAdmission{
		logger:        logger,
		kubeClientset: config.KubeClientset,
		karydiaConfig: config.KarydiaConfig,
	}, nil
}

func (k *KarydiaAdmission) Admit(ar v1beta1.AdmissionReview, mutationAllowed bool) *v1beta1.AdmissionResponse {
	req := ar.Request
	if shouldIgnoreEvent(ar) {
		return k8sutil.AllowAdmissionResponse()
	}

	switch req.Kind {
	case kindPod:
		pod, err := decodePod(req.Object.Raw)
		if err != nil {
			k.logger.Errorf("failed to decode object: %v", err)
			return k8sutil.ErrToAdmissionResponse(err)
		}

		namespace, err := k.getNamespaceFromAdmissionRequest(*req)
		if err != nil {
			k.logger.Errorf("%v", err)
			return k8sutil.ErrToAdmissionResponse(err)
		}

		if mutationAllowed {
			return k.mutatePod(pod, namespace)
		}
		return k.validatePod(pod, namespace)
	case kindServiceAccount:
		sAcc, err := decodeServiceAccount(req.Object.Raw)
		if err != nil {
			k.logger.Errorf("failed to decode object: %v", err)
			return k8sutil.ErrToAdmissionResponse(err)
		}

		namespace, err := k.getNamespaceFromAdmissionRequest(*req)
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

func (patches *Patches) toBytes() []byte {
	patchBytes, err := json.Marshal(patches.operations)
	if err != nil {
		return nil
	}
	return patchBytes
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
