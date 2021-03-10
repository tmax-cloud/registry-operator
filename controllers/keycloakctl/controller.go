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
	"path"

	gocloak "github.com/Nerzal/gocloak/v7"
	"github.com/go-logr/logr"
	"github.com/operator-framework/operator-lib/status"

	regv1 "github.com/tmax-cloud/registry-operator/api/v1"
	"github.com/tmax-cloud/registry-operator/internal/common/certs"
	"github.com/tmax-cloud/registry-operator/internal/common/config"
	cmhttp "github.com/tmax-cloud/registry-operator/internal/common/http"
	"github.com/tmax-cloud/registry-operator/internal/utils"
	corev1 "k8s.io/api/core/v1"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
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
	logger := logf.Log.WithName("keycloak controller").WithValues("namespace", namespace, "registry name", name)
	KeycloakServer := config.Config.GetString(config.ConfigKeycloakService)
	keycloakUser := config.Config.GetString("keycloak.username")
	keycloakPwd := config.Config.GetString("keycloak.password")
	client := gocloak.NewClient(KeycloakServer)
	restyClient := client.RestyClient()
	// TODO: 인증서 추가할 것
	restyClient.SetTLSClientConfig(&tls.Config{
		InsecureSkipVerify: true,
	})

	// set realm name
	var realm string
	clusterName := config.Config.GetString(config.ConfigClusterName)
	if clusterName == "" {
		realm = fmt.Sprintf("%s-%s", namespace, name)
	} else {
		realm = fmt.Sprintf("%s-%s-%s", clusterName, namespace, name)
	}

	// login admin
	token, err := client.LoginAdmin(context.Background(), keycloakUser, keycloakPwd, "master")
	if err != nil {
		logger.Error(err, "Couldn't get access token from keycloak")
		return nil
	}

	return &KeycloakController{
		name:       realm,
		client:     client,
		logger:     logger,
		token:      token.AccessToken,
		httpClient: cmhttp.NewHTTPClient(KeycloakServer, keycloakUser, keycloakPwd, nil, false),
	}
}

func (c *KeycloakController) GetRealmName() string {
	return c.name
}

func (c *KeycloakController) GetDockerV2ClientName() string {
	return c.name + "-docker-client"
}

func (c *KeycloakController) GetAdminToken() (string, error) {
	keycloakUser := config.Config.GetString("keycloak.username")
	keycloakPwd := config.Config.GetString("keycloak.password")

	// login admin
	token, err := c.client.LoginAdmin(context.Background(), keycloakUser, keycloakPwd, "master")
	if err != nil {
		c.logger.Error(err, "Couldn't get access token from keycloak")
		return "", err
	}
	c.token = token.AccessToken
	return token.AccessToken, nil
}

// CreateResources is ...
func (c *KeycloakController) CreateResources(reg, patchReg *regv1.Registry) error {
	var err error = nil
	condition := &status.Condition{
		Status: corev1.ConditionFalse,
		Type:   regv1.ConditionTypeKeycloakResources,
	}

	defer utils.SetCondition(err, patchReg, condition)

	if _, err := c.GetAdminToken(); err != nil {
		c.logger.Error(err, "Couldn't get access token from keycloak")
		return err
	}

	if !c.isExistRealm(c.name) {
		c.logger.Info(fmt.Sprintf("%s realm is not found in keystore", c.name))
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
	}

	if !c.isExistClient() {
		clientName := c.GetDockerV2ClientName()
		c.logger.Info(fmt.Sprintf("%s client is not found", clientName))

		// create docker-v2 client
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
		c.logger.Info(fmt.Sprintf("%s is not found in keystore", rootCAName))
		if err := c.AddCertificate(); err != nil {
			c.logger.Error(err, "Couldn't create a certificate component")
			condition.Message = err.Error()
			return err
		}
	}

	if !c.isExistUser(reg.Spec.LoginID) {
		c.logger.Info(fmt.Sprintf("%s user is not found in keystore", reg.Spec.LoginID))
		c.logger.Info("CreateUser", "username", reg.Spec.LoginID)
		if err := c.CreateUser(c.token, reg.Spec.LoginID, reg.Spec.LoginPassword); err != nil {
			return err
		}
	}

	if !c.isExistRealm(c.name) || !c.isExistClient() || !c.isExistCertificate() || !c.isExistUser(reg.Spec.LoginID) {
		return fmt.Errorf("failed to create realm/client/certificate/user")
	}

	condition.Status = corev1.ConditionTrue
	return nil
}

// DeleteRealm is ...
func (c *KeycloakController) DeleteRealm(namespace string, name string) error {
	if !c.isExistRealm(c.name) {
		return nil
	}
	if _, err := c.GetAdminToken(); err != nil {
		c.logger.Error(err, "Couldn't get access token from keycloak")
		return err
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
	caSecret, err := certs.GetSystemRootCASecret(nil)
	if err != nil {
		return err
	}
	cacrt, cakey := certs.CAData(caSecret)
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

	c.logger.Info("call", "method", http.MethodPost, "api", reqURL)
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
	KeycloakServer := config.Config.GetString(config.ConfigKeycloakService)
	keycloakUser := config.Config.GetString("keycloak.username")
	return KeycloakServer + "/" + path.Join("auth", keycloakUser, "realms", c.GetRealmName(), "components")
}

func (c *KeycloakController) isExistRealm(name string) bool {
	if _, err := c.client.GetRealm(context.Background(), c.token, name); err != nil {
		return false
	}

	return true
}

func (c *KeycloakController) isExistClient() bool {
	clientName := c.GetDockerV2ClientName()
	logger := c.logger.WithValues("realm", c.GetRealmName(), "clientName", clientName)
	params := gocloak.GetClientsParams{
		ClientID: &clientName,
	}
	clients, err := c.client.GetClients(context.Background(), c.token, c.GetRealmName(), params)
	if err != nil {
		logger.Error(err, "failed to get client")
		return false
	}

	clientID := ""
	for _, client := range clients {
		logger.Info("debug", "*client.ClientID", *client.ClientID, "ID", *client.ID)
		if *client.ClientID == clientName {
			clientID = *client.ID
			break
		}
	}

	if clientID == "" {
		return false
	}

	logger = logger.WithValues("client.ID", clientID)
	if _, err := c.client.GetClient(context.Background(), c.token, c.GetRealmName(), clientID); err != nil {
		logger.Error(err, "failed to get client")
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

	c.logger.Info("call", "method", http.MethodGet, "api", reqURL)
	req, err := http.NewRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		c.logger.Error(err, "")
		return false
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	// req.SetBasicAuth(c.httpClient.Login.Username, c.httpClient.Login.Password)

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
