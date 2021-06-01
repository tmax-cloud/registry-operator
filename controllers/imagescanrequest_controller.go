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

package controllers

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/genuinetools/reg/clair"
	"github.com/genuinetools/reg/registry"
	"github.com/genuinetools/reg/repoutils"
	"github.com/go-logr/logr"
	secureReg "github.com/tmax-cloud/registry-operator/pkg/scan/clair"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	tmaxiov1 "github.com/tmax-cloud/registry-operator/api/v1"
	"github.com/tmax-cloud/registry-operator/controllers/scanctl"
	"github.com/tmax-cloud/registry-operator/internal/common/certs"
	"github.com/tmax-cloud/registry-operator/internal/common/config"
	"github.com/tmax-cloud/registry-operator/internal/utils/k8s/secrethelper"
)

// ImageScanRequestReconciler reconciles a ImageScanRequest object
type ImageScanRequestReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

var (
	worker *scanctl.ScanWorker
	// FIXME: Remove clair library dependency
	scanner *clair.Clair
	verbose = false
)

const (
	requestQueueSize = 100
	nWorkers         = 5
	timeout          = time.Second * 30
)

func init() {
	// TODO: Load value from operator config
	worker = scanctl.NewScanWorker(requestQueueSize, nWorkers)
	worker.Start()

	// FIXME: Promote using ENV env to operator scope
	switch os.Getenv("ENV") {
	case "dev":
		verbose = true
	case "prod":
		verbose = false
	default:
		verbose = true
	}

	// FIXME: Regenerate instance on change manager config
	scanner, _ = clair.New(config.Config.GetString(config.ConfigImageScanSvr), clair.Opt{
		Debug:    verbose,
		Timeout:  timeout,
		Insecure: config.Config.GetBool("scanning.scanner.insecure"),
	})
}

// +kubebuilder:rbac:groups=tmax.io,resources=imagescanrequests,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=tmax.io,resources=imagescanrequests/status,verbs=get;update;patch

func (r *ImageScanRequestReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	logger := r.Log.WithValues("Request.Namespace", req.Namespace, "Request.Name", req.Name)
	logger.Info("Reconciling Scanning")

	instance := &tmaxiov1.ImageScanRequest{}
	err := r.Client.Get(context.TODO(), req.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return ctrl.Result{}, err
	}

	switch instance.Status.Status {
	case "":
		err = r.doRecept(instance)
	case tmaxiov1.ScanRequestPending:
		logger.Info("Already pending request...")
		// XXX: Cancel job?
		// return ctrl.Result{Requeue: true}, nil
	case tmaxiov1.ScanRequestProcessing:
		logger.Info("Already in procssing...")
		// return ctrl.Result{Requeue: true}, nil
	case tmaxiov1.ScanRequestSuccess:
		logger.Info("Already Success request")
		// err = r.doRecept(instance)
	case tmaxiov1.ScanRequestFail:
		logger.Info("Already Failed request")
		// err = r.doRecept(instance)
	}

	if err != nil {
		logger.Error(err, "")
		_ = r.updateStatus(instance, tmaxiov1.ScanRequestError, err.Error(), nil)
	}

	return ctrl.Result{}, nil
}

func (r *ImageScanRequestReconciler) doRecept(instance *tmaxiov1.ImageScanRequest) error {

	// validation phase
	// TODO: Move this to validation webhook
	if instance.Spec.MaxFixable < 0 {
		return fmt.Errorf("fixable threshold must be a positive integer")
	}

	// preprocess phase
	jobs := []*scanctl.ScanJob{}
	for _, e := range instance.Spec.ScanTargets {
		ctx := context.TODO()

		// FIXME: Is here the right place to get keyclock cert?
		tlsSecret := &corev1.Secret{}
		if err := r.Client.Get(ctx, types.NamespacedName{Name: tmaxiov1.RegistryRootCASecretName, Namespace: config.Config.GetString("operator.namespace")}, tlsSecret); err != nil {
			return fmt.Errorf("TLS Secret not found: %s\n", tmaxiov1.RegistryRootCASecretName)
		}
		tlsCertData, err := secrethelper.GetCert(tlsSecret, certs.RootCACert)
		if err != nil {
			return err
		}

		if len(e.CertificateSecret) > 0 {
			tlsSecret := &corev1.Secret{}
			if err := r.Client.Get(ctx, types.NamespacedName{Name: e.CertificateSecret, Namespace: instance.Namespace}, tlsSecret); err != nil {
				return fmt.Errorf("TLS Secret not found: %s\n", e.CertificateSecret)
			}

			privateCertData, err := secrethelper.GetCert(tlsSecret, certs.RootCACert)
			if err != nil {
				return err
			}

			tlsCertData = append(tlsCertData, []byte("\n")...)
			tlsCertData = append(tlsCertData, privateCertData...)
		}

		username := ""
		password := ""
		// XXX: Is it right default docker.io when empty registry url?
		if len(e.RegistryURL) == 0 || e.RegistryURL == "docker.io" {
			e.RegistryURL = "https://registry-1.docker.io"
		}

		if strings.HasPrefix(e.RegistryURL, "http://") || strings.HasPrefix(e.RegistryURL, "https://") {
			return fmt.Errorf("registry url must not have protocol(http, https).")
		}

		// FIXME: needs handle http
		targetUrl := "https://" + e.RegistryURL
		_, err = url.ParseRequestURI(targetUrl)
		if err != nil {
			return err
		}

		if len(e.ImagePullSecret) > 0 {
			secret := &corev1.Secret{}
			if err := r.Client.Get(ctx, types.NamespacedName{Name: e.ImagePullSecret, Namespace: instance.Namespace}, secret); err != nil {
				return fmt.Errorf("ImagePullSecret not found: %s\n", e.ImagePullSecret)
			}

			imagePullSecret, err := secrethelper.NewImagePullSecret(secret)
			if err != nil {
				return err
			}

			login, err := imagePullSecret.GetHostCredential(targetUrl)
			if err != nil {
				return err
			}
			username = login.Username
			password = string(login.Password)
		}

		authCfg, err := repoutils.GetAuthConfig(username, password, targetUrl)
		if err != nil {
			return err
		}

		r, err := secureReg.New(context.TODO(), authCfg, registry.Opt{
			Insecure: instance.Spec.Insecure,
			Debug:    verbose,
			SkipPing: false,
			Timeout:  timeout,
		}, tlsCertData)
		if err != nil {
			return err
		}

		job := scanctl.NewScanJob(r, scanner, e.Images, instance.Spec.MaxFixable, instance.Spec.SendReport)
		jobs = append(jobs, job)
	}

	// TODO: Load config value from operator config
	es := scanctl.NewReportClient(config.Config.GetString(config.ConfigImageReportSvr),
		&http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	)

	task := scanctl.NewScanTask(jobs,
		func(st *scanctl.ScanTask) {
			_ = r.updateStatus(instance, tmaxiov1.ScanRequestProcessing, "", nil)
		}, func(st *scanctl.ScanTask) {

			result := map[string]tmaxiov1.ScanResult{}

			for _, job := range st.Jobs() {
				for imageName, r := range job.Result() {

					scanResult := convertReport(r, job.MaxVuls())

					if job.SendReportEnabled {
						esReport := tmaxiov1.ImageScanRequestESReport{
							Image:  imageName,
							Result: scanResult,
						}

						err := es.SendReport(instance.Namespace, &esReport)
						if err != nil {
							log.Error(err, "Failed to send report.")
						}
					}

					// Do not update detail on object
					scanResult.Vulnerabilities = nil
					result[imageName] = scanResult
				}
			}

			_ = r.updateStatus(instance, tmaxiov1.ScanRequestSuccess, "success", result)

		}, func(err error) {
			_ = r.updateStatus(instance, tmaxiov1.ScanRequestFail, err.Error(), nil)
		})

	worker.Submit(task)
	_ = r.updateStatus(instance, tmaxiov1.ScanRequestPending, "", nil)

	return nil
}

func (r *ImageScanRequestReconciler) updateStatus(instance *tmaxiov1.ImageScanRequest, status tmaxiov1.ScanRequestStatusType, msg string, results map[string]tmaxiov1.ScanResult) error {
	original := instance.DeepCopy()

	instance.Status.Status = status
	if len(msg) > 0 {
		instance.Status.Message = msg
	}
	if results != nil {
		instance.Status.Results = results
	}

	return r.Client.Status().Patch(context.TODO(), instance, client.MergeFrom(original))
}

func (r *ImageScanRequestReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&tmaxiov1.ImageScanRequest{}).
		Complete(r)
}

func convertReport(reports *clair.VulnerabilityReport, threshold int) (ret tmaxiov1.ScanResult) {

	SeverityNames := []string{"Unknown", "Negligible", "Low", "Medium", "High", "Critical", "Defcon1"}

	summary := map[string]int{}
	fatal := []string{}
	vuls := map[string]tmaxiov1.Vulnerabilities{}

	// FIXME: Load maxBadVuls value from manager.config
	maxBadVuls := 10
	nBadVuls := 0

	for _, n := range SeverityNames {
		summary[n] = 0
	}

	for severity, vulnerabilityList := range reports.VulnsBySeverity {
		//
		summary[severity] = len(vulnerabilityList)

		//
		vul := tmaxiov1.Vulnerabilities{}
		for _, v := range vulnerabilityList {

			vul = append(vul, tmaxiov1.Vulnerability{
				Name:          v.Name,
				NamespaceName: v.NamespaceName,
				Description:   v.Description,
				Link:          v.Link,
				Severity:      v.Severity,
				//Metadata:      obj,
				FixedBy: v.FixedBy,
			})
		}
		vuls[severity] = vul

		// Count the number of bad vulnerability
		if severity == "High" || severity == "Critical" || severity == "Defcon1" {
			nBadVuls++
		}
	}

	if fixable, ok := reports.VulnsBySeverity["Fixable"]; ok && len(fixable) > threshold {
		fatal = append(fatal, fmt.Sprintf("%d fixable vulnerabilities found", len(fixable)))
	}

	if nBadVuls > maxBadVuls {
		fatal = append(fatal, fmt.Sprintf("%d bad vulnerabilities found", nBadVuls))
	}

	return tmaxiov1.ScanResult{
		Summary:         summary,
		Fatal:           fatal,
		Vulnerabilities: vuls,
	}
}
