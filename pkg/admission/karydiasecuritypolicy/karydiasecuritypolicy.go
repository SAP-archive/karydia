package karydiasecuritypolicy

// Loosely modelled after the PSP admission plugin in Kubernetes:
// https://github.com/kubernetes/kubernetes/blob/v1.12.2/plugin/pkg/admission/security/podsecuritypolicy/admission.go

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"k8s.io/api/admission/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	informers "github.com/kinvolk/karydia/pkg/client/informers/externalversions"
	listers "github.com/kinvolk/karydia/pkg/client/listers/karydia/v1alpha1"
)

var (
	resourcePod = metav1.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}
)

type KarydiaSecurityPolicyAdmission struct {
	logger *logrus.Logger

	lister    listers.KarydiaSecurityPolicyLister
	readyFunc func() bool
}

func New() (*KarydiaSecurityPolicyAdmission, error) {
	logger := logrus.New()
	logger.Level = logrus.InfoLevel

	return &KarydiaSecurityPolicyAdmission{
		logger: logger,
	}, nil
}

func (k *KarydiaSecurityPolicyAdmission) WaitForReady() bool {
	if k.readyFunc == nil {
		return true
	}
	timeout := time.After(10 * time.Second)
	for !k.readyFunc() {
		select {
		case <-time.After(100 * time.Millisecond):
		case <-timeout:
			return k.readyFunc()
		}
	}
	return true
}

func (k *KarydiaSecurityPolicyAdmission) SetExternalInformerFactory(f informers.SharedInformerFactory) {
	informer := f.Karydia().V1alpha1().KarydiaSecurityPolicies()
	k.lister = informer.Lister()
	k.readyFunc = informer.Informer().HasSynced
}

func (k *KarydiaSecurityPolicyAdmission) Admit(ar v1beta1.AdmissionReview) *v1beta1.AdmissionResponse {
	if ignore, err := shouldIgnore(ar); err != nil {
		return toAdmissionResponse(err)
	} else if ignore {
		return &v1beta1.AdmissionResponse{
			Allowed: true,
		}
	}
	return k.computeSecurityContext(ar, true)
}

func (k *KarydiaSecurityPolicyAdmission) computeSecurityContext(ar v1beta1.AdmissionReview, specMutationAllowed bool) *v1beta1.AdmissionResponse {
	policies, err := k.lister.List(labels.Everything())
	if err != nil {
		return toAdmissionResponse(err)
	}

	if len(policies) == 0 {
		return toAdmissionResponse(fmt.Errorf("no karydia security policy found to validate against"))
	}

	// Sort policies by name to make order deterministic
	sort.SliceStable(policies, func(i, j int) bool {
		return strings.Compare(policies[i].ObjectMeta.Name, policies[j].ObjectMeta.Name) < 0
	})

	switch ar.Request.Resource {
	case resourcePod:
		return k.computeSecurityContextPod(ar, specMutationAllowed, policies)
	}

	return &v1beta1.AdmissionResponse{
		Allowed: true,
	}
}

func shouldIgnore(ar v1beta1.AdmissionReview) (bool, error) {
	// We don't care about 'CONNECT' and 'DELETE' currently
	// https://github.com/kubernetes/api/blob/kubernetes-1.12.2/admission/v1beta1/types.go#L118-L127
	if ar.Request.Operation != v1beta1.Create && ar.Request.Operation != v1beta1.Update {
		return true, nil
	}
	return false, nil
}
