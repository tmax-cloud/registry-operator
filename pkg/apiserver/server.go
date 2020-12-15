package apiserver

import (
	"context"
	"fmt"
	v1 "github.com/tmax-cloud/registry-operator/pkg/apiserver/apis/v1"
	"net/http"
	"os"
	"path"

	"github.com/gorilla/mux"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
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

func addPath(paths *[]string, w *wrapper.RouterWrapper) {
	if w.Handler != nil {
		*paths = append(*paths, w.FullPath())
	}

	for _, c := range w.Children {
		addPath(paths, c)
	}
}
