package auth

import (
	"fmt"
	"k8s.io/api/core/v1"
)

type SecretAuthProvider struct {
	Secret *corev1.Secret
	credit *DockerLoginCredential
}

type DockerLoginCredential struct {
	Auth     	string `json:"auth"`
	Email 		string `json:"email"`
	Username 	string `json:"username"`
	Password 	string `json:"password"`
}

type RegistryURL string

type DockerConfigJson struct {
	Auths  map[RegistryURL]DockerLoginCredential `json:"auths"`
}

func NewSecretAuth(secret v1.Secret) *SecretAuthProvider {
	instance := SecretAuthProvider{secret}

	err := instance.init()
	if err != nil {
		fmt.Println("Failed to initialize secret provider")
	}

	return &instance
}

func (p *SecretAuthProvider) init() error {
	
	switch p.secret.Type {
	case v1.SecretTypeDockerConfigJson:
		p.user, p.password, err := p.parseDocerConfigJson()
		if err != nil {
			return err
		}
	// case v1.SecretTypeOpaque:
	// 	p.user, p.password, err := p.parseOpaque()
	default:
		return error.New("Unsupported Secret type.")
	} 

	return nil
}

func (p *SecretAuthProvider) getID() (string, error) {
	if len(p.credit.Username) < 1 {
		return nil, error.New("User ID is empty")
	}
	return p.credit.Username, nil
}

func (p *SecretAuthProvider) getPassword() (string, error) {
	if len(p.credit.Password) < 1 {
		return nil, error.New("Password is empty")
	}
	return p.credit.Password, nil
}

func (p *SecretAuthProvider) getServerAddress() (string, error) {
	if len(p.credit.Password) < 1 {
		return nil, error.New("Password is empty")
	}
	return p.credit.Password, nil
}

func (p *SecretAuthProvider) parseDocerConfigJson() (id, password string, error) {
	
	data, ok := p.Secret.Data[v1.DockerConfigJsonKey]
	if !ok {
		return nil, error.New("Invalid DockerConfigSecret")
	}

	auths := make(map[RegistryURL]DockerLoginCredential)
	if err := json.Unmarshal(data, auths); err != nil {
		return nil, error.New()
	}

	if len(auths) > 1 {
		return nil, error.New("Too many auth provided. Provide just one.")
	}

	for registry, login := range auths {
		p.credit = login
	}
}

// func (p *SecretAuthProvider) parseOpaqueSecret() (id, password string, error) {

// }