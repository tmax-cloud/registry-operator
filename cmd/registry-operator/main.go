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
	"flag"
	"github.com/go-logr/logr"
	"github.com/robfig/cron"
	tmaxiov1 "github.com/tmax-cloud/registry-operator/api/v1"
	"github.com/tmax-cloud/registry-operator/controllers"
	"github.com/tmax-cloud/registry-operator/internal/common/config"
	"github.com/tmax-cloud/registry-operator/pkg/apiserver"
	regmgr "github.com/tmax-cloud/registry-operator/pkg/manager"
	"github.com/tmax-cloud/registry-operator/server"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
	"io"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"os"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sync"
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
	webhook, err := apiserver.NewApiServer(ctrl.Log.WithName("apiserver"))
	if err != nil {
		setupLog.Error(err, "failed to create apiserver")
		os.Exit(1)
	}

	go func() {
		setupLog.Info("Start webhook on " + webhook.Addr)
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
