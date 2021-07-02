package regctl

import (
	"github.com/operator-framework/operator-lib/status"
	regv1 "github.com/tmax-cloud/registry-operator/api/v1"
)

type ResourceController interface {
	//ReconcileByConditionStatus
	ReconcileByConditionStatus(*regv1.Registry) (bool, error)
	Require(status.ConditionType) ResourceController
}
