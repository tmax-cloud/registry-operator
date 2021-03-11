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

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	exreghandler "github.com/tmax-cloud/registry-operator/controllers/exregctl/handler"
	replhandler "github.com/tmax-cloud/registry-operator/controllers/replicatectl/handler"
	"github.com/tmax-cloud/registry-operator/internal/common/operatorlog"
	"github.com/tmax-cloud/registry-operator/pkg/scheduler"

	tmaxiov1 "github.com/tmax-cloud/registry-operator/api/v1"
	"github.com/tmax-cloud/registry-operator/controllers"
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
	if err := exreghandler.RegisterHandler(mgr, s); err != nil {
		setupLog.Error(err, "unable to register handler", "handler", "ExternalRegistry")
		os.Exit(1)
	}
	if err := replhandler.RegisterHandler(mgr, s); err != nil {
		setupLog.Error(err, "unable to register handler", "handler", "ImageReplicate")
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

	// +kubebuilder:scaffold:builder

	// Start job operator
	setupLog.Info("starting job operator")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running job operator")
		os.Exit(1)
	}
}
