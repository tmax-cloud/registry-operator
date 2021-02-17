package scanctl

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	tmaxiov1 "github.com/tmax-cloud/registry-operator/api/v1"
)

type ReportClient struct {
	url string

	client *http.Client
}

func NewReportClient(url string, transport *http.Transport) *ReportClient {

	return &ReportClient{
		url: url,
		client: &http.Client{
			Transport: transport,
		},
	}
}

func (c *ReportClient) SendReport(namespace string, report *tmaxiov1.ImageScanRequestESReport) error {

	index := "/image-scanning-" + namespace
	doc := strings.ReplaceAll(report.Image, "/", "_")
	endpoint := fmt.Sprintf("%s/%s/_doc/%s", c.url, index, doc)
	dat, err := json.Marshal(report)
	if err != nil {
		return err
	}

	response, err := c.client.Post(endpoint, "application/json", bytes.NewReader(dat))
	if err != nil {
		return err
	}
	defer response.Body.Close()
	return err
}
