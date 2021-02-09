package exregctl

import (
	regv1 "github.com/tmax-cloud/registry-operator/api/v1"
	"github.com/tmax-cloud/registry-operator/internal/utils"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ExternalRegistrySubresource interface {
	Handle(client.Client, *regv1.ExternalRegistry, *regv1.ExternalRegistry, *runtime.Scheme) error
	Ready(client.Client, *regv1.ExternalRegistry, *regv1.ExternalRegistry, bool) error

	create(client.Client, *regv1.ExternalRegistry, *regv1.ExternalRegistry, *runtime.Scheme) error
	get(client.Client, *regv1.ExternalRegistry) error
	delete(client.Client, *regv1.ExternalRegistry) error

	patch(client.Client, *regv1.ExternalRegistry, *regv1.ExternalRegistry, []utils.Diff) error
	compare(*regv1.ExternalRegistry) []utils.Diff
}
