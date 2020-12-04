package trust

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"github.com/docker/distribution/reference"
	"github.com/docker/distribution/registry/client"
	"github.com/docker/distribution/registry/client/auth/challenge"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"time"
)

const (
	DefaultServer       = "https://registry-1.docker.io"
	DefaultNotaryServer = "https://notary.docker.io"
)

type Image struct {
	ServerUrl       string
	NotaryServerUrl string

	Host string
	Name string
	Tag  string

	BasicAuth string
	Tokens    map[TokenType]string

	HttpClient http.Client
}

func NewImage(uri, registryServer, notaryServer, basicAuth string, ca []byte) (*Image, error) {
	r := &Image{}

	// Parse image url
	img, err := ParseNamed(uri)
	if err != nil {
		return nil, err
	}
	img = WithDefaultTag(img)
	r.Host = img.Hostname()
	r.Name = img.RemoteName()
	if tagged, isTagged := img.(reference.NamedTagged); isTagged {
		r.Tag = tagged.Tag()
	} else {
		return nil, fmt.Errorf("no tag given")
	}

	// Server url
	if registryServer == "" {
		r.ServerUrl = DefaultServer
	} else {
		r.ServerUrl = registryServer
	}

	// Notary Server url
	if notaryServer == "" {
		r.NotaryServerUrl = DefaultNotaryServer
	} else {
		r.NotaryServerUrl = notaryServer
	}

	// Auth
	r.BasicAuth = basicAuth
	r.Tokens = map[TokenType]string{}

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

func (r Image) GetImageNameWithHost() string {
	return path.Join(r.Host, r.Name)
}

func (r Image) GetImageManifest() (string, int64, error) {
	u, err := url.Parse(r.ServerUrl)
	if err != nil {
		return "", 0, err
	}
	u.Path = path.Join(u.Path, fmt.Sprintf("v2/%s/manifests/%s", r.Name, r.Tag))
	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return "", 0, err
	}

	token, err := r.GetToken(TokenTypeRegistry)
	if err != nil {
		return "", 0, err
	}

	req.Header.Set("Accept", "application/vnd.docker.distribution.manifest.v2+json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	resp, err := r.HttpClient.Do(req)
	if err != nil {
		return "", 0, err
	}
	defer resp.Body.Close()
	if !client.SuccessStatus(resp.StatusCode) {
		err := client.HandleErrorResponse(resp)
		return "", 0, err
	}

	digest := resp.Header.Get("Docker-Content-Digest")
	lengthStr := resp.Header.Get("Content-Length")

	if digest == "" || lengthStr == "" {
		return "", 0, fmt.Errorf("expected headers not exist")
	}

	length, err := strconv.Atoi(lengthStr)
	if err != nil {
		return "", 0, err
	}

	return digest, int64(length), nil
}

type TokenType string

const (
	TokenTypeRegistry = TokenType("registry")
	TokenTypeNotary   = TokenType("notary")
)

func (r Image) GetToken(tokenType TokenType) (string, error) {
	t, ok := r.Tokens[tokenType]
	if !ok {
		err := r.fetchToken(tokenType)
		if err != nil {
			return "", err
		}
		t, ok = r.Tokens[tokenType]
		if !ok {
			return "", fmt.Errorf("no token is fetched for %s", tokenType)
		}
	}
	return t, nil
}

func (r Image) fetchToken(tokenType TokenType) error {
	// Ping
	var server string
	switch tokenType {
	case TokenTypeRegistry:
		server = r.ServerUrl
	case TokenTypeNotary:
		server = r.NotaryServerUrl
	default:
		return fmt.Errorf("token type %s is not supported", tokenType)
	}
	u, err := url.Parse(server)
	if err != nil {
		return err
	}
	u.Path = path.Join(u.Path, "v2")
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
	scope, scopeExist := challenges[0].Parameters["scope"]
	if !scopeExist {
		img := r.Name
		if tokenType == TokenTypeNotary {
			img = r.GetImageNameWithHost()
		}
		scope = fmt.Sprintf("repository:%s:pull,push", img)
	}
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
	token := &tokenResponse{}
	if err := decoder.Decode(token); err != nil {
		return err
	}

	r.Tokens[tokenType] = token.Token

	return nil
}

type tokenResponse struct {
	Token        string    `json:"token"`
	AccessToken  string    `json:"access_token"`
	ExpiresIn    int       `json:"expires_in"`
	IssuedAt     time.Time `json:"issued_at"`
	RefreshToken string    `json:"refresh_token"`
}
