package e2e

import (
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAutomountServiceAccountToken(t *testing.T) {
	namespace, err := f.CreateTestNamespace()
	if err != nil {
		t.Fatalf("failed to create test namespace: %v", err)
	}

	namespace.ObjectMeta.Annotations = map[string]string{
		"karydia.gardener.cloud/automountServiceAccountToken": "forbidden",
	}

	namespace, err = f.KubeClientset.CoreV1().Namespaces().Update(namespace)
	if err != nil {
		t.Fatalf("failed to annotate test namespace: %v", err)
	}

	ns := namespace.ObjectMeta.Name

	automountServiceAccountToken := true

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "karydia-e2e-test-pod",
			Namespace: ns,
		},
		Spec: corev1.PodSpec{
			AutomountServiceAccountToken: &automountServiceAccountToken,
			Containers: []corev1.Container{
				{
					Name:  "nginx",
					Image: "nginx",
				},
			},
		},
	}

	_, err = f.KubeClientset.CoreV1().Pods(ns).Create(pod)
	if err == nil {
		t.Fatalf("expected pod creation to fail")
	}

	pod.Spec.AutomountServiceAccountToken = nil

	_, err = f.KubeClientset.CoreV1().Pods(ns).Create(pod)
	if err == nil {
		t.Fatalf("expected pod creation to fail")
	}

	automountServiceAccountToken = false
	pod.Spec.AutomountServiceAccountToken = &automountServiceAccountToken

	pod, err = f.KubeClientset.CoreV1().Pods(ns).Create(pod)
	if err != nil {
		t.Fatalf("failed to create pod: %v", err)
	}

	timeout := 2 * time.Minute
	if err := f.WaitPodRunning(pod.ObjectMeta.Namespace, pod.ObjectMeta.Name, timeout); err != nil {
		t.Fatalf("pod never reached state running")
	}
}
