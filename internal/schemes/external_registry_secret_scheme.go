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
	registryURLs := []string{}

	// set RegistryURL if RegistryType is DockerHub
	if exreg.Spec.RegistryType == regv1.RegistryTypeDockerHub {
		registryURLs = append(registryURLs, image.LegacyV1Server+"/", image.LegacyV2Server+"/")
	}

	if len(registryURLs) == 0 && exreg.Spec.RegistryURL != "" {
		registryURLs = append(registryURLs, exreg.Spec.RegistryURL)
	}

	if len(registryURLs) == 0 {
		return nil, errors.New("registry url is empty")
	}

	data := map[string][]byte{}
	config := DockerConfig{
		Auths: map[string]AuthValue{},
	}

	auth := AuthValue{utils.EncryptBasicAuth(exreg.Spec.LoginID, exreg.Spec.LoginPassword)}
	for _, url := range registryURLs {
		config.Auths[url] = auth
	}

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
