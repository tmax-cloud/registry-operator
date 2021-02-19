package base

import (
	regv1 "github.com/tmax-cloud/registry-operator/api/v1"
)

type Readable interface {
	ListRepositories() *regv1.APIRepositories
	ListTags(repository string) *regv1.APIRepository
}

type Synchronizable interface {
	Readable
	Synchronize() error
}
