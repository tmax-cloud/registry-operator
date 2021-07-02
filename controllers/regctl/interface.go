package regctl

import (
	"github.com/operator-framework/operator-lib/status"
	regv1 "github.com/tmax-cloud/registry-operator/api/v1"
)

type ResourceController interface {
	ReconcileByConditionStatus(*regv1.Registry) error
	Require(status.ConditionType) ResourceController
}
