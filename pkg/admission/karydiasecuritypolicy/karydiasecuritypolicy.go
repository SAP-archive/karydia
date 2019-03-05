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

// Loosely modelled after the PSP admission plugin in Kubernetes:
// https://github.com/kubernetes/kubernetes/blob/v1.12.2/plugin/pkg/admission/security/podsecuritypolicy/admission.go

import (
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

	"github.com/karydia/karydia/pkg/apis/karydia/v1alpha1"
	informers "github.com/karydia/karydia/pkg/client/informers/externalversions"
	listers "github.com/karydia/karydia/pkg/client/listers/karydia/v1alpha1"
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

func (k *KarydiaSecurityPolicyAdmission) Admit(ar v1beta1.AdmissionReview, mutationAllowed bool) *v1beta1.AdmissionResponse {
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
	return k.computeSecurityContext(ar, mutationAllowed)
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

func (k *KarydiaSecurityPolicyAdmission) computeSecurityContext(ar v1beta1.AdmissionReview, mutationAllowed bool) *v1beta1.AdmissionResponse {
	apiserverUserInfo := transformUserInfo(ar.Request.UserInfo)
	policies, err := k.userPolicies(apiserverUserInfo, ar.Request.Namespace)
	if err != nil {
		k.logger.Errorf("failed to get karydia security policies for user %q: %v", ar.Request.UserInfo.Username, err)
		return &v1beta1.AdmissionResponse{
			Allowed: true,
		}
	}

	if len(policies) == 0 {
		k.logger.Warnf("no karydia security policy found for user %q in groups %v - request will be allowed w/o admission", ar.Request.UserInfo.Username, ar.Request.UserInfo.Groups)
		return &v1beta1.AdmissionResponse{
			Allowed: true,
		}
	}

	// Sort policies by name to make order deterministic
	sort.SliceStable(policies, func(i, j int) bool {
		return strings.Compare(policies[i].ObjectMeta.Name, policies[j].ObjectMeta.Name) < 0
	})

	var response *v1beta1.AdmissionResponse

	switch ar.Request.Resource {
	case resourcePod:
		response = k.computeSecurityContextPod(ar, mutationAllowed, policies)
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
