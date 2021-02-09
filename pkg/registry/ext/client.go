package ext

import (
	regv1 "github.com/tmax-cloud/registry-operator/api/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var Logger = log.Log.WithName("ext-registry")

type Readable interface {
	ListRepositories() *regv1.APIRepositories
	ListTags(repository string) *regv1.APIRepository
}

type Synchronizable interface {
	Readable
	Synchronize() error
}
