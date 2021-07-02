package regctl

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	regv1 "github.com/tmax-cloud/registry-operator/api/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/operator-framework/operator-lib/status"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// RegistryConfigMap contains things to handle deployment resource
type RegistryConfigMap struct {
	c            client.Client
	manifest     func() (interface{}, error)
	cond         status.ConditionType
	requirements []status.ConditionType
	logger       logr.Logger
}

// NewRegistryConfigMap creates new registry configmap controller
func NewRegistryConfigMap(client client.Client, manifest func() (interface{}, error), cond status.ConditionType, logger logr.Logger) *RegistryConfigMap {
	return &RegistryConfigMap{
		c:        client,
		manifest: manifest,
		cond:     cond,
		logger:   logger.WithName("Configmap"),
	}
}

func (r *RegistryConfigMap) ReconcileByConditionStatus(reg *regv1.Registry) error {
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
	manifest := m.(corev1.ConfigMap)
	cm := &corev1.ConfigMap{}
	if err = r.c.Get(ctx, types.NamespacedName{Name: manifest.Name, Namespace: manifest.Namespace}, cm); err != nil {
		if errors.IsNotFound(err) {
			if err = r.c.Create(ctx, &manifest); err != nil {
				return err
			}
			return nil
		}
		return err
	}

	if _, exist := cm.Data["config.yml"]; !exist {
		err = regv1.MakeRegistryError("NotReady")
		return err
	}

	reg.Status.Conditions.SetCondition(
		status.Condition{
			Type:    r.cond,
			Status:  corev1.ConditionTrue,
			Message: "Success",
		})
	return nil
}

func (r *RegistryConfigMap) Require(cond status.ConditionType) ResourceController {
	r.requirements = append(r.requirements, cond)
	return r
}
