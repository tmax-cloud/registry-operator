package regctl

import (
	"time"

	"github.com/tmax-cloud/registry-operator/internal/utils"
	"k8s.io/apimachinery/pkg/runtime"

	regv1 "github.com/tmax-cloud/registry-operator/api/v1"
)

// RegistrySubresource is an interface to handle resigstry subreousrces
type RegistrySubresource interface {
	CreateIfNotExist(*regv1.Registry, *regv1.Registry, *runtime.Scheme) error
	IsReady(*regv1.Registry, *regv1.Registry, bool) error

	create(*regv1.Registry, *regv1.Registry, *runtime.Scheme) error
	get(*regv1.Registry) error
	patch(*regv1.Registry, *regv1.Registry, []utils.Diff) error
	delete(*regv1.Registry) error
	compare(*regv1.Registry) []utils.Diff
}

// Dependent checks dependent subresource is OK
type Dependent interface {
	// IsSuccessfullyCompleted returns true if subresource is successfully performed to meet the conditions
	IsSuccessfullyCompleted(reg *regv1.Registry) bool
	// ModifiedTime returns the modified time of the subresource condition
	ModifiedTime(reg *regv1.Registry) []time.Time
	// Condition returns dependent subresource's condition type
	Condition() string
}
