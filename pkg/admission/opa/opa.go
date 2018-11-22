package opa

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/Azure/kubernetes-policy-controller/pkg/opa"
	"github.com/Azure/kubernetes-policy-controller/pkg/policies/types"
	logger "github.com/sirupsen/logrus"
	"k8s.io/api/admission/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kinvolk/karydia/pkg/k8sutil"

	"github.com/open-policy-agent/opa/util"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

type OPAAdmission struct {
	opaClient opa.Client
}

type Config struct {
	OPAURL string
}

func New(config *Config) (*OPAAdmission, error) {
	opaClient := opa.New(config.OPAURL)
	return &OPAAdmission{
		opaClient: opaClient,
	}, nil
}

var (
	resourcePod              = metav1.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}
	resourcePersistentVolume = metav1.GroupVersionResource{Group: "", Version: "v1", Resource: "persistentvolumes"}
)

func (o *OPAAdmission) Admit(ar v1beta1.AdmissionReview) *v1beta1.AdmissionResponse {
	resource := strings.ToLower(ar.Request.Resource.Resource)
	namespace := strings.ToLower(ar.Request.Namespace)
	name := strings.ToLower(ar.Request.Name)
	if name == "" {
		// For example new pods created by a replicaset don't
		// have a name yet during the 'create' operation. Set
		// and use a random name then during admission then.
		name = randStr(10)
	}

	// Prepare query for recently added POST /query endpoint
	// https://github.com/open-policy-agent/opa/pull/1028/commits/69d18b0525ef36314db96d4125fa040dcb0ccb77
	opaQueryDataPath := fmt.Sprintf(`data["kubernetes"]["%s"]["%s"]["%s"]`, resource, namespace, name)
	opaQueryPayload := fmt.Sprintf(`data.admission.deny[{
"id": id,
"resource": {"kind": "%s", "namespace": "%s", "name": "%s"},
"resolution": resolution,}]`,
		resource, namespace, name)
	opaQuery := fmt.Sprintf(`%s with %s as %s`, opaQueryPayload, opaQueryDataPath, ar.Request.Object.Raw[:])

	logger.Infof("OPA query:\n%v", opaQuery)

	resp, err := o.opaClient.PostQuery(opaQuery)
	if err != nil && !opa.IsUndefinedErr(err) {
		logger.Errorf("OPA query failed: %v", err)
		return k8sutil.ErrToAdmissionResponse(err)
	}
	logger.Infof("OPA query response:\n%+v", resp)

	allowed, reason, patchBytes, err := createPatchFromOPAResult(resp)
	if err != nil {
		logger.Errorf("cannot handle OPA response: %v", err)
		return &v1beta1.AdmissionResponse{Allowed: allowed}
	}
	if patchBytes != nil && len(patchBytes) != 0 {
		logger.Errorf("patches not implemented yet")
		return &v1beta1.AdmissionResponse{Allowed: allowed}
	}

	return &v1beta1.AdmissionResponse{
		Allowed: allowed,
		Result: &metav1.Status{
			Message: reason,
		},
	}
}

// Borrowed from https://github.com/Azure/kubernetes-policy-controller/blob/master/pkg/server/server.go
func createPatchFromOPAResult(result []map[string]interface{}) (bool, string, []byte, error) {
	var msg string
	bs, err := json.Marshal(result)
	if err != nil {
		return false, msg, nil, err
	}
	var allViolations []types.Deny
	err = util.UnmarshalJSON(bs, &allViolations)
	if err != nil {
		return false, msg, nil, err
	}
	if len(allViolations) == 0 {
		return true, "valid based on configured policies", nil, nil
	}
	valid := true
	var reason struct {
		Reason []string `json:"reason,omitempty"`
	}
	validPatches := map[string]types.PatchOperation{}
	for _, v := range allViolations {
		patchCount := len(v.Resolution.Patches)
		if patchCount == 0 {
			valid = false
			reason.Reason = append(reason.Reason, v.Resolution.Message)
			continue
		}
		for _, p := range v.Resolution.Patches {
			if existing, ok := validPatches[p.Path]; ok {
				msg = fmt.Sprintf("conflicting patches caused denied request, operations (%+v, %+v)", p, existing)
				return false, msg, nil, nil
			}
			validPatches[p.Path] = p
		}
	}
	if !valid {
		if bs, err := json.Marshal(reason.Reason); err == nil {
			msg = string(bs)
		}
		return false, msg, nil, nil
	}
	var patches []interface{}
	for _, p := range validPatches {
		patches = append(patches, p)
	}
	if len(patches) == 0 {
		panic(fmt.Errorf("unexpected no valid patches found, %+v", allViolations))
	}
	patchBytes, err := json.Marshal(patches)
	if err != nil {
		return false, "", nil, fmt.Errorf("failed creating patches, patches=%+v err=%v", patches, err)
	}

	return true, "applying patches", patchBytes, nil
}

var runes = []rune("abcdefghijklmnopqrstuvwxyz0123456789")

func randStr(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = runes[rand.Intn(len(runes))]
	}
	return string(b)
}
