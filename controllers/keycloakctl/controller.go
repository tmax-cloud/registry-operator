package keycloakctl

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"

	gocloak "github.com/Nerzal/gocloak/v7"
	"github.com/go-logr/logr"
	"github.com/operator-framework/operator-lib/status"

	regv1 "github.com/tmax-cloud/registry-operator/api/v1"
	cmhttp "github.com/tmax-cloud/registry-operator/internal/common/http"
	"github.com/tmax-cloud/registry-operator/internal/utils"
	corev1 "k8s.io/api/core/v1"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var (
	KeycloakServer = os.Getenv("KEYCLOAK_SERVICE")
	keycloakUser   = os.Getenv("KEYCLOAK_USERNAME")
	keycloakPwd    = os.Getenv("KEYCLOAK_PASSWORD")
)

const (
	rootCAName = regv1.K8sPrefix + regv1.K8sRegistryPrefix + "rootca"
)

// KeycloakController is ...
type KeycloakController struct {
	name       string
	client     gocloak.GoCloak
	logger     logr.Logger
	token      string
	httpClient *cmhttp.HttpClient
}

func NewKeycloakController(namespace, name string) *KeycloakController {
	client := gocloak.NewClient(KeycloakServer)
	restyClient := client.RestyClient()
	restyClient.SetDebug(true)
	// TODO: 인증서 추가할 것
	restyClient.SetTLSClientConfig(&tls.Config{
		InsecureSkipVerify: true,
	})
	logger := logf.Log.WithName("keycloak controller").WithValues("namespace", namespace, "registry name", name)

	// login admin
	token, err := client.LoginAdmin(context.Background(), keycloakUser, keycloakPwd, "master")
	if err != nil {
		logger.Error(err, "Couldn't get access token from keycloak")
		return nil
	}

	return &KeycloakController{
		name:       fmt.Sprintf("%s-%s", namespace, name),
		client:     client,
		logger:     logger,
		token:      token.AccessToken,
		httpClient: cmhttp.NewHTTPClient(KeycloakServer, keycloakUser, keycloakPwd),
	}
}

func (c *KeycloakController) GetRealmName() string {
	return c.name
}

func (c *KeycloakController) GetDockerV2ClientName() string {
	return c.name + "-docker-client"
}

func (c *KeycloakController) GetAdminToken() string {
	// login admin
	token, err := c.client.LoginAdmin(context.Background(), keycloakUser, keycloakPwd, "master")
	if err != nil {
		c.logger.Error(err, "Couldn't get access token from keycloak")
		return ""
	}

	return token.AccessToken
}

// CreateRealm is ...
func (c *KeycloakController) CreateRealm(reg, patchReg *regv1.Registry) error {
	var err error = nil
	condition := &status.Condition{
		Status: corev1.ConditionFalse,
		Type:   regv1.ConditionTypeKeycloakRealm,
	}

	defer utils.SetCondition(err, patchReg, condition)

	if !c.isExistRealm(c.name) {
		// make new realm
		realmEnabled := true
		_, err = c.client.CreateRealm(context.Background(), c.token, gocloak.RealmRepresentation{
			ID:      &c.name,
			Realm:   &c.name,
			Enabled: &realmEnabled,
		})
		if err != nil {
			c.logger.Error(err, "Couldn't create a new Realm")
			condition.Message = err.Error()
			return err
		}

		// make docker client
		clientName := c.GetDockerV2ClientName()
		protocol := "docker-v2"
		_, err = c.client.CreateClient(context.Background(), c.token, c.name, gocloak.Client{
			ClientID: &clientName,
			Protocol: &protocol,
		})
		if err != nil {
			c.logger.Error(err, "Couldn't create docker client in realm "+c.name)
			condition.Message = err.Error()
			return err
		}
	}

	if !c.isExistCertificate() {
		if err := c.AddCertificate(); err != nil {
			c.logger.Error(err, "Couldn't create a certificate component")
			condition.Message = err.Error()
			return err
		}
	}

	if !c.isExistUser(reg.Spec.LoginId) {
		c.logger.Info("CreateUser", "username", reg.Spec.LoginId)
		if err := c.CreateUser(c.token, reg.Spec.LoginId, reg.Spec.LoginPassword); err != nil {
			return err
		}
	}

	condition.Status = corev1.ConditionTrue
	return nil
}

// DeleteRealm is ...
func (c *KeycloakController) DeleteRealm(namespace string, name string) error {
	if !c.isExistRealm(c.name) {
		return nil
	}

	// Delete realm
	if err := c.client.DeleteRealm(context.Background(), c.token, c.name); err != nil {
		c.logger.Error(err, "Couldn't delete the realm named "+c.name)
		return err
	}

	return nil
}

// CreateUser creates new user
func (c *KeycloakController) CreateUser(token, user, password string) error {
	enabled := true
	newUser := gocloak.User{Username: &user, Enabled: &enabled}
	userID, err := c.client.CreateUser(context.TODO(), token, c.GetRealmName(), newUser)
	if err != nil {
		c.logger.Error(err, "Couldn't create user: "+user)
		return err
	}

	if err := c.client.SetPassword(
		context.TODO(),
		token,
		userID,
		c.GetRealmName(),
		password,
		false,
	); err != nil {
		c.logger.Error(err, "Couldn't set password. username: "+user)
		return err
	}

	c.logger.Info(fmt.Sprintf("create user succeeded: %s", user))

	return nil
}

func (c *KeycloakController) AddCertificate() error {
	reqURL := c.componentURL()
	cacrt, cakey := cmhttp.CAData()
	cacrt = utils.RemovePemBlock(cacrt, "CERTIFICATE")

	privBlock, privRest := pem.Decode(cakey)
	if len(privRest) != 0 {
		fmt.Printf("Private key is not PEM format %s %s", "Rest", privRest)
		return fmt.Errorf("Private key is not PEM format %s %s", "Rest", privRest)
	}
	cakey = utils.RemovePemBlock(cakey, privBlock.Type)

	component := Component{
		Name:         rootCAName,
		ProviderID:   "rsa",
		ProviderType: "org.keycloak.keys.KeyProvider",
		ParentID:     c.GetRealmName(),
		ComponentConfig: &ComponentConfig{
			Priority:    []string{"500"},
			Enabled:     []string{"true"},
			Active:      []string{"true"},
			Algorithm:   []string{"RS256"},
			PrivateKey:  []string{string(cakey)},
			Certificate: []string{string(cacrt)},
		},
	}

	body, err := json.Marshal(component)
	if err != nil {
		return err
	}

	c.logger.Info("call", "api", reqURL)
	c.logger.Info("call", "body", string(body))
	req, err := http.NewRequest(http.MethodPost, reqURL, bytes.NewBuffer(body))
	if err != nil {
		c.logger.Error(err, "")
		return err
	}

	req.Header.Set("Content-type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.token)

	res, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Error(err, "")
		return err
	}

	resBody, err := ioutil.ReadAll(res.Body)
	if err != nil {
		c.logger.Error(err, "")
		return err
	}
	c.logger.Info("add certificate success", "response", string(resBody))

	return nil
}

func (c *KeycloakController) componentURL() string {
	return KeycloakServer + "/" + path.Join("auth", keycloakUser, "realms", c.GetRealmName(), "components")
}

func (c *KeycloakController) isExistRealm(name string) bool {
	if _, err := c.client.GetRealm(context.Background(), c.token, name); err != nil {
		return false
	}

	return true
}

func (c *KeycloakController) isExistUser(username string) bool {
	users, err := c.client.GetUsers(
		context.TODO(),
		c.token,
		c.GetRealmName(),
		gocloak.GetUsersParams{
			Username: gocloak.StringP(username),
		},
	)

	if len(users) == 0 || err != nil {
		return false
	}

	return true
}

func (c *KeycloakController) isExistCertificate() bool {
	reqURL := c.componentURL()
	parent := []string{c.GetRealmName()}
	keyType := []string{"org.keycloak.keys.KeyProvider"}
	params := map[string][]string{"parent": parent, "type": keyType}
	reqURL = utils.AddQueryParams(reqURL, params)

	c.logger.Info("call", "api", reqURL)
	req, err := http.NewRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		c.logger.Error(err, "")
		return false
	}

	req.Header.Set("Authorization", "Bearer "+c.token)

	res, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Error(err, "")
		return false
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		c.logger.Error(err, "")
		return false
	}
	components := Components{}
	if err := json.Unmarshal(body, &components); err != nil {
		c.logger.Info("contents", "components", string(body))
		return false
	}

	for _, comp := range components {
		if comp.Name == rootCAName {
			return true
		}
	}

	return false
}
