package http

import (
	"crypto/tls"
	"crypto/x509"
	"net/http"
)

type Authorizer struct {
	Username string
	Password string
}

type HttpClient struct {
	Login Authorizer
	URL   string
	*http.Client
}

func NewHTTPClient(url, username, password string) *HttpClient {
	caCert, _ := CAData()
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	c := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs: caCertPool,
			},
		},
	}

	return &HttpClient{
		URL:    url,
		Login:  Authorizer{Username: username, Password: password},
		Client: c,
	}
}
