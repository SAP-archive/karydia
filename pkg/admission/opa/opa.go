package opa

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/Azure/kubernetes-policy-controller/pkg/opa"
	logger "github.com/sirupsen/logrus"
	"k8s.io/api/admission/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kinvolk/karydia/pkg/k8sutil"
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

	return &v1beta1.AdmissionResponse{
		Allowed: true,
	}
}

var runes = []rune("abcdefghijklmnopqrstuvwxyz0123456789")

func randStr(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = runes[rand.Intn(len(runes))]
	}
	return string(b)
}
