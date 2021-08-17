package scanctl

import (
	"bytes"
	"encoding/json"
	"fmt"
	tmaxiov1 "github.com/tmax-cloud/registry-operator/api/v1"
	"io/ioutil"
	"net/http"
	"net/url"
)

type ReportClient struct {
	targets []string
	client  *http.Client
}

func NewReportClient(targetUrls []string, transport *http.Transport) *ReportClient {
	return &ReportClient{
		targets: targetUrls,
		client: &http.Client{
			Transport: transport,
		},
	}
}

func (c *ReportClient) SendReport(namespace string, report *tmaxiov1.ImageScanRequestESReport) error {
	dat, err := json.Marshal(report)
	if err != nil {
		return err
	}

	for _, target := range c.targets {
		u := fmt.Sprintf("%s/image-scanning-%s/_doc/%s", target, namespace, url.PathEscape(report.Image))
		req, err := http.NewRequest(http.MethodPost, u, bytes.NewReader(dat))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")

		res, err := c.client.Do(req)
		if err != nil {
			return err
		}

		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return err
		}

		res.Body.Close()
		if res.StatusCode >= 300 {
			return fmt.Errorf("[%d] failed to send report: %s\n", res.StatusCode, body)
		}
	}

	return nil
}
