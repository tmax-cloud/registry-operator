package scan

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"

	"github.com/docker/distribution"
	reg "github.com/genuinetools/reg/clair"
	"github.com/opencontainers/go-digest"
	"github.com/tmax-cloud/registry-operator/internal/common/config"
	"github.com/tmax-cloud/registry-operator/pkg/image"
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

func GetScanResult(img *image.Image) (ResultResponse, error) {
	if img == nil {
		return nil, fmt.Errorf("img cannot be nil")
	}

	// Get layers list
	manifest, err := img.GetManifest()
	if err != nil {
		log.Error(err, "failed to get manifest")
		return nil, err
	}

	// Get clair result for each layer
	filteredLayer, err := filterEmptyLayers(manifest.Manifest.References())
	if err != nil {
		return nil, err
	}

	if len(filteredLayer) == 0 {
		return nil, fmt.Errorf("all layers are empty")
	}

	// Fetch image's vulnerabilities (only fetch the top layer)
	vul, err := fetchClairResult(filteredLayer[0].Digest.String())
	if err != nil {
		log.Error(err, "")
		return nil, err
	}

	// Make as a map
	resp := ResultResponse{}
	for _, f := range vul.Layer.Features {
		for _, v := range f.Vulnerabilities {
			if resp[v.Severity] == nil {
				resp[v.Severity] = []reg.Vulnerability{}
			}
			resp[v.Severity] = append(resp[v.Severity], v)
		}
	}

	return resp, nil
}

func fetchClairResult(layerId string) (*ClairResponse, error) {
	clairServer := config.Config.GetString(config.ConfigImageScanSvr)
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

	return layer, nil
}

func filterEmptyLayers(layers []distribution.Descriptor) ([]distribution.Descriptor, error) {
	var results []distribution.Descriptor

	for _, l := range layers {
		d, err := digest.Parse(l.Digest.String())
		if err != nil {
			return nil, err
		}
		if !reg.IsEmptyLayer(d) {
			results = append(results, l)
		}
	}

	return results, nil
}
