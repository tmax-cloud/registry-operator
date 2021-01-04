package http

import (
	"crypto/tls"
	"crypto/x509"
	"net/http"

	"github.com/go-logr/logr"
	"github.com/tmax-cloud/registry-operator/internal/common/certs"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var logger logr.Logger = logf.Log.WithName("common http")

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
	caCertPool := x509.NewCertPool()

	// add registry ca
	caSecret, _ := certs.GetSystemRootCASecret(nil)
	caCert, _ := certs.CAData(caSecret)
	caCertPool.AppendCertsFromPEM(caCert)

	// add keycloak cert
	caSecret, _ = certs.GetSystemKeycloakCert(nil)
	if caSecret != nil {
		logger.Info("append keycloak cert")
		caCert, _ = certs.CAData(caSecret)
		caCertPool.AppendCertsFromPEM(caCert)
	}

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
