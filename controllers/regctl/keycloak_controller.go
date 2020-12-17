package regctl

import (
	"context"
	"crypto/tls"
	"fmt"
	"os"

	gocloak "github.com/Nerzal/gocloak/v7"
	"github.com/go-logr/logr"
	"github.com/operator-framework/operator-lib/status"

	regv1 "github.com/tmax-cloud/registry-operator/api/v1"
	"github.com/tmax-cloud/registry-operator/internal/utils"
	corev1 "k8s.io/api/core/v1"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var (
	keycloakServer = os.Getenv("KEYCLOAK_SERVICE")
	keycloakUser   = os.Getenv("KEYCLOAK_USERNAME")
	keycloakPwd    = os.Getenv("KEYCLOAK_PASSWORD")
)

// KeycloakController is ...
type KeycloakController struct {
	name   string
	client gocloak.GoCloak
	logger logr.Logger
}

func NewKeycloakController(namespace, name string) *KeycloakController {
	client := gocloak.NewClient(keycloakServer)
	restyClient := client.RestyClient()
	restyClient.SetDebug(true)
	// TODO: 인증서 추가할 것
	restyClient.SetTLSClientConfig(&tls.Config{
		InsecureSkipVerify: true,
	})
	logger := logf.Log.WithName("keycloak controller").WithValues("namespace", namespace, "registry name", name)
	return &KeycloakController{
		name:   fmt.Sprintf("%s-%s", namespace, name),
		client: client,
		logger: logger,
	}
}

func (c *KeycloakController) GetRealmName() string {
	return c.name
}

func (c *KeycloakController) GetDockerV2ClientName() string {
	return c.name + "-docker-client"
}

// CreateRealm is ...
func (c *KeycloakController) CreateRealm(namespace string, name string, patchReg *regv1.Registry) error {
	var err error = nil
	condition := &status.Condition{
		Status: corev1.ConditionFalse,
		Type:   regv1.ConditionTypeKeycloakRealm,
	}

	defer utils.SetCondition(err, patchReg, condition)

	// login admin
	token, err := c.client.LoginAdmin(context.Background(), keycloakUser, keycloakPwd, "master")
	if err != nil {
		c.logger.Error(err, "Couldn't get access token from keycloak")
		condition.Message = err.Error()
		return err
	}

	if c.isExistRealm(token.AccessToken, c.name) {
		condition.Status = corev1.ConditionTrue
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
		return err
	}

	// make docker client
	clientName := c.GetDockerV2ClientName()
	protocol := "docker-v2"
	_, err = c.client.CreateClient(context.Background(), token.AccessToken, c.name, gocloak.Client{
		ClientID: &clientName,
		Protocol: &protocol,
	})
	if err != nil {
		c.logger.Error(err, "Couldn't create docker client in realm "+c.name)
		condition.Message = err.Error()
		return err
	}

	condition.Status = corev1.ConditionTrue
	return nil
}

// DeleteRealm is ...
func (c *KeycloakController) DeleteRealm(namespace string, name string) error {
	// login admin
	token, err := c.client.LoginAdmin(context.Background(), keycloakUser, keycloakPwd, "master")
	if err != nil {
		c.logger.Error(err, "Couldn't get access token from keycloak")
		return err
	}

	if !c.isExistRealm(token.AccessToken, c.name) {
		return nil
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
