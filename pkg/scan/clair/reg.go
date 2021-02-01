package clair

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"regexp"
	"strings"

	"github.com/docker/docker/api/types"
	reg "github.com/genuinetools/reg/registry"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var logger = logf.Log.WithName("registry-clair")
var reProtocol = regexp.MustCompile("^https?://")

// New creates a new Registry struct with the given URL and credentials.
func New(ctx context.Context, auth types.AuthConfig, opt reg.Opt, ca []byte) (*reg.Registry, error) {
	transport := http.DefaultTransport

	if opt.Insecure {
		transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		}
	} else if len(ca) > 0 {
		caPool, err := x509.SystemCertPool()
		if err != nil {
			logger.Error(err, "failed to get system cert pool")
			return newFromTransport(ctx, auth, transport, opt)
		}

		if ok := caPool.AppendCertsFromPEM(ca); !ok {
			logger.Info("failed to append external registry ca cert", "ca", string(ca))
		}
		tlsConfig := &tls.Config{
			RootCAs: caPool,
		}
		transport = &http.Transport{
			TLSClientConfig: tlsConfig,
		}
	}

	return newFromTransport(ctx, auth, transport, opt)
}

func newFromTransport(ctx context.Context, auth types.AuthConfig, transport http.RoundTripper, opt reg.Opt) (*reg.Registry, error) {
	if len(opt.Domain) < 1 || opt.Domain == "docker.io" {
		opt.Domain = auth.ServerAddress
	}
	url := strings.TrimSuffix(opt.Domain, "/")
	authURL := strings.TrimSuffix(auth.ServerAddress, "/")

	if !reProtocol.MatchString(url) {
		if !opt.NonSSL {
			url = "https://" + url
		} else {
			url = "http://" + url
		}
	}

	if !reProtocol.MatchString(authURL) {
		if !opt.NonSSL {
			authURL = "https://" + authURL
		} else {
			authURL = "http://" + authURL
		}
	}

	tokenTransport := &reg.TokenTransport{
		Transport: transport,
		Username:  auth.Username,
		Password:  auth.Password,
	}
	basicAuthTransport := &reg.BasicTransport{
		Transport: tokenTransport,
		URL:       authURL,
		Username:  auth.Username,
		Password:  auth.Password,
	}
	errorTransport := &reg.ErrorTransport{
		Transport: basicAuthTransport,
	}
	customTransport := &reg.CustomTransport{
		Transport: errorTransport,
		Headers:   opt.Headers,
	}

	// set the logging
	logf := reg.Quiet
	if opt.Debug {
		logf = reg.Log
	}

	registry := &reg.Registry{
		URL:    url,
		Domain: reProtocol.ReplaceAllString(url, ""),
		Client: &http.Client{
			Timeout:   opt.Timeout,
			Transport: customTransport,
		},
		Username: auth.Username,
		Password: auth.Password,
		Logf:     logf,
		Opt:      opt,
	}

	if registry.Pingable() && !opt.SkipPing {
		if err := registry.Ping(ctx); err != nil {
			logger.Error(err, "failed to ping to registry")
			return nil, err
		}
	}

	return registry, nil
}
