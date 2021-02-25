package factory

import (
	regv1 "github.com/tmax-cloud/registry-operator/api/v1"
	cmhttp "github.com/tmax-cloud/registry-operator/internal/common/http"
	"github.com/tmax-cloud/registry-operator/pkg/registry/base"
	harborv2 "github.com/tmax-cloud/registry-operator/pkg/registry/ext/harbor/v2"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NewSyncRegistryFactory returns SyncRegistryFactory
func NewSyncRegistryFactory(
	k8sClient client.Client,
	namespacedName types.NamespacedName,
	scheme *runtime.Scheme,
	httpClient *cmhttp.HttpClient,
) *SyncRegistryFactory {
	return &SyncRegistryFactory{
		Factory: base.Factory{
			K8sClient:      k8sClient,
			NamespacedName: namespacedName,
			Scheme:         scheme,
			HttpClient:     httpClient,
		},
	}
}

// SyncRegistryFactory creates synchronizable external registry
type SyncRegistryFactory struct {
	base.Factory
}

//
func (f *SyncRegistryFactory) Create(registryType regv1.RegistryType) base.Synchronizable {
	switch registryType {
	case regv1.RegistryTypeHarborV2:
		return harborv2.NewClient(f.K8sClient, f.NamespacedName, f.Scheme, f.HttpClient)
	}

	return nil
}
