package registry

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/cloudflare/cfssl/log"
	"github.com/docker/distribution/registry/client/auth/challenge"
	apiv1 "github.com/tmax-cloud/registry-operator/api/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	regClient "github.com/docker/distribution/registry/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

type Repositories struct {
	Repositories []string `json:"repositories"`
}

type Repository struct {
	Name string   `json:"name"`
	Tags []string `json:"tags"`
}

var logger = logf.Log.WithName("pkg-registry-api")

func getRegistry(c client.Client, regName, namespace string) (*apiv1.Registry, error) {
	reg := &apiv1.Registry{}
	if err := c.Get(context.TODO(), types.NamespacedName{Name: regName, Namespace: namespace}, reg); err != nil {
		return nil, err
	}

	return reg, nil
}

type RegCtl struct {
	client client.Client
	reg    *apiv1.Registry
}

// NewRegCtl is a controller for registry
// if registryName or registryNamespace is empty string, RegCtl is nil
func NewRegCtl(c client.Client, regName, namespace string) *RegCtl {
	if len(regName) == 0 || len(namespace) == 0 {
		return nil
	}

	reg, err := getRegistry(c, regName, namespace)
	if err != nil {
		return nil
	}

	return &RegCtl{
		client: c,
		reg:    reg,
	}
}

func (r *RegCtl) GetHostname() string {
	return strings.TrimPrefix(r.GetEndpoint(), "https://")
}

func (r *RegCtl) GetEndpoint() string {
	return r.reg.Status.ServerURL
}

func (r *RegCtl) GetNotaryEndpoint() string {
	return r.reg.Status.NotaryURL
}

const DefaultServer = "https://registry-1.docker.io"

type RegistryAPI struct {
	Scheme    string
	ServerURL string

	BasicAuth string
	Token     *Token

	HttpClient http.Client
}

type TokenType string

const (
	TokenTypeBasic  TokenType = "Basic"
	TokenTypeBearer TokenType = "Bearer"
)

type Token struct {
	Type  TokenType
	Value string
}

func NewRegistryAPI(serverURL, basicAuth string, ca []byte) *RegistryAPI {
	var scheme string
	if serverURL == "" || serverURL == "docker.io" {
		serverURL = DefaultServer
	}

	if strings.HasPrefix(serverURL, "http://") {
		scheme = "http://"
		serverURL = serverURL[len("http://"):]
	} else if strings.HasPrefix(serverURL, "https://") {
		scheme = "https://"
		serverURL = serverURL[len("https://"):]
	} else {
		scheme = "https://"
	}

	var tlsConfig *tls.Config
	if len(ca) == 0 {
		tlsConfig = &tls.Config{InsecureSkipVerify: true}
	} else {
		caPool := x509.NewCertPool()
		if ok := caPool.AppendCertsFromPEM(ca); !ok {
			logger.Info("failed to append external registry ca cert", "ca", string(ca))
		}
		tlsConfig = &tls.Config{
			RootCAs: caPool,
		}
	}
	httpClient := http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}

	return &RegistryAPI{
		Scheme:    scheme,
		ServerURL: serverURL,
		BasicAuth: basicAuth,

		HttpClient: httpClient,
	}
}

func (r *RegistryAPI) toggleScheme() {
	if r.Scheme == "http://" {
		r.Scheme = "https://"
		return
	}

	r.Scheme = "http://"
}

func (r *RegistryAPI) GetToken(scope string) (*Token, error) {
	token, err := r.fetchToken(scope)
	if err != nil {
		logger.Error(err, "failed to fetch token... retrying by changing scheme ...")
		r.toggleScheme()
		t, err := r.fetchToken(scope)
		if err != nil {
			logger.Error(err, "failed to fetch token")
			return nil, err
		}
		token = t
	}
	return token, nil
}

func (r *RegistryAPI) fetchToken(scope string) (*Token, error) {
	log.Info("Fetching token...")
	server := r.Scheme + r.ServerURL
	// Ping
	u, err := url.Parse(server)
	if err != nil {
		logger.Error(err, "failed to parse url")
		return nil, err
	}
	u.Path = path.Join(u.Path, "v2")
	pingReq, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		logger.Error(err, "failed to create ping request")
		return nil, err
	}
	pingReq.Header.Set("Authorization", fmt.Sprintf("Basic %s", r.BasicAuth))
	pingResp, err := r.HttpClient.Do(pingReq)
	if err != nil {
		logger.Error(err, "failed to ping request")
		return nil, err
	}
	defer pingResp.Body.Close()

	// If 200, use basic auth
	if pingResp.StatusCode >= 200 && pingResp.StatusCode < 300 {
		r.Token = &Token{
			Type:  TokenTypeBasic,
			Value: r.BasicAuth,
		}
		return r.Token, nil
	}

	challenges := challenge.ResponseChallenges(pingResp)
	if len(challenges) < 1 {
		return nil, fmt.Errorf("header does not contain WWW-Authenticate")
	}
	realm, realmExist := challenges[0].Parameters["realm"]
	service, serviceExist := challenges[0].Parameters["service"]
	if !realmExist || !serviceExist {
		return nil, fmt.Errorf("there is no realm or service in parameters")
	}

	if scope == "" {
		scope = "registry:catalog:*"
	}

	// Get Token
	param := map[string]string{
		"service": service,
		"scope":   scope,
	}
	tokenReq, err := http.NewRequest(http.MethodGet, realm, nil)
	if err != nil {
		logger.Error(err, "failed to create request")
		return nil, err
	}
	tokenReq.Header.Set("Authorization", fmt.Sprintf("Basic %s", r.BasicAuth))
	tokenQ := tokenReq.URL.Query()
	for k, v := range param {
		tokenQ.Add(k, v)
	}
	tokenReq.URL.RawQuery = tokenQ.Encode()

	// logger.Info(fmt.Sprintf("url=%s, service=%s, scope=%s, realm=%s, basicauth=%s", server, service, scope, realm, r.BasicAuth))

	tokenResp, err := r.HttpClient.Do(tokenReq)
	if err != nil {
		logger.Error(err, "failed to do")
		return nil, err
	}
	defer tokenResp.Body.Close()
	if !regClient.SuccessStatus(tokenResp.StatusCode) {
		err := regClient.HandleErrorResponse(tokenResp)
		return nil, err
	}

	decoder := json.NewDecoder(tokenResp.Body)
	token := &tokenResponse{}
	if err := decoder.Decode(token); err != nil {
		logger.Error(err, "failed to decode token")
		return nil, err
	}

	r.Token = &Token{
		Type:  TokenTypeBearer,
		Value: token.Token,
	}

	return r.Token, nil
}

type tokenResponse struct {
	Token        string    `json:"token"`
	AccessToken  string    `json:"access_token"`
	ExpiresIn    int       `json:"expires_in"`
	IssuedAt     time.Time `json:"issued_at"`
	RefreshToken string    `json:"refresh_token"`
}

func (r *RegistryAPI) Catalog() *Repositories {
	logger.Info("call", "api", r.Scheme+r.ServerURL+"/v2/_catalog")
	req, err := http.NewRequest(http.MethodGet, r.Scheme+r.ServerURL+"/v2/_catalog", nil)
	if err != nil {
		logger.Error(err, "")
		return nil
	}

	if r.BasicAuth != "" {
		scope := "registry:catalog:*"
		token, err := r.GetToken(scope)
		if err != nil {
			logger.Error(err, "")
			return nil
		}

		req.Header.Add("Authorization", fmt.Sprintf("%s %s", token.Type, token.Value))
	}

	res, err := r.HttpClient.Do(req)
	if err != nil {
		logger.Error(err, "")
		return nil
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		logger.Error(err, "")
		return nil
	}
	logger.Info("contents", "repositories", string(body))

	rawRepos := &Repositories{}
	repos := &Repositories{}

	if err := json.Unmarshal(body, rawRepos); err != nil {
		logger.Error(err, "failed to unmarshal registry's repository")
		return nil
	}

	for _, repo := range rawRepos.Repositories {
		tags := r.Tags(repo).Tags
		if len(tags) > 0 {
			repos.Repositories = append(repos.Repositories, repo)
		}
	}

	return repos
}

func (r *RegistryAPI) Tags(imageName string) *Repository {
	repo := &Repository{}
	logger.Info("call", "api", r.Scheme+r.ServerURL+"/v2/"+imageName+"/tags/list")
	req, err := http.NewRequest(http.MethodGet, r.Scheme+r.ServerURL+"/v2/"+imageName+"/tags/list", nil)
	if err != nil {
		logger.Error(err, "")
		return nil
	}
	if r.BasicAuth != "" {
		scope := strings.Join([]string{"repository", imageName, "pull"}, ":")
		token, err := r.GetToken(scope)
		if err != nil {
			logger.Error(err, "")
			return nil
		}
		req.Header.Add("Authorization", fmt.Sprintf("%s %s", token.Type, token.Value))
	}

	res, err := r.HttpClient.Do(req)
	if err != nil {
		logger.Error(err, "")
		return nil
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		logger.Error(err, "")
		return nil
	}
	logger.Info("contents", "tags", string(body))
	if err := json.Unmarshal(body, repo); err != nil {
		logger.Error(err, "")
		return nil
	}

	return repo
}
