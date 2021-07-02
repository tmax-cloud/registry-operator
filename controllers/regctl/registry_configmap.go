package regctl

import (
	"context"
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

func (r *RegistryConfigMap) ReconcileByConditionStatus(reg *regv1.Registry) (bool, error) {
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
	manifest := m.(*corev1.ConfigMap)
	cm := &corev1.ConfigMap{}
	if err = r.c.Get(ctx, types.NamespacedName{Name: manifest.Name, Namespace: manifest.Namespace}, cm); err != nil {
		if errors.IsNotFound(err) {
			r.logger.Info("not found. create new one.")
			if err = r.c.Create(ctx, manifest); err != nil {
				return false, err
			}
			return false, nil
		}
		return false, err
	}

	if _, exist := cm.Data["config.yml"]; !exist {
		err = regv1.MakeRegistryError("NotReady")
		return false, err
	}

	reg.Status.Conditions.SetCondition(
		status.Condition{
			Type:    r.cond,
			Status:  corev1.ConditionTrue,
			Message: "Success",
		})
	r.logger.Info("fine")
	return false, nil
}

func (r *RegistryConfigMap) Require(cond status.ConditionType) ResourceController {
	r.requirements = append(r.requirements, cond)
	return r
}
