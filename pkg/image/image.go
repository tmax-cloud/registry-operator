package image

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"path"

	"github.com/docker/distribution/registry/client"
	"github.com/docker/distribution/registry/client/auth/challenge"
	"github.com/tmax-cloud/registry-operator/internal/common/auth"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var Logger = log.Log.WithName("image-client")

const (
	DefaultServer = "https://registry-1.docker.io"
)

type Image struct {
	ServerURL string

	Host string
	Name string
	Tag  string

	BasicAuth string
	Token     auth.Token

	HttpClient http.Client
}

func NewImage(uri, registryServer, basicAuth string, ca []byte) (*Image, error) {
	r := &Image{}

	if uri != "" {
		// Parse image url
		img, err := ParseNamed(uri)
		if err != nil {
			return nil, err
		}
		img = WithDefaultTag(img)
		r.Host = img.Hostname()
		r.Name = img.RemoteName()
		if tagged, isTagged := img.(NamedTagged); isTagged {
			r.Tag = tagged.Tag()
		} else {
			return nil, fmt.Errorf("no tag given")
		}
	}

	// Server url
	if registryServer == "" {
		r.ServerURL = DefaultServer
	} else {
		r.ServerURL = registryServer
	}

	// Auth
	r.BasicAuth = basicAuth
	r.Token = auth.Token{}

	// Generate HTTPS client
	var tlsConfig *tls.Config
	if len(ca) == 0 {
		tlsConfig = &tls.Config{InsecureSkipVerify: true}
	} else {
		caPool := x509.NewCertPool()
		caPool.AppendCertsFromPEM(ca)
		tlsConfig = &tls.Config{
			RootCAs: caPool,
		}
	}
	r.HttpClient = http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}

	return r, nil
}

func (r *Image) SetImage(uri string) error {
	// Parse image url
	img, err := ParseNamed(uri)
	if err != nil {
		return err
	}
	img = WithDefaultTag(img)
	r.Host = img.Hostname()
	r.Name = img.RemoteName()
	if tagged, isTagged := img.(NamedTagged); isTagged {
		r.Tag = tagged.Tag()
	} else {
		return fmt.Errorf("no tag given")
	}

	return nil
}

func (r *Image) GetImageNameWithHost() string {
	return path.Join(r.Host, r.Name)
}

func (r *Image) GetToken(scope string) (auth.Token, error) {
	if scope == "" {
		if r.Name == "" {
			scope = catalogScope()
		} else {
			scope = repositoryScope(r.Name)
		}
	}

	if err := r.fetchToken(scope); err != nil {
		Logger.Error(err, "")
		return auth.Token{}, err
	}

	return r.Token, nil
}

func (r *Image) fetchToken(scope string) error {
	Logger.Info("Fetching token...")
	// Ping
	u, err := pingURL(r.ServerURL)
	if err != nil {
		return err
	}

	pingReq, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return err
	}
	pingReq.Header.Set("Authorization", fmt.Sprintf("Basic %s", r.BasicAuth))
	pingResp, err := r.HttpClient.Do(pingReq)
	if err != nil {
		return err
	}
	defer pingResp.Body.Close()

	// If 200, use basic auth
	if pingResp.StatusCode >= 200 && pingResp.StatusCode < 300 {
		r.Token = auth.Token{
			Type:  "Basic",
			Value: base64.StdEncoding.EncodeToString([]byte(r.BasicAuth)),
		}
		return nil
	}

	challenges := challenge.ResponseChallenges(pingResp)
	if len(challenges) < 1 {
		return fmt.Errorf("header does not contain WWW-Authenticate")
	}
	realm, realmExist := challenges[0].Parameters["realm"]
	service, serviceExist := challenges[0].Parameters["service"]
	if !realmExist || !serviceExist {
		return fmt.Errorf("there is no realm or service in parameters")
	}

	// Get Token
	param := map[string]string{
		"service": service,
		"scope":   scope,
	}
	tokenReq, err := http.NewRequest(http.MethodGet, realm, nil)
	if err != nil {
		return err
	}
	tokenReq.Header.Set("Authorization", fmt.Sprintf("Basic %s", r.BasicAuth))
	tokenQ := tokenReq.URL.Query()
	for k, v := range param {
		tokenQ.Add(k, v)
	}
	tokenReq.URL.RawQuery = tokenQ.Encode()

	tokenResp, err := r.HttpClient.Do(tokenReq)
	if err != nil {
		return err
	}
	defer tokenResp.Body.Close()
	if !client.SuccessStatus(tokenResp.StatusCode) {
		err := client.HandleErrorResponse(tokenResp)
		return err
	}

	decoder := json.NewDecoder(tokenResp.Body)
	token := &auth.TokenResponse{}
	if err := decoder.Decode(token); err != nil {
		return err
	}

	r.Token = auth.Token{
		Type:  "Bearer",
		Value: token.Token,
	}

	return nil
}
