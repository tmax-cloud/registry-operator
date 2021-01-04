package apiserver

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"

	regv1 "github.com/tmax-cloud/registry-operator/api/v1"
	v1 "github.com/tmax-cloud/registry-operator/pkg/apiserver/apis/v1"
	whv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	rbacv1 "k8s.io/api/rbac/v1"

	"github.com/gorilla/mux"
	"k8s.io/api/admission/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	apiregv1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/tmax-cloud/registry-operator/internal/utils"
	"github.com/tmax-cloud/registry-operator/internal/wrapper"
	"github.com/tmax-cloud/registry-operator/pkg/apiserver/apis"
)

const (
	Port = 24335
)

var log = ctrl.Log.WithName("extension-api-server")

type Server struct {
	Wrapper *wrapper.RouterWrapper
	Client  client.Client
}

func New() *Server {
	var err error

	server := &Server{}
	server.Wrapper = wrapper.New("/", nil, server.rootHandler)
	server.Wrapper.Router = mux.NewRouter()
	server.Wrapper.Router.HandleFunc("/", server.rootHandler)
	server.Wrapper.Router.HandleFunc("/mutate", server.mutateHandler)
	server.Wrapper.Router.HandleFunc("/imagesignrequest", server.imageSignRequestHandler)

	if err := apis.AddApis(server.Wrapper); err != nil {
		log.Error(err, "cannot add apis")
		os.Exit(1)
	}

	// Create CERT & Update Secret/ApiService
	opt := client.Options{}
	opt.Scheme = runtime.NewScheme()
	if err := apiregv1.AddToScheme(opt.Scheme); err != nil {
		log.Error(err, "cannot register scheme")
		os.Exit(1)
	}
	if err := corev1.AddToScheme(opt.Scheme); err != nil {
		log.Error(err, "cannot register scheme")
		os.Exit(1)
	}
	if err := whv1beta1.AddToScheme(opt.Scheme); err != nil {
		log.Error(err, "cannot register scheme")
		os.Exit(1)
	}
	if err := regv1.AddToScheme(opt.Scheme); err != nil {
		log.Error(err, "cannot register scheme")
		os.Exit(1)
	}
	if err := rbacv1.AddToScheme(opt.Scheme); err != nil {
		log.Error(err, "cannot register scheme")
		os.Exit(1)
	}

	server.Client, err = utils.Client(opt)
	if err != nil {
		log.Error(err, "cannot get client")
		os.Exit(1)
	}
	if err := createCert(context.TODO(), server.Client); err != nil {
		log.Error(err, "cannot create cert")
		os.Exit(1)
	}

	return server
}

func (s *Server) Start() {
	v1.Initiate()
	addr := fmt.Sprintf("0.0.0.0:%d", Port)
	log.Info(fmt.Sprintf("Server is running on %s", addr))

	cfg, err := tlsConfig(context.TODO(), s.Client)
	if err != nil {
		log.Error(err, "cannot get tls config")
		os.Exit(1)
	}

	httpServer := &http.Server{Addr: addr, Handler: s.Wrapper.Router, TLSConfig: cfg}
	if err := httpServer.ListenAndServeTLS(path.Join(CertDir, "tls.crt"), path.Join(CertDir, "tls.key")); err != nil {
		log.Error(err, "cannot launch server")
		os.Exit(1)
	}
}

func (s *Server) rootHandler(w http.ResponseWriter, _ *http.Request) {
	paths := metav1.RootPaths{}

	addPath(&paths.Paths, s.Wrapper)

	_ = utils.RespondJSON(w, paths)
}

var (
	runtimeScheme = runtime.NewScheme()
	codecs        = serializer.NewCodecFactory(runtimeScheme)
	deserializer  = codecs.UniversalDeserializer()
)

func (s *Server) mutateHandler(w http.ResponseWriter, r *http.Request) {
	paths := metav1.RootPaths{Paths: []string{"/mutate"}}
	addPath(&paths.Paths, s.Wrapper)

	var body []byte
	if r.Body != nil {
		if data, err := ioutil.ReadAll(r.Body); err == nil {
			body = data
		}
	}
	if len(body) == 0 {
		log.Error(nil, "empty body")
		http.Error(w, "empty body", http.StatusBadRequest)
		return
	}
	// verify the content type is accurate
	contentType := r.Header.Get("Content-Type")
	if contentType != "application/json" {
		log.Error(nil, "Content-Type=%s, expect application/json", contentType)
		http.Error(w, "invalid Content-Type, expect `application/json`", http.StatusUnsupportedMediaType)
		return
	}
	var admissionResponse *v1beta1.AdmissionResponse
	ar := v1beta1.AdmissionReview{}
	if _, _, err := deserializer.Decode(body, nil, &ar); err != nil {
		log.Error(nil, "Can't decode body: %v", err)
		admissionResponse = &v1beta1.AdmissionResponse{
			Result: &metav1.Status{
				Message: err.Error(),
			},
		}
	} else {
		admissionResponse = v1.Mutate(&ar, s.Client)
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
		log.Error(nil, "Can't encode response: %v", err)
		http.Error(w, fmt.Sprintf("could not encode response: %v", err), http.StatusInternalServerError)
	}
	log.Info("Ready to write reponse ...")
	if _, err := w.Write(resp); err != nil {
		log.Error(nil, "Can't write response: %v", err)
		http.Error(w, fmt.Sprintf("could not write response: %v", err), http.StatusInternalServerError)
	}
}

func (s *Server) imageSignRequestHandler(w http.ResponseWriter, r *http.Request) {
	paths := metav1.RootPaths{Paths: []string{"/imagesignrequest"}}
	addPath(&paths.Paths, s.Wrapper)

	var body []byte
	if r.Body != nil {
		if data, err := ioutil.ReadAll(r.Body); err == nil {
			body = data
		}
	}
	if len(body) == 0 {
		log.Error(nil, "empty body")
		http.Error(w, "empty body", http.StatusBadRequest)
		return
	}
	// verify the content type is accurate
	contentType := r.Header.Get("Content-Type")
	if contentType != "application/json" {
		log.Error(nil, "Content-Type=%s, expect application/json", contentType)
		http.Error(w, "invalid Content-Type, expect `application/json`", http.StatusUnsupportedMediaType)
		return
	}
	var admissionResponse *v1beta1.AdmissionResponse
	ar := v1beta1.AdmissionReview{}
	if _, _, err := deserializer.Decode(body, nil, &ar); err != nil {
		log.Error(nil, "Can't decode body: %v", err)
		admissionResponse = &v1beta1.AdmissionResponse{
			Result: &metav1.Status{
				Message: err.Error(),
			},
		}
	} else {
		admissionResponse = v1.ImageSignRequest(&ar, w, r)
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
		log.Error(nil, "Can't encode response: %v", err)
		http.Error(w, fmt.Sprintf("could not encode response: %v", err), http.StatusInternalServerError)
	}
	log.Info("Ready to write reponse ...")
	if _, err := w.Write(resp); err != nil {
		log.Error(nil, "Can't write response: %v", err)
		http.Error(w, fmt.Sprintf("could not write response: %v", err), http.StatusInternalServerError)
	}
}

func addPath(paths *[]string, w *wrapper.RouterWrapper) {
	if w.Handler != nil {
		*paths = append(*paths, w.FullPath())
	}

	for _, c := range w.Children {
		addPath(paths, c)
	}
}
