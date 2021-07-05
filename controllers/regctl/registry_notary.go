package regctl

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

// RegistryNotary contains things to handle notary resource
type RegistryNotary struct {
	c            client.Client
	manifest     func() (interface{}, error)
	cond         status.ConditionType
	requirements []status.ConditionType
	logger       logr.Logger
}

// NewRegistryNotary creates new registry notary controller
func NewRegistryNotary(client client.Client, manifest func() (interface{}, error), cond status.ConditionType, logger logr.Logger) *RegistryNotary {

	return &RegistryNotary{
		c:        client,
		manifest: manifest,
		cond:     cond,
		logger:   logger.WithName("Notary"),
	}
}

func (r *RegistryNotary) ReconcileByConditionStatus(reg *regv1.Registry) (bool, error) {
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
	manifest := m.(*regv1.Notary)
	notary := &regv1.Notary{}
	if err = r.c.Get(ctx, types.NamespacedName{Name: manifest.Name, Namespace: manifest.Namespace}, notary); err != nil {
		if errors.IsNotFound(err) {
			r.logger.Info("not found. create new one.")
			if err = r.c.Create(ctx, manifest); err != nil {
				return true, err
			}
			return true, nil
		}
		return true, err
	}
	if notary.Status.NotaryURL == "" {
		err = regv1.MakeRegistryError("NotReady")
		return true, err
	}

	reg.Status.NotaryURL = notary.Status.NotaryURL
	reg.Status.Conditions.SetCondition(
		status.Condition{
			Type:    r.cond,
			Status:  corev1.ConditionTrue,
			Message: "Success",
		})

	return true, nil
}

func (r *RegistryNotary) Require(cond status.ConditionType) ResourceController {
	r.requirements = append(r.requirements, cond)
	return r
}
