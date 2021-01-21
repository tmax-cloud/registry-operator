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
	"io/ioutil"
	"path"

	reg "github.com/genuinetools/reg/clair"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	tmaxiov1 "github.com/tmax-cloud/registry-operator/api/v1"
	"github.com/tmax-cloud/registry-operator/controllers/scanctl"
	"github.com/tmax-cloud/registry-operator/internal/common/config"
)

// ImageScanRequestReconciler reconciles a ImageScanRequest object
type ImageScanRequestReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=tmax.io,resources=imagescanrequests,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=tmax.io,resources=imagescanrequests/status,verbs=get;update;patch

func (r *ImageScanRequestReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	reqLogger := r.Log.WithValues("Request.Namespace", req.Namespace, "Request.Name", req.Name)
	reqLogger.Info("Reconciling Scanning")

	// your logic here
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
	if len(instance.Status.Status) == 0 {
		return r.updateScanningStatus(instance, nil, nil)
	} else if instance.Status.Status != tmaxiov1.ScanRequestProcessing {
		reqLogger.Info("already handled scannning")
		return ctrl.Result{}, nil
	}
	//get vulnerability
	reports, err := scanctl.GetVulnerability(r.Client, instance)
	if err != nil {
		reqLogger.Error(err, "failed to get vulnerability")
	}

	//update status
	return r.updateScanningStatus(instance, reports, err)
}

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

			esURL := config.Config.GetString("elastic_search.url")
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
	}

	// set status
	instanceWithStatus.Status = status

	if errUp := r.Client.Status().Patch(context.TODO(), instanceWithStatus, client.MergeFrom(instance)); errUp != nil {
		reqLogger.Error(errUp, "could not update scanning")
		return ctrl.Result{}, errUp
	}

	reqLogger.Info("succeed to update scanning status")
	return ctrl.Result{}, err
}

func (r *ImageScanRequestReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&tmaxiov1.ImageScanRequest{}).
		Complete(r)
}
