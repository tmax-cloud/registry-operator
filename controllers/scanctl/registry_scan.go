package scanctl

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/genuinetools/reg/clair"
	reg "github.com/genuinetools/reg/clair"
	"github.com/genuinetools/reg/registry"
	"github.com/genuinetools/reg/repoutils"
	tmaxiov1 "github.com/tmax-cloud/registry-operator/api/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
)

var (
	// Priorities are the vulnerability priority labels.
	Priorities = []string{"Unknown", "Negligible", "Low", "Medium", "High", "Critical", "Defcon1"}
)

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
