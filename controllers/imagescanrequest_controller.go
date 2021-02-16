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
	"fmt"
	"net/url"

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
)

func init() {
	worker = scanctl.NewScanWorker(1, 1)
	worker.Start()
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
		if err := r.validate(instance, req.Namespace); err != nil {
			return ctrl.Result{}, err
		}
		r.doRecept(instance, req.Namespace)

		if err := r.updateStatus(instance, tmaxiov1.ScanRequestRecepted); err != nil {
			return ctrl.Result{}, err
		}
	case tmaxiov1.ScanRequestRecepted:
		logger.Info("Already recepted request...")
		return ctrl.Result{}, nil
	case tmaxiov1.ScanRequestProcessing:
		logger.Info("Already in procssing...")
		return ctrl.Result{}, nil
	}

	return ctrl.Result{}, nil
}

<<<<<<< HEAD
func (r *ImageScanRequestReconciler) updateScanningStatus(instance *tmaxiov1.ImageScanRequest, reports map[string]map[string]*reg.VulnerabilityReport, err error) (ctrl.Result, error) {
	reqLogger := r.Log.WithName("update Scanning status")
	// set condition depending on the error
	instanceWithStatus := instance.DeepCopy()
	var status tmaxiov1.ImageScanRequestStatus

	if err == nil {
		//start processing
		if len(instance.Status.Status) == 0 {
			status.Message = "Scanning in process"
			status.Status = tmaxiov1.ScanRequestProcessing

		} else if instance.Status.Status == tmaxiov1.ScanRequestProcessing {
			status.Message = "succeed to get vulnerability"
			status.Status = tmaxiov1.ScanRequestSuccess
			status.Results = map[string]tmaxiov1.ScanResult{}

			esURL := config.Config.GetString(config.ConfigElasticSearchURL)
			for registry, imageReports := range reports {
				for image, report := range imageReports {
					for _, target := range instance.Spec.ScanTargets {
						if target.RegistryURL != registry {
							continue
						}

						esReport := tmaxiov1.ImageScanRequestESReport{Image: fmt.Sprintf("%s/%s", registry, image)}
						reqLogger.Info("new elasticsearch report", "image", fmt.Sprintf("%s/%s", registry, image))
						// set scan result
						esReport.Result.Summary, esReport.Result.Fatal, esReport.Result.Vulnerabilities = scanctl.ParseAnalysis(target.FixableThreshold, report)
						status.Results[path.Join(registry, image)] = tmaxiov1.ScanResult{Summary: esReport.Result.Summary}

						// send logging server
						if err == nil && len(esURL) != 0 {
							if target.RegistryURL == registry && target.ElasticSearch {
								res, err := scanctl.SendElasticSearchServer(esURL, instance.Namespace, instance.Name, &esReport)
								if err != nil {
									reqLogger.Error(err, "failed to send ES Server")
								}
								if err == nil {
									bodyBytes, _ := ioutil.ReadAll(res.Body)
									reqLogger.Info("webhook: " + string(bodyBytes))
								}
							}
						}

					}
				}
			}
		}
	} else {
		status.Message = err.Error()
		status.Reason = "error occurs while analyze vulnerability"
		status.Status = "Error"
=======
// TODO: Move this to validatation webhook
func (r *ImageScanRequestReconciler) validate(instance *tmaxiov1.ImageScanRequest, namespace string) error {

	for _, e := range instance.Spec.ScanTargets {
		if len(e.RegistryURL) < 1 {
			return errors.NewBadRequest("Empty registry URL")
		}

		// if !e.ForceNonSSL && strings.HasPrefix(config.ServerAddress, "http:") {
		// 	return errors.NewBadRequest("attempted to use insecure protocol! Use force-non-ssl option to force")
		// }

		if e.FixableThreshold < 0 {
			return errors.NewBadRequest("fixable threshold must be a positive integer")
		}

		if len(e.ImagePullSecret) < 1 {
			return errors.NewBadRequest("Empty ImagePullSecret")
		}
		// FIXME: insecure 허용 시 예외처리
		if len(e.CertificateSecret) < 1 {
			return errors.NewBadRequest("Empty TLS Secret")
		}

		ctx := context.TODO()
		imagePullSecret := &corev1.Secret{}
		if err := r.Client.Get(ctx, types.NamespacedName{Name: e.ImagePullSecret, Namespace: namespace}, imagePullSecret); err != nil {
			return errors.NewBadRequest(fmt.Sprintf("ImagePullSecret not found: %s\n", e.ImagePullSecret))
		}

		// FIXME: insecure 허용 시 예외처리
		tlsSecret := &corev1.Secret{}
		if err := r.Client.Get(ctx, types.NamespacedName{Name: e.CertificateSecret, Namespace: namespace}, tlsSecret); err != nil {
			return errors.NewBadRequest(fmt.Sprintf("TLS Secret not found: %s\n", e.CertificateSecret))
		}
>>>>>>> [refactor] Refactor ImageScanRequestController and enable to build
	}

	return nil
}

func (r *ImageScanRequestReconciler) doRecept(instance *tmaxiov1.ImageScanRequest, namespace string) error {

	jobs := []*scanctl.ScanJob{}
	for _, e := range instance.Spec.ScanTargets {
		ctx := context.TODO()

		secret := &corev1.Secret{}
		if err := r.Client.Get(ctx, types.NamespacedName{Name: e.ImagePullSecret, Namespace: namespace}, secret); err != nil {
			return fmt.Errorf("ImagePullSecret not found: %s\n", e.ImagePullSecret)
		}

		imagePullSecret, err := secrethelper.NewImagePullSecret(secret)
		if err != nil {
			return err
		}

		registryURL, err := url.Parse(e.RegistryURL)
		if err != nil {
			return err
		}
		registryHost := registryURL.Hostname()

		login, err := imagePullSecret.GetHostCredential(registryHost)
		if err != nil {
			return err
		}
		username := login.Username
		password := string(login.Password)

		// FIXME: insecure 허용 시 예외처리
		tlsSecret := &corev1.Secret{}
		if err := r.Client.Get(ctx, types.NamespacedName{Name: e.CertificateSecret, Namespace: namespace}, tlsSecret); err != nil {
			return fmt.Errorf("TLS Secret not found: %s\n", e.CertificateSecret)
		}

		tlsCertData, err := secrethelper.GetCert(tlsSecret, "")
		if err != nil {
			return err
		}

		// if err := r.Client.Get(ctx, types.NamespacedName{Name: tmaxiov1.KeycloakCASecretName, Namespace: config.Config.GetString("operator.namespace")}, tlsSecret); err != nil {
		// 	return fmt.Errorf("TLS Secret not found: %s\n", tmaxiov1.KeycloakCASecretName)
		// }

		// keyclockCertData, err := secrethelper.GetCert(tlsSecret, certs.RootCACert)
		// if err != nil {
		// 	return err
		// }

		// tlsCertData = append(tlsCertData, keyclockCertData...)

		// get auth config
		authCfg, err := repoutils.GetAuthConfig(username, password, e.RegistryURL)
		if err != nil {
			return err
		}

		// Create the registry client.
		r, err := secureReg.New(context.TODO(), authCfg, registry.Opt{
			Insecure: e.Insecure,
			Debug:    e.Debug,
			SkipPing: e.SkipPing,
			NonSSL:   e.ForceNonSSL,
			Timeout:  e.TimeOut,
		}, tlsCertData)

		// FIXME: replace url
		clairAddress := config.Config.GetString("clair.url")
		cr, err := clair.New(clairAddress, clair.Opt{
			Debug:    e.Debug,
			Timeout:  e.TimeOut,
			Insecure: e.Insecure,
		})
		if err != nil {
			return err
		}

		job := scanctl.NewScanJob(r, cr, e.Images)
		jobs = append(jobs, job)
	}

	worker.Submit(jobs)
	return nil
}

func (r *ImageScanRequestReconciler) updateStatus(instance *tmaxiov1.ImageScanRequest, status tmaxiov1.ScanRequestStatusType) error {
	original := instance.DeepCopy()
	instance.Status.Status = status
	return r.Client.Status().Patch(context.TODO(), instance, client.MergeFrom(original))
}

func (r *ImageScanRequestReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&tmaxiov1.ImageScanRequest{}).
		Complete(r)
}
