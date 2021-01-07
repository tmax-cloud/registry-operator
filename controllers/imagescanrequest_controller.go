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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/genuinetools/reg/clair"
	reg "github.com/genuinetools/reg/clair"
	"github.com/genuinetools/reg/registry"
	"github.com/genuinetools/reg/repoutils"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	tmaxiov1 "github.com/tmax-cloud/registry-operator/api/v1"
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
	if len(instance.Status.Status) != 0 {
		reqLogger.Info("already handled scannning")
		return ctrl.Result{}, nil
	}
	//get vulnerability
	report, err := GetVulnerability(instance)

	//update status
	return r.updateScanningStatus(instance, &report, err)
}

func (r *ImageScanRequestReconciler) updateScanningStatus(instance *tmaxiov1.ImageScanRequest, report *reg.VulnerabilityReport, err error) (ctrl.Result, error) {
	reqLogger := r.Log.WithName("update Scanning status")
	// set condition depending on the error
	instanceWithStatus := instance.DeepCopy()

	var cond tmaxiov1.ImageScanRequestStatus
	if err == nil {
		cond.Message = "succeed to get vulnerability"
		cond.Status = "Success"
		cond.Summary, cond.Fatal, cond.Vulnerabilities = ParseAnalysis(instance.Spec.FixableThreshold, report)
	} else {
		cond.Message = err.Error()
		cond.Reason = "error occurs while analyze vulnerability"
		cond.Status = "Error"
	}

	// send logging server
	webhookUrl := os.Getenv("WEBHOOK_URL")
	if err == nil && len(webhookUrl) != 0 && instance.Spec.Webhook {
		data, err := json.Marshal(cond)
		if err != nil {
			reqLogger.Error(err, "fail marshal request")
		}
		requestUrl := webhookUrl + "/image-scanning-" + instance.Namespace + "/_doc/" + instance.Name
		res, err := http.Post(requestUrl, "application/json", bytes.NewReader(data))
		if err != nil {
			reqLogger.Error(err, "cannot send webhook server")
		} else {
			bodyBytes, _ := ioutil.ReadAll(res.Body)
			reqLogger.Info("webhook: " + string(bodyBytes))
			defer res.Body.Close()
		}
	}

	// set status
	instanceWithStatus.Status = cond

	if errUp := r.Client.Status().Patch(context.TODO(), instanceWithStatus, client.MergeFrom(instance)); errUp != nil {
		reqLogger.Error(errUp, "could not update scanning")
		return ctrl.Result{}, errUp
	}

	reqLogger.Info("succeed to update scanning status")
	return ctrl.Result{}, err
}

func ParseAnalysis(threshold int, report *reg.VulnerabilityReport) (map[string]int, []string, map[string]tmaxiov1.Vulnerabilities) {
	vulnerabilities := make(map[string]tmaxiov1.Vulnerabilities)
	summary := make(map[string]int)
	var fatal []string

	//set vulnerabilites
	for sev, vulns := range report.VulnsBySeverity {
		var vuls []tmaxiov1.Vulnerability
		for _, v := range vulns {
			obj := runtime.RawExtension{}
			meta, _ := json.Marshal(v.Metadata)
			obj.Raw = meta
			vul := tmaxiov1.Vulnerability{
				Name:          v.Name,
				NamespaceName: v.NamespaceName,
				Description:   v.Description,
				Link:          v.Link,
				Severity:      v.Severity,
				Metadata:      obj,
				FixedBy:       v.FixedBy,
			}
			vuls = append(vuls, vul)
		}
		vulnerabilities[sev] = vuls
	}

	if len(report.VulnsBySeverity) < 1 {
		return summary, fatal, vulnerabilities
	}

	//set summary
	for sev, vulns := range report.VulnsBySeverity {
		summary[sev] = len(vulns)
	}

	//set fatal
	fixable, ok := report.VulnsBySeverity["Fixable"]
	if ok {
		if len(fixable) > threshold {
			fatal = append(fatal, fmt.Sprintf("%d fixable vulnerabilities found", len(fixable)))
		}
	}

	// Return an error if there are more than 10 bad vulns.
	badVulns := 0
	// Include any high vulns.
	if highVulns, ok := report.VulnsBySeverity["High"]; ok {
		badVulns += len(highVulns)
	}
	// Include any critical vulns.
	if criticalVulns, ok := report.VulnsBySeverity["Critical"]; ok {
		badVulns += len(criticalVulns)
	}
	// Include any defcon1 vulns.
	if defcon1Vulns, ok := report.VulnsBySeverity["Defcon1"]; ok {
		badVulns += len(defcon1Vulns)
	}
	if badVulns > 10 {
		fatal = append(fatal, fmt.Sprintf("%d bad vulnerabilities found", len(fixable)))
	}
	return summary, fatal, vulnerabilities
}

func InitParameter(instance *tmaxiov1.ImageScanRequest) {
	if instance.Spec.TimeOut == 0 {
		instance.Spec.TimeOut = time.Minute
	}
}

func GetVulnerability(instance *tmaxiov1.ImageScanRequest) (reg.VulnerabilityReport, error) {

	InitParameter(instance)
	report := reg.VulnerabilityReport{}

	//get clair url
	clairServer := os.Getenv("CLAIR_URL")
	if len(clairServer) == 0 {
		return report, errors.NewBadRequest("cannot find clairUrl")
	}

	if instance.Spec.FixableThreshold < 0 {
		return report, errors.NewBadRequest("fixable threshold must be a positive integer")
	}
	image, err := registry.ParseImage(instance.Spec.ImageUrl)
	if err != nil {
		return report, err
	}

	// Create the registry client.
	r, err := createRegistryClient(instance, image.Domain)
	if err != nil {
		return report, err
	}

	// Initialize clair client.
	cr, err := clair.New(clairServer, clair.Opt{
		Debug:    instance.Spec.Debug,
		Timeout:  instance.Spec.TimeOut,
		Insecure: instance.Spec.Insecure,
	})
	if err != nil {
		return report, err
	}

	// Get the vulnerability report.
	if report, err = cr.VulnerabilitiesV3(context.TODO(), r, image.Path, image.Reference()); err != nil {
		// Fallback to Clair v2 API.
		if report, err = cr.Vulnerabilities(context.TODO(), r, image.Path, image.Reference()); err != nil {
			return report, err
		}
	}

	return report, err
}

func createRegistryClient(instance *tmaxiov1.ImageScanRequest, domain string) (*registry.Registry, error) {
	// Use the auth-url domain if provided.
	authDomain := instance.Spec.AuthUrl
	if authDomain == "" {
		authDomain = domain
	}
	auth, err := repoutils.GetAuthConfig(instance.Spec.Username, instance.Spec.Password, authDomain)
	if err != nil {
		return nil, err
	}

	// Prevent non-ssl unless explicitly forced
	if !instance.Spec.ForceNonSSL && strings.HasPrefix(auth.ServerAddress, "http:") {
		return nil, fmt.Errorf("attempted to use insecure protocol! Use force-non-ssl option to force")
	}

	// Create the registry client.
	return registry.New(context.TODO(), auth, registry.Opt{
		Domain:   domain,
		Insecure: instance.Spec.Insecure,
		Debug:    instance.Spec.Debug,
		SkipPing: instance.Spec.SkipPing,
		NonSSL:   instance.Spec.ForceNonSSL,
		Timeout:  instance.Spec.TimeOut,
	})
}

func (r *ImageScanRequestReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&tmaxiov1.ImageScanRequest{}).
		Complete(r)
}
