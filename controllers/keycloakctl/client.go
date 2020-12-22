package keycloakctl

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"path"
	"strings"

	"github.com/go-logr/logr"
	cmhttp "github.com/tmax-cloud/registry-operator/internal/common/http"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

type KeycloakClient struct {
	realm   string
	service string
	*cmhttp.HttpClient
	logger logr.Logger
}

func NewKeycloakClient(username, password, realm, service string) *KeycloakClient {
	logger := logf.Log.WithName("keycloak controller")
	return &KeycloakClient{
		realm:      realm,
		service:    service,
		HttpClient: cmhttp.NewHTTPClient(KeycloakServer, username, password),
		logger:     logger,
	}
}

func (c *KeycloakClient) GetRealm() string {
	return c.realm
}

func (c *KeycloakClient) GetService() string {
	return c.service
}

func (c *KeycloakClient) GetToken(scopes []string) (string, error) {
	reqURL := c.tokenURL()
	service := []string{c.service}
	params := map[string][]string{"service": service}
	if len(scopes) > 0 {
		params["scope"] = scopes
	}

	reqURL = c.addQueryParams(reqURL, params)

	c.logger.Info("call", "api", reqURL)
	req, err := http.NewRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		c.logger.Error(err, "")
		return "", err
	}

	req.SetBasicAuth(c.Login.Username, c.Login.Password)

	res, err := c.Client.Do(req)
	if err != nil {
		c.logger.Error(err, "")
		return "", err
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		c.logger.Error(err, "")
		return "", err
	}

	token := &KeycloakTokenResponse{}
	if err := json.Unmarshal(body, token); err != nil {
		c.logger.Info("contents", "token", string(body))
		return "", err
	}

	c.logger.Info("token", "val", token.Token)

	return token.Token, nil
}

func (c *KeycloakClient) tokenURL() string {
	return c.URL + "/" + path.Join("auth", "realms", c.realm, "protocol", "docker-v2", "auth")
}

func (c *KeycloakClient) addQueryParams(url string, params map[string][]string) string {
	isFirst := false
	if !strings.Contains(url, "?") {
		url += "?"
		isFirst = true
	}
	for key, values := range params {
		for _, v := range values {
			if !isFirst {
				url += "&"
			}
			url += strings.Join([]string{key, v}, "=")
			isFirst = false
		}
	}

	return url
}

type KeycloakTokenResponse struct {
	Token      string `json:"token"`
	Expires_in int    `json:"expires_in"`
	Issued_at  string `json:"issued_at"`
}
