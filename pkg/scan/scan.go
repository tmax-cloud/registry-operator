package scan

import (
	"encoding/json"
	"fmt"
	reg "github.com/genuinetools/reg/clair"
	"github.com/tmax-cloud/registry-operator/pkg/trust"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	ctrl "sigs.k8s.io/controller-runtime"
)

var log = ctrl.Log.WithName("scan")

// For scan requests
type Request struct {
	Registries []RequestRegistry `json:"registries"`
}

type RequestRegistry struct {
	Name         string              `json:"name"`
	Repositories []RequestRepository `json:"repositories"`
}

type RequestRepository struct {
	Name     string   `json:"name"`
	Versions []string `json:"versions"`
}

type RequestResponse struct {
	ImageScanRequestName string `json:"imageScanRequestName"`
}

// For scan responses
type ResultResponse map[string][]reg.Vulnerability

type ClairResponse struct {
	Layer reg.Layer `json:"Layer"`
}

func GetScanResult(img *trust.Image) (ResultResponse, error) {
	if img == nil {
		return nil, fmt.Errorf("img cannot be nil")
	}

	// Get layers list
	manifest, err := img.GetImageManifest()
	if err != nil {
		log.Error(err, "")
		return nil, err
	}

	// Get clair result for each layer
	var vuls []reg.Vulnerability
	for _, l := range manifest.Layers {
		vul, err := fetchClairResult(l.Digest)
		if err != nil {
			log.Error(err, "")
			return nil, err
		}
		vuls = append(vuls, vul...)
	}

	// Make as a map
	resp := ResultResponse{}
	for _, v := range vuls {
		if resp[v.Severity] == nil {
			resp[v.Severity] = []reg.Vulnerability{}
		}
		resp[v.Severity] = append(resp[v.Severity], v)
	}

	return resp, nil
}

func fetchClairResult(layerId string) ([]reg.Vulnerability, error) {
	clairServer := os.Getenv("CLAIR_URL")
	if clairServer == "" {
		return nil, fmt.Errorf("CLAIR_URL is not set")
	}

	u, err := url.Parse(clairServer)
	if err != nil {
		log.Error(err, "")
		return nil, err
	}
	u.Path = path.Join(u.Path, fmt.Sprintf("v1/layers/%s", layerId))
	tokenQ := u.Query()
	tokenQ.Add("features", "")
	tokenQ.Add("vulnerabilities", "")
	u.RawQuery = tokenQ.Encode()

	resp, err := http.Get(u.String())
	if err != nil {
		log.Error(err, "")
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Error(err, "")
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("error: %d, msg: %s", resp.StatusCode, string(body))
	}

	layer := &ClairResponse{}
	if err := json.Unmarshal(body, layer); err != nil {
		return nil, err
	}

	var results []reg.Vulnerability
	for _, f := range layer.Layer.Features {
		results = append(results, f.Vulnerabilities...)
	}

	return results, nil
}
