package replicatectl

import (
	regv1 "github.com/tmax-cloud/registry-operator/api/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ImageReplicateSubresource interface {
	Handle(client.Client, *regv1.ImageReplicate, *regv1.ImageReplicate, *runtime.Scheme) error
	Ready(client.Client, *regv1.ImageReplicate, *regv1.ImageReplicate, bool) error
}
