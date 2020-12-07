package regctl

import (
	"context"
	"crypto/tls"
	"fmt"

	gocloak "github.com/Nerzal/gocloak/v7"
	"github.com/operator-framework/operator-lib/status"

	regv1 "github.com/tmax-cloud/registry-operator/api/v1"
	"github.com/tmax-cloud/registry-operator/internal/utils"
	corev1 "k8s.io/api/core/v1"
)

const (
	// TODO: hypercloud의 keycloak 서비스와 맞추기
	testKeycloakServer = "https://172.22.11.19:8443"
	testKeycloakUser   = "admin"
	testKeycloakPwd    = "admin"
)

// KeycloakController is ...
type KeycloakController struct {
	name   string
	client gocloak.GoCloak
	logger *utils.RegistryLogger
}

func (c *KeycloakController) makeController(namespace string, name string) {
	client := gocloak.NewClient(testKeycloakServer)
	restyClient := client.RestyClient()
	restyClient.SetDebug(true)
	// TODO: 인증서 추가할 것
	restyClient.SetTLSClientConfig(&tls.Config{
		InsecureSkipVerify: true,
	})

	c.client = client
	c.logger = utils.NewRegistryLogger(*c, namespace, name+" registry's keycloak realm")
	c.name = fmt.Sprintf("%s-%s", namespace, name)
}

// CreateRealm is ...
func (c *KeycloakController) CreateRealm(reg *regv1.Registry) error {
	c.makeController(reg.Namespace, reg.Name)
	condition := status.Condition{
		Status: corev1.ConditionFalse,
		Type:   regv1.ConditionKeycloakRealm,
	}

	// login admin
	token, err := c.client.LoginAdmin(context.Background(), testKeycloakUser, testKeycloakPwd, "master")
	if err != nil {
		c.logger.Error(err, "Couldn't get access token from keycloak")
		condition.Message = err.Error()
		reg.Status.Conditions.SetCondition(condition)
		return err
	}

	if c.isExistRealm(token.AccessToken, c.name) {
		return nil
	}

	// make new realm
	realmEnabled := true
	_, err = c.client.CreateRealm(context.Background(), token.AccessToken, gocloak.RealmRepresentation{
		Realm:   &c.name,
		Enabled: &realmEnabled,
	})
	if err != nil {
		c.logger.Error(err, "Couldn't create a new Realm")
		condition.Message = err.Error()
		reg.Status.Conditions.SetCondition(condition)
		return err
	}

	// make docker client
	clientName := c.name + "-docker-client"
	protocol := "docker-v2"
	_, err = c.client.CreateClient(context.Background(), token.AccessToken, c.name, gocloak.Client{
		ClientID: &clientName,
		Protocol: &protocol,
	})
	if err != nil {
		c.logger.Error(err, "Couldn't create docker client in realm "+c.name)
		condition.Message = err.Error()
		reg.Status.Conditions.SetCondition(condition)
		return err
	}

	condition.Status = corev1.ConditionTrue
	reg.Status.Conditions.SetCondition(condition)
	return nil
}

// DeleteRealm is ...
func (c *KeycloakController) DeleteRealm(namespace string, name string) error {
	c.makeController(namespace, name)

	// login admin
	token, err := c.client.LoginAdmin(context.Background(), testKeycloakUser, testKeycloakPwd, "master")
	if err != nil {
		c.logger.Error(err, "Couldn't get access token from keycloak")
		return err
	}

	// Delete realm
	if err := c.client.DeleteRealm(context.Background(), token.AccessToken, c.name); err != nil {
		c.logger.Error(err, "Couldn't delete the realm named "+c.name)
		return err
	}

	return nil
}

func (c *KeycloakController) isExistRealm(token string, name string) bool {
	if _, err := c.client.GetRealm(context.Background(), token, name); err != nil {
		return false
	}

	return true
}
