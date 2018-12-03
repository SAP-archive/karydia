package webhook

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/sirupsen/logrus"
	"k8s.io/api/admission/v1beta1"

	"github.com/kinvolk/karydia/pkg/k8sutil"
	"github.com/kinvolk/karydia/pkg/k8sutil/scheme"
)

type AdmissionPlugin interface {
	Admit(v1beta1.AdmissionReview) *v1beta1.AdmissionResponse
}

type Webhook struct {
	logger *logrus.Logger

	admissionPlugins []AdmissionPlugin
}

type Config struct {
	Logger *logrus.Logger
}

func New(config *Config) (*Webhook, error) {
	webhook := &Webhook{
		logger: config.Logger,
	}

	if config.Logger == nil {
		webhook.logger = logrus.New()
		webhook.logger.Level = logrus.InfoLevel
	} else {
		// convenience
		webhook.logger = config.Logger
	}

	return webhook, nil
}

func (wh *Webhook) RegisterAdmissionPlugin(p AdmissionPlugin) {
	wh.admissionPlugins = append(wh.admissionPlugins, p)
}

func (wh *Webhook) admit(ar v1beta1.AdmissionReview) *v1beta1.AdmissionResponse {
	var responseWithPatches *v1beta1.AdmissionResponse
	for _, ap := range wh.admissionPlugins {
		response := ap.Admit(ar)
		if !response.Allowed {
			return response
		}
		if response.Patch != nil && responseWithPatches == nil {
			responseWithPatches = response
			continue
		}
		if response.Patch != nil {
			return k8sutil.ErrToAdmissionResponse(fmt.Errorf("cannot admit with patches, request is patched already"))
		}
	}
	if responseWithPatches != nil {
		return responseWithPatches
	}
	return &v1beta1.AdmissionResponse{Allowed: true}
}

func (wh *Webhook) Serve(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		wh.logger.Errorf("received unexpected %s request, expecting POST", r.Method)
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	contentType := r.Header.Get("Content-Type")
	if contentType != "application/json" {
		wh.logger.Errorf("received request with unexpected content type %q", contentType)
		http.Error(w, http.StatusText(http.StatusUnsupportedMediaType), http.StatusUnsupportedMediaType)
		return
	}

	var body []byte
	if r.Body != nil {
		// Arbitrarily chosen limit of 32KB, adjust if necessary
		r.Body = http.MaxBytesReader(w, r.Body, 32000)
		var err error
		body, err = ioutil.ReadAll(r.Body)
		if err != nil {
			wh.logger.Errorf("failed to read request body: %v", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
	}

	if len(body) == 0 {
		wh.logger.Error("received request with empty body")
		http.Error(w, "empty body", http.StatusBadRequest)
		return
	}

	requestedAdmissionReview := v1beta1.AdmissionReview{}
	responseAdmissionReview := v1beta1.AdmissionReview{}

	deserializer := scheme.Codecs.UniversalDeserializer()
	if _, _, err := deserializer.Decode(body, nil, &requestedAdmissionReview); err != nil {
		wh.logger.Errorf("failed to decode body: %v", err)
		responseAdmissionReview.Response = k8sutil.ErrToAdmissionResponse(err)
	} else {
		wh.logger.Debugf("received admission review request: %+v", requestedAdmissionReview.Request)
		responseAdmissionReview.Response = wh.admit(requestedAdmissionReview)
	}

	// Make sure to return the request UID
	responseAdmissionReview.Response.UID = requestedAdmissionReview.Request.UID

	wh.logger.Debugf("sending admission review response: %+v", responseAdmissionReview.Response)

	respBytes, err := json.Marshal(responseAdmissionReview)
	if err != nil {
		wh.logger.Errorf("failed to marshal response: %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	if _, err := w.Write(respBytes); err != nil {
		wh.logger.Errorf("failed to send response: %v", err)
	}
}
