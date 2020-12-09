package notaryctl

import (
	regv1 "github.com/tmax-cloud/registry-operator/api/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type NotarySubresource interface {
	Handle(client.Client, *regv1.Notary, *regv1.Notary, *runtime.Scheme) error
	Ready(client.Client, *regv1.Notary, *regv1.Notary, bool) error

	create(client.Client, *regv1.Notary, *regv1.Notary, *runtime.Scheme) error
	get(client.Client, *regv1.Notary) error
	delete(client.Client, *regv1.Notary) error
}
