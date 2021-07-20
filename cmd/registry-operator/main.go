/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"github.com/go-logr/logr"
	"github.com/gorilla/mux"
	"github.com/robfig/cron"
	"github.com/tmax-cloud/registry-operator/internal/utils"
	wbapi "github.com/tmax-cloud/registry-operator/pkg/apiserver"
	v1 "github.com/tmax-cloud/registry-operator/pkg/apiserver/apis/v1"
	whapiv1 "github.com/tmax-cloud/registry-operator/pkg/apiserver/apis/v1"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
	"io"
	"io/ioutil"
	"k8s.io/api/admissionregistration/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/cert"
	apiregv1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"
	certResources "knative.dev/pkg/webhook/certificates/resources"
	"net/http"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sync"
	"time"

	"github.com/tmax-cloud/registry-operator/internal/common/config"
	regmgr "github.com/tmax-cloud/registry-operator/pkg/manager"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"

	tmaxiov1 "github.com/tmax-cloud/registry-operator/api/v1"
	"github.com/tmax-cloud/registry-operator/controllers"
	"github.com/tmax-cloud/registry-operator/server"
	// +kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(tmaxiov1.AddToScheme(scheme))
	// +kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	flag.StringVar(&metricsAddr, "metrics-addr", ":8080", "The address the metric endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "enable-leader-election", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.Parse()

	// set config
	config.InitEnv()
	config.ReadInConfig()
	config.PrintConfig()
	config.OnConfigChange(3)

	ctrl.SetLogger(createDailyRotateLogger("/var/log/registry-operator/operator.log"))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: metricsAddr,
		Port:               9443,
		LeaderElection:     enableLeaderElection,
		LeaderElectionID:   "8f6e6510.io",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if err = (&controllers.RegistryReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("Registry"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Registry")
		os.Exit(1)
	}
	if err = (&controllers.RepositoryReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("Repository"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Repository")
		os.Exit(1)
	}
	if err = (&controllers.NotaryReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("Notary"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Notary")
		os.Exit(1)
	}
	if err = (&controllers.ImageSignerReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("ImageSigner"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ImageSigner")
		os.Exit(1)
	}
	if err = (&controllers.ImageSignRequestReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("ImageSignRequest"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ImageSignRequest")
		os.Exit(1)
	}
	if err = (&controllers.ImageScanRequestReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("ImageScanRequest"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ImageScanRequest")
		os.Exit(1)
	}
	if err = (&controllers.ExternalRegistryReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("ExternalRegistry"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ExternalRegistry")
		os.Exit(1)
	}
	if err = (&controllers.ImageReplicateReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("ImageReplicate"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ImageReplicate")
		os.Exit(1)
	}
	// +kubebuilder:scaffold:builder

	// API Server
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

	ctx := context.Background()
	if err = renewCertForWebhook(ctx, mgr.GetClient()); err != nil {
		setupLog.Error(err, "failed to setup apiservice")
		os.Exit(1)
	}
	v1.Initiate()

	webhook := &http.Server{
		Addr:    "0.0.0.0:24335",
		Handler: r,
		TLSConfig: &tls.Config{
			ClientCAs: func() *x509.CertPool {
				extensionApiServerAuthConfigMap := &corev1.ConfigMap{}
				requestConfigMap := types.NamespacedName{Name: "extension-apiserver-authentication", Namespace: metav1.NamespaceSystem}
				if err = mgr.GetClient().Get(ctx, requestConfigMap, extensionApiServerAuthConfigMap); err != nil {
					setupLog.Error(err, "failed to get configuration for webhook")
					os.Exit(1)
				}
				clientCA, ok := extensionApiServerAuthConfigMap.Data["requestheader-client-ca-file"]
				if !ok {
					setupLog.Error(fmt.Errorf("not found key: requestheader-client-ca-file"), "failed to get configuration for webhook")
					os.Exit(1)
				}
				certs, err := cert.ParseCertsPEM([]byte(clientCA))
				if err != nil {
					setupLog.Error(err, "failed to get configuration for webhook")
					os.Exit(1)
				}
				caPool := x509.NewCertPool()
				for _, c := range certs {
					caPool.AddCert(c)
				}
				return caPool
			}(),
			ClientAuth: tls.VerifyClientCertIfGiven,
		},
	}

	go func() {
		setupLog.Info("Start webhook on ", webhook.Addr)
		if err = webhook.ListenAndServeTLS("/tmp/run-api/tls.crt", "/tmp/run-api/tls.key"); err != nil {
			setupLog.Error(err, "failed to launch webhook server")
			os.Exit(1)
		}
	}()

	var wg sync.WaitGroup
	wg.Add(2)

	// Added for registry

	go func() {
		setupLog.Info("Start registry webhook server")
		server.StartServer(mgr)
		wg.Done()
	}()

	// Start manager
	go func() {
		setupLog.Info("starting manager")
		if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
			setupLog.Error(err, "problem running manager")
			os.Exit(1)
		}
		wg.Done()
	}()

	// Synchronize All Registries
	if err := regmgr.SyncAllRegistry(mgr.GetClient(), mgr.GetScheme()); err != nil {
		setupLog.Error(err, "failed to synchronize all registries")
	}

	if err := regmgr.SyncAllRepoSigner(mgr.GetClient(), mgr.GetScheme()); err != nil {
		setupLog.Error(err, "failed to synchronize all repository signers")
	}

	// Wait until webserver and manager is over
	wg.Wait()
}

func createDailyRotateLogger(logpath string) logr.Logger {
	lumberlogger := &lumberjack.Logger{
		Filename: logpath,
		MaxSize:  500, // megabytes
		//MaxAge:     90, // days
	}
	mw := io.MultiWriter(os.Stdout, zapcore.AddSync(lumberlogger))

	cronJob := cron.New()
	if err := cronJob.AddFunc("@daily", func() {
		if err := lumberlogger.Rotate(); err != nil {
			setupLog.Error(err, "failed to rotate log.")
		}
	}); err != nil {
		setupLog.Error(err, "failed to add cronjob for logging")
		os.Exit(1)
	}
	defer cronJob.Start()

	return zap.New(zap.UseDevMode(true), zap.WriteTo(mw))
}

func renewCertForWebhook(ctx context.Context, c client.Client) error {
	if err := os.MkdirAll("/tmp/run-api", os.ModePerm); err != nil {
		return err
	}
	svc := utils.OperatorServiceName()
	ns, err := utils.Namespace()
	if err != nil {
		return err
	}

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

	mwConfig := &v1beta1.MutatingWebhookConfiguration{}
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
