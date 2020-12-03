package regctl

import (
	"context"
	"crypto/tls"

	gocloak "github.com/Nerzal/gocloak/v7"

	regv1 "github.com/tmax-cloud/registry-operator/api/v1"
	"github.com/tmax-cloud/registry-operator/internal/utils"
)

const (
	// TODO: hypercloud의 keycloak 서비스와 맞추기
	testKeycloakServer = "https://172.22.11.19:8443"
	testKeycloakUser   = "admin"
	testKeycloakPwd    = "admin"
)

// KeycloakController is ...
type KeycloakController struct {
	client gocloak.GoCloak
	logger *utils.RegistryLogger
}

func (c *KeycloakController) makeController(reg *regv1.Registry) *KeycloakController {
	client := gocloak.NewClient(testKeycloakServer)
	restyClient := client.RestyClient()
	restyClient.SetDebug(true)
	// TODO: 인증서 추가할 것
	restyClient.SetTLSClientConfig(&tls.Config{
		InsecureSkipVerify: true,
	})

	return &KeycloakController{
		client: client,
		logger: utils.NewRegistryLogger(*c, reg.Namespace, reg.Name+" registry's pod"),
	}
}

// CreateRealm is ...
func (c *KeycloakController) CreateRealm(reg *regv1.Registry, name string) error {
	// login admin
	token, err := c.client.LoginAdmin(context.Background(), testKeycloakUser, testKeycloakPwd, "master")
	if err != nil {
		c.logger.Error(err, "Couldn't get access token from keycloak")
	}

	// make new realm
	realmEnabled := true
	_, err = c.client.CreateRealm(context.Background(), token.AccessToken, gocloak.RealmRepresentation{
		Realm:   &name,
		Enabled: &realmEnabled,
	})
	if err != nil {
		c.logger.Error(err, "Couldn't create a new Realm")
	}

	// make docker client
	clientName := name + "-docker-client"
	protocol := "docker-v2"
	_, err = c.client.CreateClient(context.Background(), token.AccessToken, name, gocloak.Client{
		ClientID: &clientName,
		Protocol: &protocol,
	})
	if err != nil {
		c.logger.Error(err, "Couldn't create docker client in realm "+name)
	}

	return nil
}

// DeleteRealm is ...
func (c *KeycloakController) DeleteRealm(reg *regv1.Registry, name string) error {
	// login admin
	token, err := c.client.LoginAdmin(context.Background(), testKeycloakUser, testKeycloakPwd, "master")
	if err != nil {
		c.logger.Error(err, "Couldn't get access token from keycloak")
	}

	// Delete realm
	if err := c.client.DeleteRealm(context.Background(), token.AccessToken, name); err != nil {
		c.logger.Error(err, "Couldn't delete the realm named "+name)
	}

	return nil
}
