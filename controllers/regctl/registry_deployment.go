package regctl

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	"github.com/operator-framework/operator-lib/status"
	regv1 "github.com/tmax-cloud/registry-operator/api/v1"
	appsv1 "k8s.io/api/apps/v1"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// RegistryDeployment contains things to handle deployment resource
type RegistryDeployment struct {
	c            client.Client
	cond         status.ConditionType
	requirements []status.ConditionType
	manifest     func() (interface{}, error)
	logger       logr.Logger
}

// NewRegistryDeployment creates new registry deployment controller
// deps: pvc, svc, cm
func NewRegistryDeployment(client client.Client, manifest func() (interface{}, error), cond status.ConditionType, logger logr.Logger) *RegistryDeployment {
	return &RegistryDeployment{
		c:        client,
		manifest: manifest,
		cond:     cond,
		logger:   logger.WithName("Deployment"),
	}
}

func (r *RegistryDeployment) ReconcileByConditionStatus(reg *regv1.Registry) error {
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
			err = fmt.Errorf("required conditions is not ready")
			return err
		}
	}

	ctx := context.TODO()
	m, err := r.manifest()
	if err != nil {
		return err
	}
	manifest := m.(appsv1.Deployment)
	deployment := &appsv1.Deployment{}
	if err = r.c.Get(ctx, types.NamespacedName{Name: manifest.Name, Namespace: manifest.Namespace}, deployment); err != nil {
		if errors.IsNotFound(err) {
			if err = r.c.Create(ctx, &manifest); err != nil {
				return err
			}
			return nil
		}
		return err
	}

	reg.Status.ReadOnly = reg.Spec.ReadOnly
	reg.Status.Conditions.SetCondition(
		status.Condition{
			Type:    r.cond,
			Status:  corev1.ConditionTrue,
			Message: "Success",
		})
	return nil
}

func (r *RegistryDeployment) Require(cond status.ConditionType) ResourceController {
	r.requirements = append(r.requirements, cond)
	return r
}
