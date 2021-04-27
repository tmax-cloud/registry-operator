package secrethelper

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
)

const (
	DockerConfigAuthKey     = "auth"
	DockerConfigUserKey     = "user"
	DockerConfigPasswordKey = "password"
)

type DockerConfigJson struct {
	Auths map[string]DockerLoginCredential `json:"auths"`
}

type DockerLoginCredential map[string]string

type ImagePullSecret struct {
	secret *corev1.Secret
	json   *DockerConfigJson
}

type LoginCredential struct {
	Auth     []byte
	Username string
	Password []byte
}

func NewImagePullSecret(secret *corev1.Secret) (*ImagePullSecret, error) {

	if secret.Type != corev1.SecretTypeDockerConfigJson {
		return nil, fmt.Errorf("Unsupported secret type")
	}

	imagePullSecretData, ok := secret.Data[corev1.DockerConfigJsonKey]
	if !ok {
		return nil, fmt.Errorf("Failed to get dockerconfig from ImagePullSecret")
	}

	var dockerConfigJson DockerConfigJson
	if err := json.Unmarshal(imagePullSecretData, &dockerConfigJson); err != nil {
		return nil, fmt.Errorf("Failed to unmarshal ImagePullSecret(%s)'s dockerconfig", secret.Name)
	}

	return &ImagePullSecret{
		secret: secret,
		json:   &dockerConfigJson,
	}, nil
}

func (s *ImagePullSecret) GetHostCredential(host string) (*LoginCredential, error) {

	loginAuth, ok := s.json.Auths[host]
	if !ok {
		return nil, fmt.Errorf("Secret(%s)'s dockerconfig host not found ", s.secret.Name)
	}

	basicAuth, isBasicPresent := loginAuth[DockerConfigAuthKey]
	if !isBasicPresent {
		return nil, fmt.Errorf("Not found key named 'auth' in dockerconfigjson(%s).", host)
	}

	username, isUserPresent := loginAuth[DockerConfigUserKey]
	password, isPasswordPresent := loginAuth[DockerConfigPasswordKey]
	if isUserPresent && isPasswordPresent {
		return &LoginCredential{
			Auth:     []byte(basicAuth),
			Username: username,
			Password: []byte(password),
		}, nil
	}

	decodedBasicAuth, err := base64.StdEncoding.DecodeString(basicAuth)
	if err != nil {
		return nil, fmt.Errorf("Failed to decode auth")
	}
	tokens := strings.Split(string(decodedBasicAuth), ":")

	return &LoginCredential{
		Auth:     []byte(basicAuth),
		Username: tokens[0],
		Password: []byte(tokens[1]),
	}, nil
}
