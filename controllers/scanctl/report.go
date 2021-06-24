package scanctl

import (
	"bytes"
	"encoding/json"
	"fmt"
	tmaxiov1 "github.com/tmax-cloud/registry-operator/api/v1"
	"io/ioutil"
	"net/http"
)

type ReportClient struct {
	serverUrl string
	client    *http.Client
}

func NewReportClient(url string, transport *http.Transport) *ReportClient {
	return &ReportClient{
		serverUrl: url,
		client: &http.Client{
			Transport: transport,
		},
	}
}

func (c *ReportClient) SendReport(report *tmaxiov1.ImageScanRequestESReport) error {
	dat, err := json.Marshal(report)
	if err != nil {
		return err
	}

	response, err := c.client.Post(c.serverUrl, "application/json", bytes.NewReader(dat))
	if err != nil {
		return err
	}

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode >= 400 && response.StatusCode < 600 {
		return fmt.Errorf(fmt.Sprintf("ES server respond with %d(%s)\n", response.StatusCode, body))
	}
	return nil
}
