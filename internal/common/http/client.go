package http

import (
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"strings"

	"github.com/go-logr/logr"
	regv1 "github.com/tmax-cloud/registry-operator/api/v1"
	"github.com/tmax-cloud/registry-operator/internal/common/auth"
	"github.com/tmax-cloud/registry-operator/internal/common/certs"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var logger logr.Logger = logf.Log.WithName("common http")

type HttpClient struct {
	Login    regv1.Authorizer
	URL      string
	CA       []byte
	Insecure bool
	Token    auth.Token
	*http.Client
}

func NewHTTPClient(url, username, password string, ca []byte, insecure bool) *HttpClient {
	if !strings.HasPrefix(url, "https://") && !strings.HasPrefix(url, "http://") {
		url = "https://" + url
	}

	if insecure {
		c := &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			},
		}

		return &HttpClient{
			URL:      url,
			Login:    regv1.Authorizer{Username: username, Password: password},
			CA:       ca,
			Insecure: insecure,
			Client:   c,
		}
	}

	caCertPool, err := x509.SystemCertPool()
	if err != nil {
		logger.Error(err, "failed to get system X509 cert pool")
		caCertPool = x509.NewCertPool()
		// add registry ca
		caSecret, _ := certs.GetSystemRootCASecret(nil)
		caCert, _ := certs.CAData(caSecret)
		if ok := caCertPool.AppendCertsFromPEM(caCert); !ok {
			logger.Info("failed to append registry ca cert", "ca", string(caCert))
		}
	}

	// add keycloak cert
	caSecret, _ := certs.GetSystemKeycloakCert(nil)
	if caSecret != nil {
		caCert, _ := certs.CAData(caSecret)
		if ok := caCertPool.AppendCertsFromPEM(caCert); !ok {
			logger.Info("failed to append keycloak ca cert", "ca", string(caCert))
		}
	}

	if len(ca) > 0 {
		if ok := caCertPool.AppendCertsFromPEM(ca); !ok {
			logger.Info("failed to append ca cert", "ca", string(ca))
		}
	}

	c := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs: caCertPool,
			},
		},
	}

	return &HttpClient{
		URL:      url,
		Login:    regv1.Authorizer{Username: username, Password: password},
		CA:       ca,
		Insecure: insecure,
		Client:   c,
	}
}
