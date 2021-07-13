package exregctl

import (
	"context"
	"github.com/go-logr/logr"

	"github.com/operator-framework/operator-lib/status"
	regv1 "github.com/tmax-cloud/registry-operator/api/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type RegistryCronJob struct {
	c            client.Client
	manifest     func() (interface{}, error)
	cond         status.ConditionType
	requirements []status.ConditionType
	logger       logr.Logger
}

func NewRegistryCronJob(client client.Client, manifest func() (interface{}, error), cond status.ConditionType, logger logr.Logger) *RegistryCronJob {
	return &RegistryCronJob{
		c:        client,
		manifest: manifest,
		cond:     cond,
		logger:   logger.WithName("cronjob"),
	}
}

func (r *RegistryCronJob) ReconcileByConditionStatus(reg *regv1.ExternalRegistry) (bool, error) {
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
	manifest := m.(*regv1.RegistryCronJob)
	cronjob := &regv1.RegistryCronJob{}
	if err = r.c.Get(ctx, types.NamespacedName{Name: manifest.Name, Namespace: manifest.Namespace}, cronjob); err != nil {
		if errors.IsNotFound(err) {
			r.logger.Info("not found. create new one.")
			if err = r.c.Create(ctx, manifest); err != nil {
				return false, err
			}
			return false, nil
		}
		return false, err
	}
	reg.Status.Conditions.SetCondition(
		status.Condition{
			Type:    r.cond,
			Status:  corev1.ConditionTrue,
			Message: "Success",
		})

	return false, nil
}

func (r *RegistryCronJob) Require(cond status.ConditionType) ResourceController {
	r.requirements = append(r.requirements, cond)
	return r
}
