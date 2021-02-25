package registry

import (
	"context"
	"fmt"

	regv1 "github.com/tmax-cloud/registry-operator/api/v1"
	cmhttp "github.com/tmax-cloud/registry-operator/internal/common/http"
	"github.com/tmax-cloud/registry-operator/internal/utils"
	"github.com/tmax-cloud/registry-operator/pkg/registry/base"
	"github.com/tmax-cloud/registry-operator/pkg/registry/inter"
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
	}

	return nil
}

// GetHTTPClient returns httpClient
func GetHTTPClient(url, namespace, imagePullSecret, certificateSecret string) *cmhttp.HttpClient {
	username, password := "", ""
	if imagePullSecret != "" {
		basic, err := utils.GetBasicAuth(imagePullSecret, namespace, url)
		if err != nil {
			inter.Logger.Error(err, "failed to get basic auth")
		}

		username, password = utils.DecodeBasicAuth(basic)
	}

	var ca []byte
	if certificateSecret != "" {
		data, err := utils.GetCAData(certificateSecret, namespace)
		if err != nil {
			inter.Logger.Error(err, "failed to get ca data")
		}
		ca = data
	}

	return cmhttp.NewHTTPClient(
		url,
		username, password,
		ca,
		len(ca) == 0,
	)
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
	}

	return "", fmt.Errorf("%s/%s(type:%s) registry url is not found", registry.Namespace, registry.Name, registryType)
}
