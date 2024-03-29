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
	"k8s.io/api/admission/v1beta1"
	coreV1 "k8s.io/api/core/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sfake "k8s.io/client-go/kubernetes/fake"
)

/* Admission with ´mutating and validating webhook */
func TestTerminatingPod(t *testing.T) {
	var kubeobjects []runtime.Object

	namespace := &coreV1.Namespace{}
	namespace.Name = "normal"
	kubeobjects = append(kubeobjects, namespace)

	kubeclient := k8sfake.NewSimpleClientset(kubeobjects...)

	karydiaAdmission, err := New(&Config{
		KubeClientset: kubeclient,
	})
	if err != nil {
		t.Fatal("Failed to load karydia admission:", err)
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "karydia-e2e-test-pod",
			Namespace: "normal",
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "nginx",
					Image: "nginx",
				},
			},
		},
		Status: corev1.PodStatus{
			Phase: "Terminating",
		},
	}
	rawPod, _ := json.Marshal(pod)

	oldPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "karydia-e2e-test-pod",
			Namespace: "normal",
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "nginx",
					Image: "nginx",
				},
			},
		},
		Status: corev1.PodStatus{
			Phase: "Running",
		},
	}
	rawOldPod, _ := json.Marshal(oldPod)

	ar := v1beta1.AdmissionReview{
		Request: &v1beta1.AdmissionRequest{
			Operation: "UPDATE",
			Namespace: "normal",
			Kind:      metav1.GroupVersionKind{Group: "", Version: "v1", Kind: "Pod"},
			Object: runtime.RawExtension{
				Raw: rawPod,
			},
			OldObject: runtime.RawExtension{
				Raw: rawOldPod,
			},
		},
	}

	mutationResponse := karydiaAdmission.Admit(ar, true)
	if !mutationResponse.Allowed {
		t.Error("expected mutation response to be true but is", mutationResponse.Allowed)
	}

	var patches []patchOperation
	err = json.Unmarshal(mutationResponse.Patch, &patches)
	if len(patches) != 0 {
		t.Error("expected number of patches to be 0 but is", len(patches))
	}

	t.Log(patches)
}

func TestPodPlainSeccomp(t *testing.T) {
	var kubeobjects []runtime.Object

	namespace := &coreV1.Namespace{}
	namespace.Name = "special"
	namespace.Annotations = map[string]string{
		"karydia.gardener.cloud/seccompProfile":     "runtime/default",
		"karydia.gardener.cloud/podSecurityContext": "nobody",
	}
	kubeobjects = append(kubeobjects, namespace)

	kubeclient := k8sfake.NewSimpleClientset(kubeobjects...)

	karydiaAdmission, err := New(&Config{
		KubeClientset: kubeclient,
	})
	if err != nil {
		t.Fatal("Failed to load karydia admission:", err)
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "karydia-e2e-test-pod",
			Namespace: "special",
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "nginx",
					Image: "nginx",
				},
			},
		},
	}
	rawPod, _ := json.Marshal(pod)

	ar := v1beta1.AdmissionReview{
		Request: &v1beta1.AdmissionRequest{
			Operation: "CREATE",
			Namespace: "special",
			Kind:      metav1.GroupVersionKind{Group: "", Version: "v1", Kind: "Pod"},
			Object: runtime.RawExtension{
				Raw: rawPod,
			},
		},
	}

	mutationResponse := karydiaAdmission.Admit(ar, true)
	if !mutationResponse.Allowed {
		t.Error("expected mutation response to be true but is", mutationResponse.Allowed)
	}

	var patches []patchOperation
	err = json.Unmarshal(mutationResponse.Patch, &patches)
	if len(patches) != 5 {
		t.Error("expected number of patches to be 5 but is", len(patches))
	}

	t.Log(patches)

	mutatedPod, err := patchPodRaw(*pod, mutationResponse.Patch)
	if err != nil {
		t.Error("failed to apply patches:", err)
	}

	validationResponse := karydiaAdmission.Admit(ar, false)
	if validationResponse.Allowed {
		t.Error("expected validation response to be false but is", validationResponse.Allowed)
	}

	ar.Request.Object.Raw, _ = json.Marshal(mutatedPod)

	validationResponse = karydiaAdmission.Admit(ar, false)
	if !validationResponse.Allowed {
		t.Error("expected validation response to be true but is", validationResponse.Allowed)
	}

}

func TestPodPlainSecContext(t *testing.T) {
	var kubeobjects []runtime.Object

	namespace := &coreV1.Namespace{}
	namespace.Name = "special"
	namespace.Annotations = map[string]string{
		"karydia.gardener.cloud/podSecurityContext": "nobody",
	}
	kubeobjects = append(kubeobjects, namespace)

	kubeclient := k8sfake.NewSimpleClientset(kubeobjects...)

	karydiaAdmission, err := New(&Config{
		KubeClientset: kubeclient,
	})
	if err != nil {
		t.Fatal("Failed to load karydia admission:", err)
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "karydia-e2e-test-pod",
			Namespace: "special",
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "nginx",
					Image: "nginx",
				},
			},
		},
	}
	rawPod, _ := json.Marshal(pod)

	ar := v1beta1.AdmissionReview{
		Request: &v1beta1.AdmissionRequest{
			Operation: "CREATE",
			Namespace: "special",
			Kind:      metav1.GroupVersionKind{Group: "", Version: "v1", Kind: "Pod"},
			Object: runtime.RawExtension{
				Raw: rawPod,
			},
		},
	}

	mutationResponse := karydiaAdmission.Admit(ar, true)
	if !mutationResponse.Allowed {
		t.Error("expected mutation response to be true but is", mutationResponse.Allowed)
	}

	var patches []patchOperation
	err = json.Unmarshal(mutationResponse.Patch, &patches)
	if len(patches) != 3 {
		t.Error("expected number of patches to be 3 but is", len(patches))
	}

	t.Log(patches)

	mutatedPod, err := patchPodRaw(*pod, mutationResponse.Patch)
	if err != nil {
		t.Error("failed to apply patches:", err)
	}

	validationResponse := karydiaAdmission.Admit(ar, false)
	if validationResponse.Allowed {
		t.Error("expected validation response to be false but is", validationResponse.Allowed)
	}

	ar.Request.Object.Raw, _ = json.Marshal(mutatedPod)

	validationResponse = karydiaAdmission.Admit(ar, false)
	if !validationResponse.Allowed {
		t.Error("expected validation response to be true but is", validationResponse.Allowed)
	}

}

func TestPodDefinedSecContext(t *testing.T) {
	var kubeobjects []runtime.Object

	namespace := &coreV1.Namespace{}
	namespace.Name = "special"
	namespace.Annotations = map[string]string{
		"karydia.gardener.cloud/podSecurityContext": "nobody",
	}
	kubeobjects = append(kubeobjects, namespace)

	kubeclient := k8sfake.NewSimpleClientset(kubeobjects...)

	karydiaAdmission, err := New(&Config{
		KubeClientset: kubeclient,
	})
	if err != nil {
		t.Fatal("Failed to load karydia admission:", err)
	}

	var uid int64 = 1000
	var allowPrivilegeEscalation = true

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "karydia-e2e-test-pod",
			Namespace: "special",
		},
		Spec: corev1.PodSpec{
			SecurityContext: &corev1.PodSecurityContext{
				RunAsUser: &uid,
			},
			Containers: []corev1.Container{
				{
					Name:  "nginx",
					Image: "nginx",
					SecurityContext: &corev1.SecurityContext{
						AllowPrivilegeEscalation: &allowPrivilegeEscalation,
					},
				},
			},
		},
	}
	rawPod, _ := json.Marshal(pod)

	ar := v1beta1.AdmissionReview{
		Request: &v1beta1.AdmissionRequest{
			Operation: "CREATE",
			Namespace: "special",
			Kind:      metav1.GroupVersionKind{Group: "", Version: "v1", Kind: "Pod"},
			Object: runtime.RawExtension{
				Raw: rawPod,
			},
		},
	}

	mutationResponse := karydiaAdmission.Admit(ar, true)
	if !mutationResponse.Allowed {
		t.Error("expected mutation response to be true but is", mutationResponse.Allowed)
	}

	var patches []patchOperation
	err = json.Unmarshal(mutationResponse.Patch, &patches)
	if len(patches) != 0 {
		t.Error("expected number of patches to be 0 but is", len(patches))
	}

	t.Log(patches)

	mutatedPod, err := patchPodRaw(*pod, mutationResponse.Patch)
	if err != nil {
		t.Error("failed to apply patches:", err)
	}

	validationResponse := karydiaAdmission.Admit(ar, false)
	if !validationResponse.Allowed {
		t.Error("expected validation response to be true but is", validationResponse.Allowed)
	}

	ar.Request.Object.Raw, _ = json.Marshal(mutatedPod)

	validationResponse = karydiaAdmission.Admit(ar, false)
	if !validationResponse.Allowed {
		t.Error("expected validation response to be true but is", validationResponse.Allowed)
	}

}

func TestPodDefinedSecContext2(t *testing.T) {
	var kubeobjects []runtime.Object

	namespace := &coreV1.Namespace{}
	namespace.Name = "special"
	namespace.Annotations = map[string]string{
		"karydia.gardener.cloud/podSecurityContext": "nobody",
	}
	kubeobjects = append(kubeobjects, namespace)

	kubeclient := k8sfake.NewSimpleClientset(kubeobjects...)

	karydiaAdmission, err := New(&Config{
		KubeClientset: kubeclient,
	})
	if err != nil {
		t.Fatal("Failed to load karydia admission:", err)
	}

	var uid int64 = 1000
	var fsgid int64 = 2000

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "karydia-e2e-test-pod",
			Namespace: "special",
		},
		Spec: corev1.PodSpec{
			SecurityContext: &corev1.PodSecurityContext{
				FSGroup: &fsgid,
			},
			Containers: []corev1.Container{
				{
					Name:  "nginx",
					Image: "nginx",
					SecurityContext: &corev1.SecurityContext{
						RunAsUser: &uid,
					},
				},
			},
		},
	}
	rawPod, _ := json.Marshal(pod)

	ar := v1beta1.AdmissionReview{
		Request: &v1beta1.AdmissionRequest{
			Operation: "CREATE",
			Namespace: "special",
			Kind:      metav1.GroupVersionKind{Group: "", Version: "v1", Kind: "Pod"},
			Object: runtime.RawExtension{
				Raw: rawPod,
			},
		},
	}

	mutationResponse := karydiaAdmission.Admit(ar, true)
	if !mutationResponse.Allowed {
		t.Error("expected mutation response to be true but is", mutationResponse.Allowed)
	}

	var patches []patchOperation
	err = json.Unmarshal(mutationResponse.Patch, &patches)
	if len(patches) != 4 {
		t.Error("expected number of patches to be 4 but is", len(patches))
	}

	t.Log(patches)

	mutatedPod, err := patchPodRaw(*pod, mutationResponse.Patch)
	if err != nil {
		t.Error("failed to apply patches:", err)
	}

	validationResponse := karydiaAdmission.Admit(ar, false)
	if validationResponse.Allowed {
		t.Error("expected validation response to be false but is", validationResponse.Allowed)
	}

	ar.Request.Object.Raw, _ = json.Marshal(mutatedPod)

	validationResponse = karydiaAdmission.Admit(ar, false)
	if !validationResponse.Allowed {
		t.Error("expected validation response to be true but is", validationResponse.Allowed)
	}

	if mutatedPod.Spec.SecurityContext == nil {
		t.Error("expected security context to be defined but is nil")
	} else {
		if mutatedPod.Spec.SecurityContext.RunAsUser == nil {
			t.Error("expected security context user id to be defined but is nil")
		} else {
			if *mutatedPod.Spec.SecurityContext.RunAsUser != 65534 {
				t.Errorf("expected security context user id to be %v but is %v", 65534, *mutatedPod.Spec.SecurityContext.RunAsUser)
			}
		}
		if mutatedPod.Spec.SecurityContext.RunAsGroup == nil {
			t.Error("expected security context group id to be defined but is nil")
		} else {
			if *mutatedPod.Spec.SecurityContext.RunAsGroup != 65534 {
				t.Errorf("expected security context group id to be %v but is %v", 65534, *mutatedPod.Spec.SecurityContext.RunAsGroup)
			}
		}
		if mutatedPod.Spec.SecurityContext.FSGroup == nil {
			t.Error("expected security context fs group id to be defined but is nil")
		} else {
			if *mutatedPod.Spec.SecurityContext.FSGroup != fsgid {
				t.Errorf("expected security context fs group id to be %v but is %v", fsgid, *mutatedPod.Spec.SecurityContext.FSGroup)
			}
		}
	}

	mutatedContainer := mutatedPod.Spec.Containers[0]

	if mutatedContainer.SecurityContext == nil {
		t.Error("expected container security context to be defined but is nil")
	} else {
		if mutatedContainer.SecurityContext.RunAsUser == nil {
			t.Error("expected container security context user id to be defined but is nil")
		} else {
			if *mutatedContainer.SecurityContext.RunAsUser != uid {
				t.Errorf("expected container security context user id to be %v but is %v", uid, *mutatedContainer.SecurityContext.RunAsUser)
			}
		}
		if mutatedContainer.SecurityContext.AllowPrivilegeEscalation == nil {
			t.Error("expected container security context allow privilege escalation to be defined but is nil")
		} else {
			if *mutatedContainer.SecurityContext.AllowPrivilegeEscalation {
				t.Errorf("expected container security context allow privilege escalation to be %v but is %v", false, *mutatedContainer.SecurityContext.AllowPrivilegeEscalation)
			}
		}
	}
}

func TestPodCorrectSeccomp(t *testing.T) {
	var kubeobjects []runtime.Object

	namespace := &coreV1.Namespace{}
	namespace.Name = "special"
	namespace.Annotations = map[string]string{
		"karydia.gardener.cloud/seccompProfile": "docker/default",
	}
	kubeobjects = append(kubeobjects, namespace)

	kubeclient := k8sfake.NewSimpleClientset(kubeobjects...)

	karydiaAdmission, err := New(&Config{
		KubeClientset: kubeclient,
	})
	if err != nil {
		t.Fatal("Failed to load karydia admission:", err)
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "karydia-e2e-test-pod",
			Namespace: "special",
			Annotations: map[string]string{
				"seccomp.security.alpha.kubernetes.io/pod": "runtime/default",
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "nginx",
					Image: "nginx",
				},
			},
		},
	}
	rawPod, _ := json.Marshal(pod)

	ar := v1beta1.AdmissionReview{
		Request: &v1beta1.AdmissionRequest{
			Operation: "CREATE",
			Namespace: "special",
			Kind:      metav1.GroupVersionKind{Group: "", Version: "v1", Kind: "Pod"},
			Object: runtime.RawExtension{
				Raw: rawPod,
			},
		},
	}

	mutationResponse := karydiaAdmission.Admit(ar, true)
	if !mutationResponse.Allowed {
		t.Error("expected mutation response to be true but is", mutationResponse.Allowed)
	}

	var patches []patchOperation
	err = json.Unmarshal(mutationResponse.Patch, &patches)
	if len(patches) != 0 {
		t.Error("expected number of patches to be 0 but is", len(patches))
	}

	validationResponse := karydiaAdmission.Admit(ar, false)
	if !validationResponse.Allowed {
		t.Error("expected validation response to be true but is", mutationResponse.Allowed)
	}
}

func TestServiceAccountPlain(t *testing.T) {
	var kubeobjects []runtime.Object

	namespace := &coreV1.Namespace{}
	namespace.Name = "special"
	namespace.Annotations = map[string]string{
		"karydia.gardener.cloud/automountServiceAccountToken": "change-all",
	}
	kubeobjects = append(kubeobjects, namespace)

	kubeclient := k8sfake.NewSimpleClientset(kubeobjects...)

	karydiaAdmission, err := New(&Config{
		KubeClientset: kubeclient,
	})
	if err != nil {
		t.Fatal("Failed to load karydia admission:", err)
	}

	servieAccount := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "special-service-account",
			Namespace: "special",
		},
	}
	servieAccount.Name = "special-service-account"

	rawServieAccount, _ := json.Marshal(servieAccount)

	ar := v1beta1.AdmissionReview{
		Request: &v1beta1.AdmissionRequest{
			Operation: "CREATE",
			Namespace: "special",
			Kind:      metav1.GroupVersionKind{Group: "", Version: "v1", Kind: "ServiceAccount"},
			Object: runtime.RawExtension{
				Raw: rawServieAccount,
			},
		},
	}

	mutationResponse := karydiaAdmission.Admit(ar, true)
	if !mutationResponse.Allowed {
		t.Error("expected mutation response to be true but is", mutationResponse.Allowed)
	}

	var patches []patchOperation
	err = json.Unmarshal(mutationResponse.Patch, &patches)
	if len(patches) != 2 {
		t.Error("expected number of patches to be 2 but is", len(patches))
	}

	mutatedServiceAccount, err := patchServiceAccountRaw(*servieAccount, mutationResponse.Patch)
	if err != nil {
		t.Error("failed to apply patches:", err)
	}

	validationResponse := karydiaAdmission.Admit(ar, false)
	if validationResponse.Allowed {
		t.Error("expected validation response to be false but is", mutationResponse.Allowed)
	}

	ar.Request.Object.Raw, _ = json.Marshal(mutatedServiceAccount)

	validationResponse = karydiaAdmission.Admit(ar, false)
	if !validationResponse.Allowed {
		t.Error("expected validation response to be true but is", mutationResponse.Allowed)
	}

}

func TestServiceAccountAutomountDefined(t *testing.T) {
	var kubeobjects []runtime.Object

	namespace := &coreV1.Namespace{}
	namespace.Name = "special"
	namespace.Annotations = map[string]string{
		"karydia.gardener.cloud/automountServiceAccountToken": "change-all",
	}
	kubeobjects = append(kubeobjects, namespace)

	kubeclient := k8sfake.NewSimpleClientset(kubeobjects...)

	karydiaAdmission, err := New(&Config{
		KubeClientset: kubeclient,
	})
	if err != nil {
		t.Fatal("Failed to load karydia admission:", err)
	}

	var vTrue = true

	servieAccount := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "special-service-account",
			Namespace: "special",
		},
		AutomountServiceAccountToken: &vTrue,
	}
	servieAccount.Name = "special-service-account"

	rawServieAccount, _ := json.Marshal(servieAccount)

	ar := v1beta1.AdmissionReview{
		Request: &v1beta1.AdmissionRequest{
			Operation: "CREATE",
			Namespace: "special",
			Kind:      metav1.GroupVersionKind{Group: "", Version: "v1", Kind: "ServiceAccount"},
			Object: runtime.RawExtension{
				Raw: rawServieAccount,
			},
		},
	}

	mutationResponse := karydiaAdmission.Admit(ar, true)
	if !mutationResponse.Allowed {
		t.Error("expected mutation response to be true but is", mutationResponse.Allowed)
	}

	var patches []patchOperation
	err = json.Unmarshal(mutationResponse.Patch, &patches)
	if len(patches) != 0 {
		t.Error("expected number of patches to be 0 but is", len(patches))
	}

	mutatedServiceAccount, err := patchServiceAccountRaw(*servieAccount, mutationResponse.Patch)
	if err != nil {
		t.Error("failed to apply patches:", err)
	}

	validationResponse := karydiaAdmission.Admit(ar, false)
	if !validationResponse.Allowed {
		t.Error("expected validation response to be true but is", mutationResponse.Allowed)
	}

	ar.Request.Object.Raw, _ = json.Marshal(mutatedServiceAccount)

	validationResponse = karydiaAdmission.Admit(ar, false)
	if !validationResponse.Allowed {
		t.Error("expected validation response to be true but is", mutationResponse.Allowed)
	}

}

func TestServiceAccountDefault(t *testing.T) {
	var kubeobjects []runtime.Object

	namespace := &coreV1.Namespace{}
	namespace.Name = "special"
	namespace.Annotations = map[string]string{
		"karydia.gardener.cloud/automountServiceAccountToken": "change-default",
	}
	kubeobjects = append(kubeobjects, namespace)

	kubeclient := k8sfake.NewSimpleClientset(kubeobjects...)

	karydiaAdmission, err := New(&Config{
		KubeClientset: kubeclient,
	})
	if err != nil {
		t.Fatal("Failed to load karydia admission:", err)
	}

	servieAccount := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "default",
			Namespace: "special",
		},
	}
	servieAccount.Name = "default"

	rawServieAccount, _ := json.Marshal(servieAccount)

	ar := v1beta1.AdmissionReview{
		Request: &v1beta1.AdmissionRequest{
			Operation: "CREATE",
			Namespace: "special",
			Kind:      metav1.GroupVersionKind{Group: "", Version: "v1", Kind: "ServiceAccount"},
			Object: runtime.RawExtension{
				Raw: rawServieAccount,
			},
		},
	}

	mutationResponse := karydiaAdmission.Admit(ar, true)
	if !mutationResponse.Allowed {
		t.Error("expected mutation response to be true but is", mutationResponse.Allowed)
	}

	var patches []patchOperation
	err = json.Unmarshal(mutationResponse.Patch, &patches)
	if len(patches) != 2 {
		t.Error("expected number of patches to be 2 but is", len(patches))
	}

	mutatedServiceAccount, err := patchServiceAccountRaw(*servieAccount, mutationResponse.Patch)
	if err != nil {
		t.Error("failed to apply patches:", err)
	}

	validationResponse := karydiaAdmission.Admit(ar, false)
	if validationResponse.Allowed {
		t.Error("expected validation response to be false but is", mutationResponse.Allowed)
	}

	ar.Request.Object.Raw, _ = json.Marshal(mutatedServiceAccount)

	validationResponse = karydiaAdmission.Admit(ar, false)
	if !validationResponse.Allowed {
		t.Error("expected validation response to be true but is", mutationResponse.Allowed)
	}

}

func TestInvalidAdmissionReview(t *testing.T) {
	kubeclient := k8sfake.NewSimpleClientset()

	karydiaAdmission, err := New(&Config{
		KubeClientset: kubeclient,
	})
	if err != nil {
		t.Fatal("Failed to load karydia admission:", err)
	}

	/* DELETE operation -> is currently ignored */
	ar := v1beta1.AdmissionReview{
		Request: &v1beta1.AdmissionRequest{
			Operation: "DELETE",
			Namespace: "special",
			Kind:      metav1.GroupVersionKind{Group: "", Version: "v1", Kind: "ServiceAccount"},
		},
	}

	mutationResponse := karydiaAdmission.Admit(ar, true)
	if !mutationResponse.Allowed {
		t.Error("expected mutation response to be true but is", mutationResponse.Allowed)
	}

	validationResponse := karydiaAdmission.Admit(ar, false)
	if !validationResponse.Allowed {
		t.Error("expected validation response to be true but is", mutationResponse.Allowed)
	}

	/* Ignored resource type */
	ar = v1beta1.AdmissionReview{
		Request: &v1beta1.AdmissionRequest{
			Operation: "CREATE",
			Namespace: "special",
			Kind:      metav1.GroupVersionKind{Group: "", Version: "v1", Kind: "XPodX"},
		},
	}

	mutationResponse = karydiaAdmission.Admit(ar, true)
	if !mutationResponse.Allowed {
		t.Error("expected mutation response to be true but is", mutationResponse.Allowed)
	}

	validationResponse = karydiaAdmission.Admit(ar, false)
	if !validationResponse.Allowed {
		t.Error("expected validation response to be true but is", mutationResponse.Allowed)
	}

	/* Unknown namepsace ServiceAccount */
	ar = v1beta1.AdmissionReview{
		Request: &v1beta1.AdmissionRequest{
			Operation: "UPDATE",
			Namespace: "special",
			Kind:      metav1.GroupVersionKind{Group: "", Version: "v1", Kind: "ServiceAccount"},
		},
	}

	mutationResponse = karydiaAdmission.Admit(ar, true)
	if mutationResponse.Allowed {
		t.Error("expected mutation response to be false but is", mutationResponse.Allowed)
	}

	validationResponse = karydiaAdmission.Admit(ar, false)
	if validationResponse.Allowed {
		t.Error("expected validation response to be false but is", mutationResponse.Allowed)
	}

	/* Unknown namepsace Pod */
	ar = v1beta1.AdmissionReview{
		Request: &v1beta1.AdmissionRequest{
			Operation: "UPDATE",
			Namespace: "special",
			Kind:      metav1.GroupVersionKind{Group: "", Version: "v1", Kind: "Pod"},
		},
	}

	mutationResponse = karydiaAdmission.Admit(ar, true)
	if mutationResponse.Allowed {
		t.Error("expected mutation response to be false but is", mutationResponse.Allowed)
	}

	validationResponse = karydiaAdmission.Admit(ar, false)
	if validationResponse.Allowed {
		t.Error("expected validation response to be false but is", mutationResponse.Allowed)
	}

	/* Missing namepsace */
	ar = v1beta1.AdmissionReview{
		Request: &v1beta1.AdmissionRequest{
			Operation: "UPDATE",
			Kind:      metav1.GroupVersionKind{Group: "", Version: "v1", Kind: "ServiceAccount"},
		},
	}

	mutationResponse = karydiaAdmission.Admit(ar, true)
	if mutationResponse.Allowed {
		t.Error("expected mutation response to be false but is", mutationResponse.Allowed)
	}

	validationResponse = karydiaAdmission.Admit(ar, false)
	if validationResponse.Allowed {
		t.Error("expected validation response to be false but is", mutationResponse.Allowed)
	}
}

func TestInvalidDecodeOfResources(t *testing.T) {
	var kubeobjects []runtime.Object

	namespace := &coreV1.Namespace{}
	namespace.Name = "special"
	namespace.Annotations = map[string]string{
		"karydia.gardener.cloud/automountServiceAccountToken": "change-default",
	}
	kubeobjects = append(kubeobjects, namespace)

	kubeclient := k8sfake.NewSimpleClientset(kubeobjects...)

	karydiaAdmission, err := New(&Config{
		KubeClientset: kubeclient,
	})
	if err != nil {
		t.Fatal("Failed to load karydia admission:", err)
	}

	invalidRawServiceAccount := make([]byte, 4)

	ar := v1beta1.AdmissionReview{
		Request: &v1beta1.AdmissionRequest{
			Operation: "CREATE",
			Namespace: "special",
			Kind:      metav1.GroupVersionKind{Group: "", Version: "v1", Kind: "ServiceAccount"},
			Object: runtime.RawExtension{
				Raw: invalidRawServiceAccount,
			},
		},
	}

	mutationResponse := karydiaAdmission.Admit(ar, true)
	if mutationResponse.Allowed {
		t.Error("expected mutation response to be false but is", mutationResponse.Allowed)
	}

	ar = v1beta1.AdmissionReview{
		Request: &v1beta1.AdmissionRequest{
			Operation: "CREATE",
			Namespace: "special",
			Kind:      metav1.GroupVersionKind{Group: "", Version: "v1", Kind: "Pod"},
			Object: runtime.RawExtension{
				Raw: invalidRawServiceAccount,
			},
		},
	}

	mutationResponse = karydiaAdmission.Admit(ar, true)
	if mutationResponse.Allowed {
		t.Error("expected mutation response to be false but is", mutationResponse.Allowed)
	}
}

/* Helper functions to patch k8s resources */
func patchPodRaw(pod corev1.Pod, patches []byte) (corev1.Pod, error) {
	var podJSON []byte
	podJSON, err := json.Marshal(&pod)
	if err != nil {
		return pod, err
	}

	patchObj, err := jsonpatch.DecodePatch(patches)
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

func patchServiceAccountRaw(serviceAccount corev1.ServiceAccount, patches []byte) (corev1.ServiceAccount, error) {
	var serviceAccountJSON []byte
	serviceAccountJSON, err := json.Marshal(&serviceAccount)
	if err != nil {
		return serviceAccount, err
	}

	patchObj, err := jsonpatch.DecodePatch(patches)
	if err != nil {
		return serviceAccount, err
	}
	serviceAccountPatchedJSON, err := patchObj.Apply(serviceAccountJSON)
	if err != nil {
		return serviceAccount, err
	}

	var serviceAccountPatched corev1.ServiceAccount
	json.Unmarshal(serviceAccountPatchedJSON, &serviceAccountPatched)

	return serviceAccountPatched, nil
}
