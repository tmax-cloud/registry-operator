package schemes

import (
	"encoding/json"
	"errors"

	regv1 "github.com/tmax-cloud/registry-operator/api/v1"
	"github.com/tmax-cloud/registry-operator/internal/utils"
	"github.com/tmax-cloud/registry-operator/pkg/image"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ExternalRegistryLoginSecret scheme
func ExternalRegistryLoginSecret(exreg *regv1.ExternalRegistry) (*corev1.Secret, error) {
	if !exregBodyCheckForLoginSecret(exreg) {
		return nil, errors.New("login info is empty")
	}

	registryURL := exreg.Spec.RegistryURL
	// set RegistryURL if RegistryType is DockerHub
	if exreg.Spec.RegistryType == regv1.RegistryTypeDockerHub {
		registryURL = image.DefaultServer
	}

	data := map[string][]byte{}
	config := DockerConfig{
		Auths: map[string]AuthValue{},
	}

	auth := AuthValue{utils.EncryptBasicAuth(exreg.Spec.LoginID, exreg.Spec.LoginPassword)}
	config.Auths[registryURL] = auth

	configBytes, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}
	data[corev1.DockerConfigJsonKey] = configBytes

	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      SubresourceName(exreg, SubTypeExternalRegistryLoginSecret),
			Namespace: exreg.Namespace,
			Labels: map[string]string{
				"secret": "exreg",
			},
		},
		Type: corev1.SecretTypeDockerConfigJson,
		Data: data,
	}, nil
}

func exregBodyCheckForLoginSecret(exreg *regv1.ExternalRegistry) bool {
	if exreg.Status.LoginSecret == "" &&
		exreg.Spec.LoginID == "" &&
		exreg.Spec.LoginPassword == "" {
		return false
	}
	return true
}
