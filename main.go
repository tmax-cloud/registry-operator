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
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/tmax-cloud/registry-operator/pkg/apiserver"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/tmax-cloud/registry-operator/internal/common/config"
	"github.com/tmax-cloud/registry-operator/internal/common/operatorlog"
	regmgr "github.com/tmax-cloud/registry-operator/pkg/manager"
	"github.com/tmax-cloud/registry-operator/pkg/scheduler"

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

	logFile, err := operatorlog.LogFile()
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	defer logFile.Close()

	// logging to stdout stream and a log file
	w := io.MultiWriter(logFile, os.Stdout)
	ctrl.SetLogger(zap.New(zap.UseDevMode(true), zap.WriteTo(w)))
	setupLog.Info("logging to a file", "filepath", logFile.Name())

	// backup Logfile daily
	operatorlog.StartDailyBackup(logFile)

	// set config
	config.InitEnv()
	config.ReadInConfig()
	config.PrintConfig()
	config.OnConfigChange(3)

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: metricsAddr,
		Port:               9443,
		LeaderElection:     enableLeaderElection,
		LeaderElectionID:   "8f6e6509.io",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	s := scheduler.New(mgr.GetClient(), mgr.GetScheme())

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
	if err = (&controllers.RegistryJobReconciler{
		Client:    mgr.GetClient(),
		Log:       ctrl.Log.WithName("controllers").WithName("RegistryJob"),
		Scheme:    mgr.GetScheme(),
		Scheduler: s,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "RegistryJob")
		os.Exit(1)
	}
	controllers.StartRegistryCronJobController(mgr.GetClient(), ctrl.Log.WithName("controllers").WithName("RegistryCronJob"), mgr.GetScheme())
	if err = (&controllers.ExternalRegistryReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("ExternalRegistry"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr, s); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ExternalRegistry")
		os.Exit(1)
	}

	if err = (&controllers.ImageReplicateReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("ImageReplicate"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr, s); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ImageReplicate")
		os.Exit(1)
	}
	// +kubebuilder:scaffold:builder

	// API Server
	apiServer := apiserver.New()
	go apiServer.Start()

	var wg sync.WaitGroup
	wg.Add(2)

	// Added for registry
	setupLog.Info("Start web server")
	go func() {
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
