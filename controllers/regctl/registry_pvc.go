package regctl

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	"github.com/operator-framework/operator-lib/status"
	regv1 "github.com/tmax-cloud/registry-operator/api/v1"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// RegistryPVC things to handle pvc resource
type RegistryPVC struct {
	c            client.Client
	cond         status.ConditionType
	requirements []status.ConditionType
	manifest     func() (interface{}, error)
	logger       logr.Logger
}

// NewRegistryPVC creates new registry pvc controller
func NewRegistryPVC(client client.Client, manifest func() (interface{}, error), cond status.ConditionType, logger logr.Logger) *RegistryPVC {
	return &RegistryPVC{
		c:        client,
		manifest: manifest,
		cond:     cond,
		logger:   logger.WithName("PVC"),
	}
}

func (r *RegistryPVC) ReconcileByConditionStatus(reg *regv1.Registry) (bool, error) {
	var err error
	defer func() {
		if err != nil {
			reg.Status.Conditions.SetCondition(
				status.Condition{
					Type:    r.cond,
					Status:  corev1.ConditionFalse,
					Message: err.Error(),
				})
		}
	}()

	for _, dep := range r.requirements {
		if !reg.Status.Conditions.GetCondition(dep).IsTrue() {
			r.logger.Info(string(r.cond) + " needs " + string(dep))
			return true, nil
		}
	}

	ctx := context.TODO()
	m, err := r.manifest()
	if err != nil {
		return false, err
	}
	manifest := m.(*corev1.PersistentVolumeClaim)
	pvc := &corev1.PersistentVolumeClaim{}
	if err = r.c.Get(ctx, types.NamespacedName{Name: manifest.Name, Namespace: manifest.Namespace}, pvc); err != nil {
		if errors.IsNotFound(err) {
			r.logger.Info("not found. create new one.")
			if err = r.c.Create(ctx, manifest); err != nil {
				return true, err
			}
		}
		return true, err
	}

	if string(pvc.Status.Phase) == "pending" {
		return true, fmt.Errorf("pvc is pending")
	}

	reg.Status.Capacity = pvc.Status.Capacity.Storage().String()
	reg.Status.Conditions.SetCondition(
		status.Condition{
			Type:    r.cond,
			Status:  corev1.ConditionTrue,
			Message: "Success",
		})
	r.logger.Info("fine")
	return false, nil
}

func (r *RegistryPVC) Require(cond status.ConditionType) ResourceController {
	r.requirements = append(r.requirements, cond)
	return r
}
