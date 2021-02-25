package base

import (
	"github.com/tmax-cloud/registry-operator/pkg/image"
)

type Registry interface {
}

type Readable interface {
	ListRepositories() *image.APIRepositories
	ListTags(repository string) *image.APIRepository
}

type Synchronizable interface {
	Readable
	Synchronize() error
}

type Replicatable interface {
	GetManifest(image string) (*image.ImageManifest, error)
	PutManifest(image string, manifest *image.ImageManifest) error
	ExistBlob(repository, digest string) (bool, error)
	PullBlob(repository, digest string) (string, int64, error)
	PushBlob(repository, digest, blobPath string, size int64) error
}
