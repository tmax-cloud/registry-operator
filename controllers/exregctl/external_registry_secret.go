package exregctl

import (
	"context"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"

	"github.com/operator-framework/operator-lib/status"
	regv1 "github.com/tmax-cloud/registry-operator/api/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// LoginSecret ...
type LoginSecret struct {
	c            client.Client
	manifest     func() (interface{}, error)
	cond         status.ConditionType
	requirements []status.ConditionType
	logger       logr.Logger
}

func NewLoginSecret(client client.Client, manifest func() (interface{}, error), cond status.ConditionType, logger logr.Logger) *LoginSecret {
	return &LoginSecret{
		c:        client,
		manifest: manifest,
		cond:     cond,
		logger:   logger.WithName("secret"),
	}
}

func (r *LoginSecret) ReconcileByConditionStatus(reg *regv1.ExternalRegistry) (bool, error) {
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

	//reg.Spec.LoginID = ""
	//reg.Spec.LoginPassword = ""
	reg.Status.LoginSecret = secret.Name
	if reg.Spec.RegistryType == regv1.RegistryTypeDockerHub {
		reg.Spec.RegistryURL = "https://registry-1.docker.io"
	}

	reg.Status.Conditions.SetCondition(
		status.Condition{
			Type:    r.cond,
			Status:  corev1.ConditionTrue,
			Message: "Success",
		})

	return false, nil
}

func (r *LoginSecret) Require(cond status.ConditionType) ResourceController {
	r.requirements = append(r.requirements, cond)
	return r
}
