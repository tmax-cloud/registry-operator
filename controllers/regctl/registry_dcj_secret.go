package regctl

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	regv1 "github.com/tmax-cloud/registry-operator/api/v1"

	"github.com/operator-framework/operator-lib/status"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// RegistryDockerConfigSecret contains things to handle docker config json secret resource
type RegistryDockerConfigSecret struct {
	c            client.Client
	cond         status.ConditionType
	requirements []status.ConditionType
	manifest     func() (interface{}, error)
	logger       logr.Logger
}

// NewRegistryDCJSecret creates new registry docker config json secret controller
// deps: service
func NewRegistryDCJSecret(client client.Client, manifest func() (interface{}, error), cond status.ConditionType, logger logr.Logger) *RegistryDockerConfigSecret {
	return &RegistryDockerConfigSecret{
		c:        client,
		manifest: manifest,
		cond:     cond,
		logger:   logger.WithName("DockerConfigSecret"),
	}
}

func (r *RegistryDockerConfigSecret) ReconcileByConditionStatus(reg *regv1.Registry) (bool, error) {
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
	manifest := m.(*corev1.Secret)
	secret := &corev1.Secret{}
	if err = r.c.Get(ctx, types.NamespacedName{Name: manifest.Name, Namespace: manifest.Namespace}, secret); err != nil {
		if errors.IsNotFound(err) {
			r.logger.Info("not found. create new one.")
			if err = r.c.Create(ctx, manifest); err != nil {
				return false, err
			}
			return false, nil
		}
		return false, err
	}

	if _, ok := secret.Data[corev1.DockerConfigJsonKey]; !ok {
		err = fmt.Errorf("secret has no .dockerconfigjson field")
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

func (r *RegistryDockerConfigSecret) Require(cond status.ConditionType) ResourceController {
	r.requirements = append(r.requirements, cond)
	return r
}
