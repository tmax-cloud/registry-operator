package apis

import (
	"encoding/json"
	"fmt"
	"github.com/go-logr/logr"
	"github.com/tmax-cloud/registry-operator/internal/utils"
	"io/ioutil"
	"k8s.io/api/admission/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	authorization "k8s.io/client-go/kubernetes/typed/authorization/v1"
	"net/http"
)

var (
	runtimeScheme = runtime.NewScheme()
	codecs        = serializer.NewCodecFactory(runtimeScheme)
	deserializer  = codecs.UniversalDeserializer()
)

type AdmissionWebhook struct {
	c      *authorization.AuthorizationV1Client
	logger logr.Logger
}

func NewAdmissionWebhook(c *authorization.AuthorizationV1Client, logger logr.Logger) *AdmissionWebhook {
	return &AdmissionWebhook{
		c:      c,
		logger: logger,
	}
}

func (h *AdmissionWebhook) RootHandler(w http.ResponseWriter, _ *http.Request) {
	paths := metav1.RootPaths{}
	_ = utils.RespondJSON(w, paths)
}

func (h *AdmissionWebhook) MutateHandler(w http.ResponseWriter, r *http.Request) {
	var body []byte
	if r.Body != nil {
		if data, err := ioutil.ReadAll(r.Body); err == nil {
			body = data
		}
	}
	if len(body) == 0 {
		h.logger.Error(nil, "empty body")
		http.Error(w, "empty body", http.StatusBadRequest)
		return
	}
	// verify the content type is accurate
	contentType := r.Header.Get("Content-Type")
	if contentType != "application/json" {
		h.logger.Error(nil, "Content-Type=%s, expect application/json", contentType)
		http.Error(w, "invalid Content-Type, expect `application/json`", http.StatusUnsupportedMediaType)
		return
	}
	var admissionResponse *v1beta1.AdmissionResponse
	ar := v1beta1.AdmissionReview{}
	if _, _, err := deserializer.Decode(body, nil, &ar); err != nil {
		h.logger.Error(nil, "Can't decode body: %v", err)
		admissionResponse = &v1beta1.AdmissionResponse{
			Result: &metav1.Status{
				Message: err.Error(),
			},
		}
	} else {
		admissionResponse = h.Mutate(&ar)
	}
	admissionReview := v1beta1.AdmissionReview{}
	if admissionResponse != nil {
		admissionReview.Response = admissionResponse
		if ar.Request != nil {
			admissionReview.Response.UID = ar.Request.UID
		}
	}

	resp, err := json.Marshal(admissionReview)
	if err != nil {
		h.logger.Error(nil, "Can't encode response: %v", err)
		http.Error(w, fmt.Sprintf("could not encode response: %v", err), http.StatusInternalServerError)
	}
	h.logger.Info("Ready to write reponse ...")
	if _, err := w.Write(resp); err != nil {
		h.logger.Error(nil, "Can't write response: %v", err)
		http.Error(w, fmt.Sprintf("could not write response: %v", err), http.StatusInternalServerError)
	}
}

func (h *AdmissionWebhook) ImageSignRequestHandler(w http.ResponseWriter, r *http.Request) {
	var body []byte
	if r.Body != nil {
		if data, err := ioutil.ReadAll(r.Body); err == nil {
			body = data
		}
	}
	if len(body) == 0 {
		h.logger.Error(nil, "empty body")
		http.Error(w, "empty body", http.StatusBadRequest)
		return
	}
	// verify the content type is accurate
	contentType := r.Header.Get("Content-Type")
	if contentType != "application/json" {
		h.logger.Error(nil, "Content-Type=%s, expect application/json", contentType)
		http.Error(w, "invalid Content-Type, expect `application/json`", http.StatusUnsupportedMediaType)
		return
	}
	var admissionResponse *v1beta1.AdmissionResponse
	ar := v1beta1.AdmissionReview{}
	if _, _, err := deserializer.Decode(body, nil, &ar); err != nil {
		h.logger.Error(nil, "Can't decode body: %v", err)
		admissionResponse = &v1beta1.AdmissionResponse{
			Result: &metav1.Status{
				Message: err.Error(),
			},
		}
	} else {
		admissionResponse = h.ImageSignRequest(&ar, w, r)
	}
	admissionReview := v1beta1.AdmissionReview{}
	if admissionResponse != nil {
		admissionReview.Response = admissionResponse
		if ar.Request != nil {
			admissionReview.Response.UID = ar.Request.UID
		}
	}

	resp, err := json.Marshal(admissionReview)
	if err != nil {
		h.logger.Error(nil, "Can't encode response: %v", err)
		http.Error(w, fmt.Sprintf("could not encode response: %v", err), http.StatusInternalServerError)
	}
	h.logger.Info("Ready to write reponse ...")
	if _, err := w.Write(resp); err != nil {
		h.logger.Error(nil, "Can't write response: %v", err)
		http.Error(w, fmt.Sprintf("could not write response: %v", err), http.StatusInternalServerError)
	}
}
