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
	"path"
	"regexp"
	"strings"
	"sync"
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
	scanner  *clair.Clair
	reporter *scanctl.ReportClient
	verbose  = false
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

	reporter = scanctl.NewReportClient(config.Config.GetString(config.ConfigImageReportSvr),
		&http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	)
}

// +kubebuilder:rbac:groups=tmax.io,resources=imagescanrequests,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=tmax.io,resources=imagescanrequests/status,verbs=get;update;patch

func (r *ImageScanRequestReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	logger := r.Log.WithValues("namespace", req.Namespace, "name", req.Name)

	instance := &tmaxiov1.ImageScanRequest{}
	err := r.Client.Get(ctx, req.NamespacedName, instance)
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
		if err = r.mutate(instance); err != nil {
			instance.Status.Status = tmaxiov1.ScanRequestError
			instance.Status.Message = err.Error()
			r.Status().Update(ctx, instance)
			return ctrl.Result{}, err
		}
		if err = validate(instance); err != nil {
			instance.Status.Status = tmaxiov1.ScanRequestError
			instance.Status.Message = err.Error()
			r.Status().Update(ctx, instance)
			return ctrl.Result{}, err
		}
		r.Update(ctx, instance)
		instance.Status.Status = tmaxiov1.ScanRequestPending
		r.Status().Update(ctx, instance)
	case tmaxiov1.ScanRequestPending:
		if err = r.doRecept(instance); err != nil {
			instance.Status.Status = tmaxiov1.ScanRequestError
			instance.Status.Message = err.Error()
			r.Status().Update(ctx, instance)
			return ctrl.Result{}, err
		}
		instance.Status.Status = tmaxiov1.ScanRequestProcessing
		r.Status().Update(ctx, instance)
	case tmaxiov1.ScanRequestProcessing:
		logger.Info("already in procssing...")
	case tmaxiov1.ScanRequestSuccess:
		logger.Info("finished job...")
		if !isImageListSameBetweenSpecAndStatus(instance) {
			instance.Status.Status = ""
			instance.Status.Message = ""
			r.Status().Update(ctx, instance)
		}
	case tmaxiov1.ScanRequestFail:
		logger.Info("failed job...")
	}
	return ctrl.Result{}, nil
}

func (r *ImageScanRequestReconciler) getRegistry(ctx context.Context, o *tmaxiov1.ImageScanRequest,
	t tmaxiov1.ScanTarget) (*registry.Registry, error) {
	// FIXME: Is here the right place to get keyclock cert?
	operatorRootCA := types.NamespacedName{
		Name:      tmaxiov1.RegistryRootCASecretName,
		Namespace: config.Config.GetString("operator.namespace"),
	}
	tlsSecret := &corev1.Secret{}
	if err := r.Get(ctx, operatorRootCA, tlsSecret); err != nil {
		return nil, fmt.Errorf("TLS Secret not found: %s\n", tmaxiov1.RegistryRootCASecretName)
	}

	tlsCertData, err := secrethelper.GetCert(tlsSecret, certs.RootCACert)
	if err != nil {
		return nil, err
	}
	if len(t.CertificateSecret) > 0 {
		tlsSecret := &corev1.Secret{}
		if err := r.Get(ctx, types.NamespacedName{Name: t.CertificateSecret, Namespace: o.Namespace}, tlsSecret); err != nil {
			return nil, fmt.Errorf("TLS Secret not found: %s\n", t.CertificateSecret)
		}
		privateCertData, err := secrethelper.GetCert(tlsSecret, certs.RootCACert)
		if err != nil {
			return nil, err
		}
		tlsCertData = append(tlsCertData, []byte("\n")...)
		tlsCertData = append(tlsCertData, privateCertData...)
	}
	username := ""
	password := ""
	if strings.HasPrefix(t.RegistryURL, "http://") || strings.HasPrefix(t.RegistryURL, "https://") {
		return nil, fmt.Errorf("registry url must not have protocol(http, https).")
	}
	_, err = url.ParseRequestURI("https://" + t.RegistryURL)
	if err != nil {
		return nil, err
	}
	if len(t.ImagePullSecret) > 0 {
		secret := &corev1.Secret{}
		if err := r.Client.Get(ctx, types.NamespacedName{Name: t.ImagePullSecret, Namespace: o.Namespace}, secret); err != nil {
			return nil, fmt.Errorf("ImagePullSecret not found: %s\n", t.ImagePullSecret)
		}
		imagePullSecret, err := secrethelper.NewImagePullSecret(secret)
		if err != nil {
			return nil, err
		}
		login, err := imagePullSecret.GetHostCredential(t.RegistryURL)
		if err != nil {
			return nil, err
		}
		username = login.Username
		password = strings.TrimSpace(string(login.Password))
	}
	targetURL := t.RegistryURL
	if targetURL == "docker.io" {
		targetURL = "https://registry-1.docker.io"
	}
	authCfg, err := repoutils.GetAuthConfig(username, password, targetURL)
	if err != nil {
		return nil, err
	}
	reg, err := secureReg.New(ctx, authCfg, registry.Opt{
		Insecure: o.Spec.Insecure,
		Debug:    verbose,
		SkipPing: false,
		//Debug:    false,
		//SkipPing: true,
		Timeout: timeout,
	}, tlsCertData)
	if err != nil {
		return nil, err
	}
	return reg, nil
}

func (r *ImageScanRequestReconciler) doRecept(o *tmaxiov1.ImageScanRequest) error {
	ctx, cancel := context.WithCancel(context.TODO())
	logger := r.Log.WithValues("namespace", o.Namespace, "name", o.Name)

	var wgErr error
	wg := new(sync.WaitGroup)

	for _, st := range o.Spec.ScanTargets {
		wg.Add(1)
		go func(e tmaxiov1.ScanTarget) {
			defer wg.Done()
			reg, err := r.getRegistry(ctx, o, e)
			if err != nil {
				wgErr = err
				cancel()
				return
			}
			logger.Info("registry ok...")

			//var resolveds []string
			//for _, image := range e.Images {
			//	resolvedPaths, err := resolveImagePath(ctx, reg, image)
			//	if err != nil {
			//		wgErr = err
			//		cancel()
			//		return
			//	}
			//	resolveds = append(resolveds, resolvedPaths...)
			//}
			//logger.Info("resolved path: " + strings.Join(resolveds, ","))

			for _, imagePath := range st.Images {
				imagePath = path.Join(reg.Domain, imagePath)
				img, err := registry.ParseImage(imagePath)
				if err != nil {
					wgErr = err
					cancel()
					return
				}
				logger.Info("start scan: " + reg.Domain + "/" + img.Path + ":" + img.Tag)
				vul, err := scanner.Vulnerabilities(ctx, reg, img.Path, img.Tag)
				if err != nil {
					wgErr = err
					cancel()
					return
				}
				logger.Info("scanning complete...")

				scanResult := convertReport(&vul, o.Spec.MaxFixable)
				if o.Spec.SendReport {
					logger.Info("start report...")
					esReport := tmaxiov1.ImageScanRequestESReport{
						Image:  imagePath,
						Result: scanResult,
					}
					err := reporter.SendReport(o.Namespace, &esReport)
					if err != nil {
						wgErr = err
						cancel()
						return
					}
					logger.Info("report complete...")
				}
				// Do not update detail on cr
				scanResult.Vulnerabilities = nil
				if o.Status.Results == nil {
					o.Status.Results = map[string]tmaxiov1.ScanResult{}
				}
				o.Status.Results[imagePath] = scanResult
			}
		}(st)
	}

	go func() {
		wg.Wait()
		if wgErr == nil {
			o.Status.Status = tmaxiov1.ScanRequestSuccess
		} else {
			o.Status.Status = tmaxiov1.ScanRequestFail
			o.Status.Message = wgErr.Error()
		}
		r.Status().Update(context.TODO(), o)
	}()

	return nil
}

func (r *ImageScanRequestReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&tmaxiov1.ImageScanRequest{}).
		Complete(r)
}

func resolveImagePath(ctx context.Context, reg *registry.Registry, image string) ([]string, error) {
	resolved := []string{}

	var repo, tag string
	isTagExist := false
	if strings.Contains(image, ":") {
		isTagExist = true
		t := strings.Split(image, ":")
		repo = t[0]
		tag = t[1]
	} else {
		repo = image
	}

	matchingRepos := []string{}
	if strings.ContainsAny("*?", repo) {
		// FIXME: Not possible in the case of docker.io
		catalog, err := reg.Catalog(ctx, "")
		if err != nil {
			return nil, err
		}

		for _, c := range catalog {
			if isMatch, _ := regexp.MatchString(convertToRegexp(repo), c); isMatch {
				matchingRepos = append(matchingRepos, c)
			}
		}
	} else {
		matchingRepos = append(matchingRepos, repo)
	}

	if isTagExist {
		for _, r := range matchingRepos {
			resolved = append(resolved, strings.Join([]string{r, tag}, ":"))
		}
	} else {
		for _, r := range matchingRepos {
			tags, err := reg.Tags(ctx, r)
			if err != nil {
				return nil, err
			}
			for _, t := range tags {
				resolved = append(resolved, strings.Join([]string{r, t}, ":"))
			}
		}
	}

	return resolved, nil
}

func convertToRegexp(s string) string {
	c1 := strings.ReplaceAll(s, "?", ".")
	c2 := strings.ReplaceAll(c1, "*", "[[:alnum:]]")
	return c2
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
		summary[severity] = len(vulnerabilityList)
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

func validate(instance *tmaxiov1.ImageScanRequest) error {
	if instance.Spec.MaxFixable < 0 {
		return fmt.Errorf(".Spec.MaxFixable cannot be negative")
	}

	for _, target := range instance.Spec.ScanTargets {
		for _, image := range target.Images {
			imagepath := path.Join(target.RegistryURL, image)
			_, err := registry.ParseImage(imagepath)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *ImageScanRequestReconciler) mutate(o *tmaxiov1.ImageScanRequest) error {
	ctx := context.TODO()
	logger := r.Log.WithValues("namespace", o.Namespace, "name", o.Name)

	for idx, st := range o.Spec.ScanTargets {
		reg, err := r.getRegistry(ctx, o, st)
		if err != nil {
			return err
		}

		var resolveds []string
		for _, image := range st.Images {
			resolvedPaths, err := resolveImagePath(ctx, reg, image)
			if err != nil {
				logger.Error(err, "failed to resolve image"+image)
				return err
			}
			resolveds = append(resolveds, resolvedPaths...)
		}
		o.Spec.ScanTargets[idx].Images = resolveds
	}

	return nil
}

func isImageListSameBetweenSpecAndStatus(instance *tmaxiov1.ImageScanRequest) bool {
	targetImagePaths := []string{}
	for _, target := range instance.Spec.ScanTargets {
		registryURL := target.RegistryURL
		if target.RegistryURL == "docker.io" {
			registryURL = "registry-1.docker.io"
		}
		for _, imgName := range target.Images {
			//  in case of changing manually
			if strings.ContainsAny("*?", imgName) {
				return false
			}
			targetImagePaths = append(targetImagePaths, path.Join(registryURL, imgName))
		}
	}
	if len(targetImagePaths) != len(instance.Status.Results) {
		return false
	}
	for _, imagePath := range targetImagePaths {
		if _, ok := instance.Status.Results[imagePath]; !ok {
			return false
		}
	}
	return true
}
