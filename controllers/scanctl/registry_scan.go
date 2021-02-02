package scanctl

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"path"
	"strings"
	"time"
<<<<<<< HEAD
<<<<<<< HEAD

	"github.com/cloudflare/cfssl/log"

=======
	
>>>>>>> Refactor GetVulnerability
=======

	"github.com/cloudflare/cfssl/log"

>>>>>>> [refactor] refactor registry_scan
	"github.com/genuinetools/reg/clair"
	"github.com/genuinetools/reg/registry"
	"github.com/genuinetools/reg/repoutils"
<<<<<<< HEAD
	tmaxiov1 "github.com/tmax-cloud/registry-operator/api/v1"
	"github.com/tmax-cloud/registry-operator/internal/common/certs"
	regConfig "github.com/tmax-cloud/registry-operator/internal/common/config"
	"github.com/tmax-cloud/registry-operator/internal/utils"
	"github.com/tmax-cloud/registry-operator/pkg/image"
	clairReg "github.com/tmax-cloud/registry-operator/pkg/scan/clair"
=======
	corev1 "k8s.io/api/core/v1"
>>>>>>> [refactor] refactor registry_scan
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	tmaxiov1 "github.com/tmax-cloud/registry-operator/api/v1"
	"github.com/tmax-cloud/registry-operator/internal/auth"
	regConfig "github.com/tmax-cloud/registry-operator/internal/common/config"
	"github.com/tmax-cloud/registry-operator/internal/utils"
	regApi "github.com/tmax-cloud/registry-operator/pkg/registry"
	clairReg "github.com/tmax-cloud/registry-operator/pkg/scan/clair"
)

var logger = logf.Log.WithName("registry_scan")

var (
	// Priorities are the vulnerability priority labels.
	Priorities = []string{"Unknown", "Negligible", "Low", "Medium", "High", "Critical", "Defcon1"}
)

func ParseAnalysis(threshold int, report *clair.VulnerabilityReport) (map[string]int, []string, map[string]tmaxiov1.Vulnerabilities) {
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
				//Metadata:      obj,
				FixedBy: v.FixedBy,
			}
			vuls = append(vuls, vul)
		}
		vulnerabilities[sev] = vuls
	}

	for _, val := range Priorities {
		summary[val] = 0
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

func getCertificateFromSecret(secretName, namespace string) ([]byte, error) {

<<<<<<< HEAD
=======
	secret, err := getSecret(secretName, namespace)
	if err != nil {
		return nil, errors.NewBadRequest("Cannot find secret named" + secretName + " from " + namespace)
	}

	if secret.Type == corev1.SecretTypeTLS {
		return secret.Data["tls.crt"], nil
	} else if secret.Type == corev1.SecretTypeOpaque {
		logger.Info("[WARN]: Certificate Secret for registry client is used as opaque type. (Using TLS type recommended)")
		return secret.Data["ca.crt"], nil
	}

	return nil, errors.NewBadRequest("Only TLS Secret is allowed.")
}

>>>>>>> [refactor] refactor registry_scan
func GetRegistryImages(c client.Client, registryURL, basicAuth, imageNamePattern, certSecret, namespace string) []string {
	images := []string{}
	var ca []byte

	// set certificate
	if certSecret != "" {
		var err error

<<<<<<< HEAD
		ca, err = utils.GetCAData(certSecret, namespace)
=======
		ca, err = getCertificateFromSecret(certSecret, namespace)
>>>>>>> [refactor] refactor registry_scan
		if err != nil {
			logger.Error(err, "failed to get ca")
			return images
		}
	}

	img, err := image.NewImage("", registryURL, basicAuth, ca)
	if err != nil {
		logger.Error(err, "faild to create image client")
	}
	repos := img.Catalog()
	if repos == nil {
		return images
	}
	for _, repo := range repos.Repositories {
		if err := img.SetImage(repo); err != nil {
			logger.Error(err, "failed to set image", "repo", repo)
			continue
		}
		vers := img.Tags()
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

<<<<<<< HEAD
<<<<<<< HEAD
func GetVulnerability(c client.Client, instance *tmaxiov1.ImageScanRequest) (map[string]map[string]*reg.VulnerabilityReport, error) {
	reports := map[string]map[string]*reg.VulnerabilityReport{}

	//get clair url
	clairServer := regConfig.Config.GetString(regConfig.ConfigClairURL)
	if len(clairServer) == 0 {
		return reports, errors.NewBadRequest("cannot find clairUrl")
=======
func getBasicAuth(imagePullSecret, namespace, registryURL string) (string, error) {
	logger.Info("Get " + imagePullSecret + "secret from " + namespace)
	secret, err := getSecret(imagePullSecret, namespace)
=======
func getBasicAuthTokenFromImagePullSecret(secretName, namespace, registryURL string) (string, error) {
	//
	secret, err := getSecret(secretName, namespace)
>>>>>>> [refactor] refactor registry_scan
	if err != nil {
		logger.Error(err, "Failed to get imagePullSecret from "+namespace)
		return "", err
	}

	// build basicAuth
	basic, err := utils.ParseBasicAuth(secret, registryURL)
	if err != nil {
		logger.Error(err, "failed to parse basic auth")
		return "", err
	}

	logger.Info("Parsed basic: " + basic)
	return basic, nil
}

func setImageNames(c client.Client, image string, imagePullSecret string, certificateSecret string, namespace string, registryURL string) ([]string, error) {
	var entries = []string{}

	if strings.Contains(image, "*") || strings.Contains(image, "?") {
		var basicAuth string

		if imagePullSecret == "" {
			return entries, errors.NewBadRequest("Image(" + image + ")'s ImagePullSecret not provided.")
		}

		basic, err := getBasicAuthTokenFromImagePullSecret(imagePullSecret, namespace, registryURL)
		if err != nil {
			return entries, errors.NewBadRequest("Failed to get basic auth from imagePullSecret" + imagePullSecret)
		}

		basicAuth = basic
		entries = append(entries, GetRegistryImages(c, registryURL, basicAuth, image, certificateSecret, namespace)...)
	} else {
		entries = append(entries, image)
	}

	return entries, nil
}
func fetchImageList(reg *registry.Registry, images []string) ([]string, error) {

	var ret = []string{}

	for _, entry := range images {
		if strings.Contains(entry, "*") {
			reg.Catalog()
		} else if strings.Contains(entry, "?") {

		} else {
			ret = append(ret, entry)
		}
	}

	return ret, nil
}

func GetVulnerability(c client.Client, o *tmaxiov1.ImageScanRequest) (map[string]map[string]*clair.VulnerabilityReport, error) {

	results := map[string]map[string]*clair.VulnerabilityReport{}

	scannerAddr := regConfig.Config.GetString("clair.url")
	if len(scannerAddr) == 0 {
<<<<<<< HEAD
		return reports, errors.NewBadRequest("Cannot get address of Clair server.")
>>>>>>> Refactor GetVulnerability
=======
		return results, errors.NewBadRequest("Cannot get address of Clair server.")
>>>>>>> [refactor] refactor registry_scan
	}

	for _, scanTarget := range o.Spec.ScanTargets {
		if scanTarget.TimeOut == 0 {
			scanTarget.TimeOut = time.Minute
		}

		if len(scanTarget.RegistryURL) < 1 {
			return nil, errors.NewBadRequest("Empty registry URL")
		} else if len(scanTarget.ImagePullSecret) < 1 {
			return nil, errors.NewBadRequest("Empty ImagePullSecret")
		} else if len(scanTarget.CertificateSecret) < 1 {
			return nil, errors.NewBadRequest("Empty CertificateSecret")
		} else if !scanTarget.ForceNonSSL && strings.HasPrefix(config.ServerAddress, "http:") {
			return nil, errors.NewBadRequest("attempted to use insecure protocol! Use force-non-ssl option to force")
		} else if scanTarget.FixableThreshold < 0 {
			return results, errors.NewBadRequest("fixable threshold must be a positive integer")
		}

<<<<<<< HEAD
		for _, targetImage := range target.Images {
<<<<<<< HEAD
			matchImages := []string{}
			if strings.Contains(targetImage, "*") || strings.Contains(targetImage, "?") {
				var basicAuth string
				if target.ImagePullSecret != "" {
					basic, err := utils.GetBasicAuth(target.ImagePullSecret, instance.Namespace, target.RegistryURL)
					if err != nil {
						logger.Error(err, fmt.Sprintf("failed to get basic auth from imagepullsecret: %s", target.ImagePullSecret))
						return reports, err
					}
					basicAuth = basic
				}
				matchImages = append(matchImages, GetRegistryImages(c, target.RegistryURL, basicAuth, targetImage, target.CertificateSecret, instance.Namespace)...)
			} else {
				matchImages = append(matchImages, targetImage)
=======
			imageNames, err := setImageNames(c, targetImage, target.ImagePullSecret, target.CertificateSecret, o.Namespace, target.RegistryURL)
=======
		dockerLoginSecret := getSecret(scanTarget.ImagePullSecret, namespace)
		dockerLogin := auth.NewLoginProvider(dockerLoginSecret)

		caSecret := getSecret(scanTarget.CertificateSecret, namespace)
		ca := auth.NewCertProvider(caSecret)

		var registryAddr string
		if len(scanTarget.AuthURL) > 0 {
			registryAddr = scanTarget.AuthURL
		} else {
			registryAddr = scanTarget.RegistryURL
		}

		regClient := newRegClient(registryAddr, dockerLogin, ca)

		for _, scanTargetImage := range scanTarget.Images {
			imageNames, err := setImageNames(c, scanTargetImage, scanTarget.ImagePullSecret, scanTarget.CertificateSecret, o.Namespace, scanTarget.RegistryURL)
>>>>>>> [refactor] refactor registry_scan
			if err != nil {
				logger.Error(err, "Failed to set image entries.")
>>>>>>> Refactor GetVulnerability
			}

			for _, name := range imageNames {
				image, err := registry.ParseImage(path.Join(scanTarget.RegistryURL, name))
				if err != nil {
					logger.Error(err, "failed to parse image")
<<<<<<< HEAD
					return reports, err
				}

				// Create the registry client.
				r, err := createRegistryClient(&target, image.Domain, o.Namespace)
				if err != nil {
					logger.Error(err, "failed to create registry client")
					return reports, err
				}

				// Initialize clair client.
				cr, err := clair.New(scannerAddr, clair.Opt{
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
					logger.Error(err, "failed to get image vulnerabilities")
					return reports, err
=======
					return results, err
>>>>>>> [refactor] refactor registry_scan
				}
				logger.Info("[Parsed image]: Domain: " + image.Domain + " /Path: " + image.Path + " /Tag: " + image.Tag)
				targets.append(image)
			}
		}
	}

	// for _, scanTarget := range targets {
	// 			// Initialize clair client.
	// 			cr, err := clair.New(scannerAddr, clair.Opt{
	// 				Debug:    scanTarget.Debug,
	// 				Timeout:  scanTarget.TimeOut,
	// 				Insecure: scanTarget.Insecure,
	// 			})
	// 			if err != nil {
	// 				logger.Error(err, "failed to new clair client")
	// 				return results, err
	// 			}

	// 			report := clair.VulnerabilityReport{}

	// 			// Get the vulnerability report.
	// 			if report, err = cr.Vulnerabilities(context.TODO(), r, image.Path, image.Reference()); err != nil {
	// 				logger.Info(err)
	// 				// Fallback to Clair v3 API.
	// 				if report, err = cr.VulnerabilitiesV3(context.TODO(), r, image.Path, image.Reference()); err != nil {
	// 					logger.Error(err, "failed to check vulnerabilities")
	// 					return results, err
	// 				}
	// 			}

	// 			// set report
	// 			if m, ok := results[scanTarget.RegistryURL]; ok {
	// 				m[name] = &report
	// 				results[scanTarget.RegistryURL] = m
	// 			} else {
	// 				results[scanTarget.RegistryURL] = map[string]*clair.VulnerabilityReport{name: &report}
	// 			}

	// 		}
	// 	}
	// }

	return results, nil
}

<<<<<<< HEAD
func createRegistryClient(target *tmaxiov1.ScanTarget, domain, namespace string) (*registry.Registry, error) {
	var ca []byte
	username, password := "", ""

	// Use the auth-url domain if provided.
	authDomain := target.AuthURL
	if authDomain == "" {
		authDomain = domain
	}

	// parse imagepullsecret
	if len(target.ImagePullSecret) > 0 {
		basic, err := utils.GetBasicAuth(target.ImagePullSecret, namespace, target.RegistryURL)
		if err != nil {
			logger.Error(err, fmt.Sprintf("failed to get basic auth from imagepullsecret: %s", target.ImagePullSecret))
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

	// get ca data
	if target.CertificateSecret != "" {
		var err error

		ca, err = utils.GetCAData(target.CertificateSecret, namespace)
		if err != nil {
			logger.Error(err, "failed to get ca data")
			return nil, err
		}
	}

	// get keycloak ca if exists
	keycloakCA, err := certs.GetSystemKeycloakCert(nil)
	if err != nil && !errors.IsNotFound(err) {
		logger.Error(err, "failed to get system keycloak cert")
		return nil, err
	}

	if keycloakCA != nil {
		caCert, _ := certs.CAData(keycloakCA)
		ca = append(ca, caCert...)
	}

	// get auth config
	auth, err := repoutils.GetAuthConfig(username, password, authDomain)
=======
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

func newRegClient(registryUrl string, login *auth.LoginProvider, cert *auth.CertProvider) (*registry.Registry, error) {

	config, err := repoutils.GetAuthConfig(login.getID(), login.getPassword(), registryUrl)
>>>>>>> [refactor] refactor registry_scan
	if err != nil {
		logger.Error(err, "Failed to get auth config")
		return nil, err
	}

	// TODO: Set opt from operator's registry config.
	return clairReg.New(context.TODO(), config, registry.Opt{
		Insecure: false,
		Debug:    true,
		SkipPing: true,
		NonSSL:   false,
		Timeout:  time.Minute,
	}, cert.getCert())
}

func SendElasticSearchServer(url string, namespace string, name string, body *tmaxiov1.ImageScanRequestESReport) (resp *http.Response, err error) {
	// send logging server
	data, err := json.Marshal(body)
	if err != nil {
		logger.Error(err, "failed to marshal elastic search report")
		return nil, err
	}

	image := strings.ReplaceAll(body.Image, "/", "_")
	requestUrl := url + "/image-scanning-" + namespace + "/_doc/" + image
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	cli := &http.Client{Transport: tr}
	res, err := cli.Post(requestUrl, "application/json", bytes.NewReader(data))
	if err != nil {
		logger.Error(err, "failed to post ES Server")
		return nil, err
	}
	log.Info(res.StatusCode)
	resBody, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	log.Info(string(resBody))
	defer res.Body.Close()
	return res, err
}
