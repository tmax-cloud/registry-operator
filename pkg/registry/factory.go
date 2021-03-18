package registry

import (
	"context"
	"fmt"

	regv1 "github.com/tmax-cloud/registry-operator/api/v1"
	cmhttp "github.com/tmax-cloud/registry-operator/internal/common/http"
	"github.com/tmax-cloud/registry-operator/internal/schemes"
	"github.com/tmax-cloud/registry-operator/internal/utils"
	"github.com/tmax-cloud/registry-operator/pkg/image"
	"github.com/tmax-cloud/registry-operator/pkg/registry/base"
	extfactory "github.com/tmax-cloud/registry-operator/pkg/registry/ext/factory"
	intfactory "github.com/tmax-cloud/registry-operator/pkg/registry/inter/factory"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type RegistryFactory interface {
	Create(registryType regv1.RegistryType) base.Registry
}

func GetFactory(registryType regv1.RegistryType, f *base.Factory) RegistryFactory {
	switch registryType {
	case regv1.RegistryTypeHpcdRegistry:
		return intfactory.NewRegistryFactory(f.K8sClient, f.NamespacedName, f.Scheme, f.HttpClient)
	case regv1.RegistryTypeDockerHub, regv1.RegistryTypeDocker, regv1.RegistryTypeHarborV2:
		return extfactory.NewRegistryFactory(f.K8sClient, f.NamespacedName, f.Scheme, f.HttpClient)
	}

	return nil
}

// GetHTTPClient returns httpClient
func GetHTTPClient(client client.Client, image *regv1.ImageInfo) (*cmhttp.HttpClient, error) {
	registry := types.NamespacedName{Namespace: image.RegistryNamespace, Name: image.RegistryName}

	url, err := GetURL(client, registry, image.RegistryType)
	if err != nil {
		return nil, err
	}

	username, password := "", ""
	imagePullSecret, err := GetLoginSecret(client, registry, image.RegistryType)
	if err != nil {
		return nil, err
	}
	base.Logger.Info("get", "imagePullSecret", imagePullSecret, "namespace", registry.Namespace)
	if imagePullSecret != "" {
		basic, err := utils.GetBasicAuth(imagePullSecret, registry.Namespace, url)
		if err != nil {
			return nil, err
		}

		username, password = utils.DecodeBasicAuth(basic)
	}

	var ca []byte
	certificateSecret, err := GetCertSecret(client, registry, image.RegistryType)
	if err != nil {
		return nil, err
	}
	base.Logger.Info("get", "certificateSecret", certificateSecret, "namespace", registry.Namespace)
	if certificateSecret != "" {
		data, err := utils.GetCAData(certificateSecret, registry.Namespace)
		if err != nil {
			return nil, err
		}
		ca = data
	}

	// if image.RegistryType == regv1.RegistryTypeHpcdRegistry {
	// 	secret, err := certs.GetSystemKeycloakCert(client)
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// 	base.Logger.Info("get", "certificateSecret", secret.Name, "namespace", secret.Namespace)
	// 	if secret != nil {
	// 		data, err := utils.GetCAData(secret.Name, secret.Namespace)
	// 		if err != nil {
	// 			return nil, err
	// 		}
	// 		ca = append(ca, data...)
	// 	}
	// }
	return cmhttp.NewHTTPClient(
		url,
		username, password,
		ca,
		len(ca) == 0,
	), nil
}

func GetLoginSecret(client client.Client, registry types.NamespacedName, registryType regv1.RegistryType) (string, error) {
	switch registryType {
	case regv1.RegistryTypeHpcdRegistry:
		reg := &regv1.Registry{}
		if err := client.Get(context.TODO(), registry, reg); err != nil {
			return "", err
		}
		return schemes.SubresourceName(reg, schemes.SubTypeRegistryDCJSecret), nil

	case regv1.RegistryTypeDockerHub, regv1.RegistryTypeDocker, regv1.RegistryTypeHarborV2:
		exreg := &regv1.ExternalRegistry{}
		if err := client.Get(context.TODO(), registry, exreg); err != nil {
			return "", err
		}
		return exreg.Status.LoginSecret, nil
	}

	return "", nil
}

func GetCertSecret(client client.Client, registry types.NamespacedName, registryType regv1.RegistryType) (string, error) {
	switch registryType {
	case regv1.RegistryTypeHpcdRegistry:
		reg := &regv1.Registry{}
		if err := client.Get(context.TODO(), registry, reg); err != nil {
			return "", err
		}
		return schemes.SubresourceName(reg, schemes.SubTypeRegistryTLSSecret), nil

	case regv1.RegistryTypeDockerHub, regv1.RegistryTypeDocker, regv1.RegistryTypeHarborV2:
		exreg := &regv1.ExternalRegistry{}
		if err := client.Get(context.TODO(), registry, exreg); err != nil {
			return "", err
		}
		return exreg.Spec.CertificateSecret, nil
	}

	return "", nil
}

// GetURL returns registry url
func GetURL(client client.Client, registry types.NamespacedName, registryType regv1.RegistryType) (string, error) {
	switch registryType {
	case regv1.RegistryTypeHpcdRegistry:
		reg := &regv1.Registry{}
		if err := client.Get(context.TODO(), registry, reg); err != nil {
			return "", err
		}
		return reg.Status.ServerURL, nil

	case regv1.RegistryTypeDockerHub:
		return image.DefaultServer, nil

	case regv1.RegistryTypeDocker, regv1.RegistryTypeHarborV2:
		exreg := &regv1.ExternalRegistry{}
		if err := client.Get(context.TODO(), registry, exreg); err != nil {
			return "", err
		}
		return exreg.Spec.RegistryURL, nil
	}

	return "", fmt.Errorf("%s/%s(type:%s) registry url is not found", registry.Namespace, registry.Name, registryType)
}
