package scanctl

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"github.com/genuinetools/reg/clair"
	reg "github.com/genuinetools/reg/clair"
	"github.com/genuinetools/reg/registry"
	"github.com/genuinetools/reg/repoutils"
	tmaxiov1 "github.com/tmax-cloud/registry-operator/api/v1"
	"github.com/tmax-cloud/registry-operator/internal/common/certs"
	"github.com/tmax-cloud/registry-operator/internal/utils"
	regApi "github.com/tmax-cloud/registry-operator/pkg/registry"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var logger = logf.Log.WithName("registry_scan")

var (
	// Priorities are the vulnerability priority labels.
	Priorities = []string{"Unknown", "Negligible", "Low", "Medium", "High", "Critical", "Defcon1"}
)

func ParseAnalysis(threshold int, report *reg.VulnerabilityReport) map[string]int {
	summary := make(map[string]int)

	for _, val := range Priorities {
		summary[val] = 0
	}

	if len(report.VulnsBySeverity) < 1 {
		return summary
	}

	//set summary
	for sev, vulns := range report.VulnsBySeverity {
		summary[sev] = len(vulns)
	}

	return summary
}

func InitParameter(target *tmaxiov1.ScanTarget) {
	if target.TimeOut == 0 {
		target.TimeOut = time.Minute
	}
}

func imageUrl(registryUrl, image string) string {
	return path.Join(registryUrl, image)
}

func GetRegistryImages(c client.Client, registryURL, basicAuth, imageNamePattern string) []string {
	images := []string{}

	caSecret, err := certs.GetSystemRootCASecret(c)
	if err != nil {
		logger.Error(err, "failed to get system root ca secret")
		return images
	}
	ca := caSecret.Data[certs.RootCACert]

	regCtl := regApi.NewRegistryAPI(registryURL, basicAuth, ca)
	repos := regCtl.Catalog()
	if repos == nil {
		return images
	}
	for _, repo := range repos.Repositories {
		vers := regCtl.Tags(repo)
		if vers == nil {
			continue
		}
		for _, ver := range vers.Tags {
			image := repo + ":" + ver
			if utils.Matched(imageNamePattern, image) {
				images = append(images, image)
			}
		}
	}

	return images
}

func getBasicAuth(imagePullSecret, namespace, registryURL string) (string, error) {
	secret, err := getSecret(imagePullSecret, namespace)
	if err != nil {
		logger.Error(err, "failed to get image pull secret")
		return "", err
	}

	basic, err := utils.ParseBasicAuth(secret, registryURL)
	if err != nil {
		logger.Error(err, "failed to parse basic auth")
		return "", err
	}

	return basic, nil
}

func GetVulnerability(c client.Client, instance *tmaxiov1.ImageScanRequest) (map[string]map[string]*reg.VulnerabilityReport, error) {
	reports := map[string]map[string]*reg.VulnerabilityReport{}

	//get clair url
	clairServer := os.Getenv("CLAIR_URL")
	if len(clairServer) == 0 {
		return reports, errors.NewBadRequest("cannot find clairUrl")
	}

	for i, target := range instance.Spec.ScanTargets {
		InitParameter(&instance.Spec.ScanTargets[i])
		if target.FixableThreshold < 0 {
			return reports, errors.NewBadRequest("fixable threshold must be a positive integer")
		}

		for _, targetImage := range target.Images {
			matchImages := []string{}
			if strings.Contains(targetImage, "*") || strings.Contains(targetImage, "?") {
				var basicAuth string
				if target.ImagePullSecret != "" {
					basic, err := getBasicAuth(target.ImagePullSecret, instance.Namespace, target.RegistryURL)
					if err != nil {
						logger.Error(err, fmt.Sprintf("failed to get basic auth from imagepullsecret: %s", target.ImagePullSecret))
						return reports, err
					}
					basicAuth = basic
				}
				matchImages = append(matchImages, GetRegistryImages(c, target.RegistryURL, basicAuth, targetImage)...)
			} else {
				matchImages = append(matchImages, targetImage)
			}

			for _, imgName := range matchImages {
				imgUrl := imageUrl(target.RegistryURL, imgName)
				logger.Info(fmt.Sprintf("scan image: %s", imgUrl))
				image, err := registry.ParseImage(imgUrl)
				if err != nil {
					logger.Error(err, "failed to parse image")
					return reports, err
				}

				// Create the registry client.
				r, err := createRegistryClient(&target, image.Domain, instance.Namespace)
				if err != nil {
					logger.Error(err, "failed to create registry client")
					return reports, err
				}

				// Initialize clair client.
				cr, err := clair.New(clairServer, clair.Opt{
					Debug:    target.Debug,
					Timeout:  target.TimeOut,
					Insecure: target.Insecure,
				})
				if err != nil {
					logger.Error(err, "failed to new clair client")
					return reports, err
				}

				report := reg.VulnerabilityReport{}

				// Get the vulnerability report.
				if report, err = cr.Vulnerabilities(context.TODO(), r, image.Path, image.Reference()); err != nil {
					// Fallback to Clair v3 API.
					if report, err = cr.VulnerabilitiesV3(context.TODO(), r, image.Path, image.Reference()); err != nil {
						logger.Error(err, "failed to check vulnerabilities")
						return reports, err
					}
				}

				// set report
				if m, ok := reports[target.RegistryURL]; ok {
					m[imgName] = &report
					reports[target.RegistryURL] = m
				} else {
					reports[target.RegistryURL] = map[string]*reg.VulnerabilityReport{imgName: &report}
				}

			}
		}
	}

	return reports, nil
}

func getSecret(name, namespace string) (*corev1.Secret, error) {
	c, err := client.New(config.GetConfigOrDie(), client.Options{})
	if err != nil {
		return nil, err
	}
	secret := &corev1.Secret{}
	if err := c.Get(context.TODO(), types.NamespacedName{Name: name, Namespace: namespace}, secret); err != nil {
		return nil, err
	}

	return secret, nil
}

func createRegistryClient(target *tmaxiov1.ScanTarget, domain, namespace string) (*registry.Registry, error) {
	// Use the auth-url domain if provided.
	authDomain := target.AuthURL
	if authDomain == "" {
		authDomain = domain
	}

	username, password := "", ""

	if len(target.ImagePullSecret) > 0 {
		secret, err := getSecret(target.ImagePullSecret, namespace)
		if err != nil {
			logger.Error(err, "failed to get image pull secret")
			return nil, err
		}

		basic, err := utils.ParseBasicAuth(secret, target.RegistryURL)
		if err != nil {
			logger.Error(err, "failed to parse basic auth")
			return nil, err
		}

		dec, err := base64.StdEncoding.DecodeString(basic)
		if err != nil {
			logger.Error(err, "failed to decode string by base64")
			return nil, err
		}

		basic = string(dec)
		sepIdx := strings.Index(basic, ":")
		username = basic[:sepIdx]
		password = basic[sepIdx+1:]
	}

	auth, err := repoutils.GetAuthConfig(username, password, authDomain)
	if err != nil {
		logger.Error(err, "failed to get auth config")
		return nil, err
	}

	// Prevent non-ssl unless explicitly forced
	if !target.ForceNonSSL && strings.HasPrefix(auth.ServerAddress, "http:") {
		return nil, fmt.Errorf("attempted to use insecure protocol! Use force-non-ssl option to force")
	}

	// Create the registry client.
	return registry.New(context.TODO(), auth, registry.Opt{
		Domain:   domain,
		Insecure: target.Insecure,
		Debug:    target.Debug,
		SkipPing: target.SkipPing,
		NonSSL:   target.ForceNonSSL,
		Timeout:  target.TimeOut,
	})
}

func SendElasticSearchServer(url string, namespace string, name string, body *tmaxiov1.ImageScanRequestStatus) (resp *http.Response, err error) {
	// send logging server
	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	requestUrl := url + "/image-scanning-" + namespace + "/_doc/" + name
	res, err := http.Post(requestUrl, "application/json", bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	return res, err
}
