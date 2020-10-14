package regctl

import (
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"registry-operator/internal/utils"
	regv1 "registry-operator/pkg/apis/tmax/v1"
)

type RegistrySubresource interface {
	Handle(client.Client, *regv1.Registry, *regv1.Registry, *runtime.Scheme) error
	Ready(client.Client, *regv1.Registry, *regv1.Registry, bool) error

	create(client.Client, *regv1.Registry, *regv1.Registry, *runtime.Scheme) error
	get(client.Client, *regv1.Registry) error
	patch(client.Client, *regv1.Registry, *regv1.Registry, []utils.Diff) error
	delete(client.Client, *regv1.Registry) error
	compare(*regv1.Registry) []utils.Diff
}
