package apiserver

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"github.com/gorilla/mux"
	regv1 "github.com/tmax-cloud/registry-operator/api/v1"
	"github.com/tmax-cloud/registry-operator/internal/utils"
	wbapi "github.com/tmax-cloud/registry-operator/pkg/apiserver/apis"
	whapiv1 "github.com/tmax-cloud/registry-operator/pkg/apiserver/apis/v1"
	"io/ioutil"
	admissionv1 "k8s.io/api/admissionregistration/v1beta1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/util/cert"
	apiregv1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"
	certResources "knative.dev/pkg/webhook/certificates/resources"
	"net/http"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"time"
)

var (
	scheme = runtime.NewScheme()
)

func init() {
	utilruntime.Must(apiregv1.AddToScheme(scheme))
	utilruntime.Must(corev1.AddToScheme(scheme))
	utilruntime.Must(admissionv1.AddToScheme(scheme))
	utilruntime.Must(regv1.AddToScheme(scheme))
	utilruntime.Must(rbacv1.AddToScheme(scheme))
	// +kubebuilder:scaffold:scheme
}

type ApiServer struct {
	http.Server
}

func NewApiServer() (*ApiServer, error) {
	r := mux.NewRouter()
	r.HandleFunc("/", wbapi.RootHandler)
	r.HandleFunc("/mutate", wbapi.MutateHandler)
	r.HandleFunc("/imagesignrequest", wbapi.ImageSignRequestHandler)

	s := r.PathPrefix("/apis/registry.tmax.io").Subrouter()
	s.Use(whapiv1.Authenticate)

	s.HandleFunc("/", whapiv1.ApisHandler)
	s.HandleFunc("/v1", whapiv1.VersionHandler)
	s.HandleFunc("/v1/namespaces/{namespace}/scans/{scanReqName}", whapiv1.ScanRequestHandler)
	s.HandleFunc("/v1/namespaces/{namespace}/ext-scans/{ext-scanReqName}", whapiv1.ExtScanRequestHandler)
	s.HandleFunc("/v1/namespaces/{namespace}/repositories/{repositoryName}/imagescanresults", whapiv1.ListScanSummaryHandler)
	s.HandleFunc("/v1/namespaces/{namespace}/repositories/{repositoryName}/imagescanresults/{tagName}", whapiv1.ScanResultHandler)
	s.HandleFunc("/v1/namespaces/{namespace}/ext-repositories/repositoryName}/imagescanresults", whapiv1.ListExtScanSummaryHandler)
	s.HandleFunc("/v1/namespaces/{namespace}/ext-repositories/{repositoryName}/imagescanresults/{tagName}", whapiv1.ExtScanResultHandler)

	cfg, err := config.GetConfig()
	if err != nil {
		return nil, err
	}
	c, err := client.New(cfg, client.Options{Scheme: scheme})
	if err != nil {
		return nil, err
	}

	if err = renewCertForWebhook(c); err != nil {
		return nil, err
	}

	configMap := &corev1.ConfigMap{}
	if err = c.Get(context.TODO(), types.NamespacedName{
		Namespace: metav1.NamespaceSystem,
		Name:      "extension-apiserver-authentication",
	}, configMap); err != nil {
		return nil, fmt.Errorf("failed to get 'extension-apiserver-authentication' configmap")
	}
	clientCA, ok := configMap.Data["requestheader-client-ca-file"]
	if !ok {
		return nil, fmt.Errorf("failed to get requestheader-client CA")
	}
	certs, err := cert.ParseCertsPEM([]byte(clientCA))
	if err != nil {
		return nil, fmt.Errorf("failed to parse requestheader-client CA PEM")
	}
	caPool := x509.NewCertPool()
	for _, c := range certs {
		caPool.AddCert(c)
	}

	//v1.Initiate()

	webhook := http.Server{
		Addr:    "0.0.0.0:24335",
		Handler: r,
		TLSConfig: &tls.Config{
			ClientCAs:  caPool,
			ClientAuth: tls.VerifyClientCertIfGiven,
		},
	}

	return &ApiServer{
		Server: webhook,
	}, nil
}

func renewCertForWebhook(c client.Client) error {
	if err := os.MkdirAll("/tmp/run-api", os.ModePerm); err != nil {
		return err
	}

	// Get service name and namespace
	svc := utils.OperatorServiceName()
	ns, err := utils.Namespace()
	if err != nil {
		return err
	}

	// Create certs
	ctx := context.Background()

	tlsKey, tlsCrt, caCrt, err := certResources.CreateCerts(ctx, svc, ns, time.Now().AddDate(10, 0, 0))
	if err != nil {
		return err
	}
	err = ioutil.WriteFile("/tmp/run-api/tls.key", tlsKey, 0644)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile("/tmp/run-api/tls.crt", tlsCrt, 0644)
	if err != nil {
		return err
	}

	apiService := &apiregv1.APIService{}
	if err = c.Get(ctx, types.NamespacedName{Name: "v1.registry.tmax.io"}, apiService); err != nil {
		return err
	}
	apiService.Spec.CABundle = caCrt
	if err = c.Update(ctx, apiService); err != nil {
		return err
	}

	mwConfig := &admissionv1.MutatingWebhookConfiguration{}
	if err = c.Get(ctx, types.NamespacedName{Name: "registry-operator-webhook-cfg"}, mwConfig); err != nil {
		return err
	}

	if len(mwConfig.Webhooks) != 2 {
		return fmt.Errorf("MutatingWebhookConfiguration's webhook must be two, but there is/are %d", len(mwConfig.Webhooks))
	}

	mwConfig.Webhooks[0].ClientConfig.CABundle = caCrt
	mwConfig.Webhooks[1].ClientConfig.CABundle = caCrt

	if err = c.Update(ctx, mwConfig); err != nil {
		return err
	}

	return nil
}
