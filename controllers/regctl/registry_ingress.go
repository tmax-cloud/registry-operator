package regctl

import (
	"context"
	"github.com/go-logr/logr"
	regv1 "github.com/tmax-cloud/registry-operator/api/v1"

	"github.com/operator-framework/operator-lib/status"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// RegistryIngress contains things to handle ingress resource
type RegistryIngress struct {
	c            client.Client
	manifest     func() (interface{}, error)
	cond         status.ConditionType
	requirements []status.ConditionType
	logger       logr.Logger
}

// NewRegistryIngress creates new registry ingress controller
// deps: cert
func NewRegistryIngress(client client.Client, manifest func() (interface{}, error), cond status.ConditionType, logger logr.Logger) *RegistryIngress {
	return &RegistryIngress{
		c:        client,
		manifest: manifest,
		cond:     cond,
		logger:   logger.WithName("Ingress"),
	}
}

func (r *RegistryIngress) ReconcileByConditionStatus(reg *regv1.Registry) (bool, error) {
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
	manifest := m.(*v1beta1.Ingress)
	ingress := &v1beta1.Ingress{}
	if err = r.c.Get(ctx, types.NamespacedName{Name: manifest.Name, Namespace: manifest.Namespace}, ingress); err != nil {
		if errors.IsNotFound(err) {
			r.logger.Info("not found. create new one.")
			if err = r.c.Create(ctx, manifest); err != nil {
				return false, err
			}
			return false, nil
		}
		return false, err
	}

	if len(ingress.Spec.TLS) > 0 {
		for _, host := range ingress.Spec.TLS[0].Hosts {
			reg.Status.ServerURL = "https://" + host
		}
	}

	reg.Status.Conditions.SetCondition(
		status.Condition{
			Type:    r.cond,
			Status:  corev1.ConditionTrue,
			Message: "Success",
		})

	return false, nil
}

func (r *RegistryIngress) Require(cond status.ConditionType) ResourceController {
	r.requirements = append(r.requirements, cond)
	return r
}
