package regctl

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	"github.com/operator-framework/operator-lib/status"
	regv1 "github.com/tmax-cloud/registry-operator/api/v1"
	"k8s.io/apimachinery/pkg/api/errors"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// RegistryTlsSecret contains things to handle tls and opaque secret resource
type RegistryTlsSecret struct {
	c            client.Client
	manifest     func() (interface{}, error)
	cond         status.ConditionType
	requirements []status.ConditionType
	logger       logr.Logger
}

func NewRegistryTlsCertSecret(client client.Client, manifest func() (interface{}, error), cond status.ConditionType, logger logr.Logger) *RegistryTlsSecret {
	return &RegistryTlsSecret{
		c:        client,
		manifest: manifest,
		cond:     cond,
		logger:   logger.WithName("TLSSecret"),
	}
}

func (r *RegistryTlsSecret) ReconcileByConditionStatus(reg *regv1.Registry) (bool, error) {
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

	if _, ok := secret.Data[corev1.TLSCertKey]; !ok {
		err = fmt.Errorf("secret has no tls.crt field")
		return false, err
	}

	if _, ok := secret.Data[corev1.TLSPrivateKeyKey]; !ok {
		err = fmt.Errorf("secret has no tls.key field")
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

func (r *RegistryTlsSecret) Require(cond status.ConditionType) ResourceController {
	r.requirements = append(r.requirements, cond)
	return r
}
