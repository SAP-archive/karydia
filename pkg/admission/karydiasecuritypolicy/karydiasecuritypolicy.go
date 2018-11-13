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
	authenticationv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apiserver/pkg/authentication/user"
	"k8s.io/apiserver/pkg/authorization/authorizer"

	"github.com/kinvolk/karydia/pkg/apis/karydia/v1alpha1"
	informers "github.com/kinvolk/karydia/pkg/client/informers/externalversions"
	listers "github.com/kinvolk/karydia/pkg/client/listers/karydia/v1alpha1"
)

var (
	resourcePod = metav1.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}
)

type KarydiaSecurityPolicyAdmission struct {
	authz authorizer.Authorizer

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

func (k *KarydiaSecurityPolicyAdmission) SetAuthorizer(authz authorizer.Authorizer) {
	k.authz = authz
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

func transformUserInfo(arUserInfo authenticationv1.UserInfo) *user.DefaultInfo {
	apiserverUserInfo := &user.DefaultInfo{
		Name:   arUserInfo.Username,
		UID:    arUserInfo.UID,
		Groups: arUserInfo.Groups,
		Extra:  make(map[string][]string),
	}
	for k, v := range arUserInfo.Extra {
		apiserverUserInfo.Extra[k] = v
	}
	return apiserverUserInfo
}

func (k *KarydiaSecurityPolicyAdmission) computeSecurityContext(ar v1beta1.AdmissionReview, specMutationAllowed bool) *v1beta1.AdmissionResponse {
	apiserverUserInfo := transformUserInfo(ar.Request.UserInfo)
	policies, err := k.userPolicies(apiserverUserInfo, ar.Request.Namespace)
	if err != nil {
		return toAdmissionResponse(err)
	}

	if len(policies) == 0 {
		k.logger.Warnf("no karydia security policy found for user %q in groups %v", ar.Request.UserInfo.Username, ar.Request.UserInfo.Groups)
		return toAdmissionResponse(fmt.Errorf("no karydia security policy found to validate against"))
	}

	// Sort policies by name to make order deterministic
	sort.SliceStable(policies, func(i, j int) bool {
		return strings.Compare(policies[i].ObjectMeta.Name, policies[j].ObjectMeta.Name) < 0
	})

	var response *v1beta1.AdmissionResponse

	switch ar.Request.Resource {
	case resourcePod:
		response = k.computeSecurityContextPod(ar, specMutationAllowed, policies)
	default:
		response = &v1beta1.AdmissionResponse{
			Allowed: true,
		}
	}

	return response

}

func (k *KarydiaSecurityPolicyAdmission) userPolicies(userInfo user.Info, namespace string) ([]*v1alpha1.KarydiaSecurityPolicy, error) {
	var userPolicies []*v1alpha1.KarydiaSecurityPolicy

	policies, err := k.lister.List(labels.Everything())
	if err != nil {
		return nil, err
	}

	for _, policy := range policies {
		if k.isAuthorizedForPolicyInAPIGroup(userInfo, namespace, policy.ObjectMeta.Name, k.authz) {
			userPolicies = append(userPolicies, policy)
		}
	}

	return userPolicies, nil
}

func (k *KarydiaSecurityPolicyAdmission) isAuthorizedForPolicyInAPIGroup(userInfo user.Info, namespace, policyName string, authz authorizer.Authorizer) bool {
	if userInfo == nil {
		return false
	}
	attr := authorizer.AttributesRecord{
		User:            userInfo,
		Verb:            "use",
		Namespace:       namespace,
		Name:            policyName,
		APIGroup:        "karydia.gardener.cloud",
		Resource:        "karydiasecuritypolicies",
		ResourceRequest: true,
	}
	decision, reason, err := authz.Authorize(attr)
	if err != nil {
		k.logger.Infof("cannot authorize for policy: %v,%v", reason, err)
	}
	var decisionStr string
	switch decision {
	case authorizer.DecisionDeny:
		decisionStr = "deny"
	case authorizer.DecisionAllow:
		decisionStr = "allow"
	case authorizer.DecisionNoOpinion:
		decisionStr = "no opinion"
	}
	k.logger.Debugf("policy authorizer decision: %q user: %q groups: %v policy: %q namespace: %q", decisionStr, userInfo.GetName(), userInfo.GetGroups(), policyName, namespace)
	return (decision == authorizer.DecisionAllow)
}

func shouldIgnore(ar v1beta1.AdmissionReview) (bool, error) {
	// We don't care about 'CONNECT' and 'DELETE' currently
	// https://github.com/kubernetes/api/blob/kubernetes-1.12.2/admission/v1beta1/types.go#L118-L127
	if ar.Request.Operation != v1beta1.Create && ar.Request.Operation != v1beta1.Update {
		return true, nil
	}
	return false, nil
}
