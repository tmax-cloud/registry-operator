package exregctl

import (
	"context"

	"github.com/operator-framework/operator-lib/status"
	regv1 "github.com/tmax-cloud/registry-operator/api/v1"
	"github.com/tmax-cloud/registry-operator/internal/schemes"
	"github.com/tmax-cloud/registry-operator/internal/utils"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type RegistryJob struct {
	job    *regv1.RegistryJob
	logger *utils.RegistryLogger
}

// Handle is to create external registry job.
func (r *RegistryJob) Handle(c client.Client, exreg *regv1.ExternalRegistry, patchExreg *regv1.ExternalRegistry, scheme *runtime.Scheme) error {
	if exreg.Status.Conditions.GetCondition(regv1.ConditionTypeExRegistryInitialized).Status == corev1.ConditionTrue {
		return nil
	}

	if err := r.get(c, exreg); err != nil {
		if errors.IsNotFound(err) {
			if err := r.create(c, exreg, patchExreg, scheme); err != nil {
				r.logger.Error(err, "create external registry job error")
				return err
			}
		} else {
			r.logger.Error(err, "external registry job error")
			return err
		}
	}

	return nil
}

// Ready is to check if the external registry job is ready
func (r *RegistryJob) Ready(c client.Client, exreg *regv1.ExternalRegistry, patchExreg *regv1.ExternalRegistry, useGet bool) error {
	if exreg.Status.Conditions.GetCondition(regv1.ConditionTypeExRegistryInitialized).Status == corev1.ConditionTrue {
		return nil
	}

	var err error = nil
	condition := &status.Condition{
		Status: corev1.ConditionFalse,
		Type:   regv1.ConditionTypeExRegistryInitialized,
	}

	defer utils.SetCondition(err, patchExreg, condition)

	if useGet {
		if err = r.get(c, exreg); err != nil {
			r.logger.Error(err, "get external registry job error")
			return err
		}
	}

	r.logger.Info("Ready")
	condition.Status = corev1.ConditionTrue
	return nil
}

func (r *RegistryJob) create(c client.Client, exreg *regv1.ExternalRegistry, patchExreg *regv1.ExternalRegistry, scheme *runtime.Scheme) error {
	if err := controllerutil.SetControllerReference(exreg, r.job, scheme); err != nil {
		r.logger.Error(err, "SetOwnerReference Failed")
		return err
	}

	r.logger.Info("Create external registry job")
	if err := c.Create(context.TODO(), r.job); err != nil {
		r.logger.Error(err, "Creating external registry job is failed.")
		return err
	}

	return nil
}

func (r *RegistryJob) get(c client.Client, exreg *regv1.ExternalRegistry) error {
	r.job = schemes.ExternalRegistryJob(exreg)
	r.logger = utils.NewRegistryLogger(*r, r.job.Namespace, r.job.Name)

	req := types.NamespacedName{Name: r.job.Name, Namespace: r.job.Namespace}
	err := c.Get(context.TODO(), req, r.job)
	if err != nil {
		r.logger.Error(err, "Get external registry job is failed")
		return err
	}

	return nil
}

func (r *RegistryJob) compare(reg *regv1.ExternalRegistry) []utils.Diff {
	diff := []utils.Diff{}

	// TODO

	return diff
}

func (r *RegistryJob) patch(c client.Client, exreg *regv1.ExternalRegistry, patchExreg *regv1.ExternalRegistry, diff []utils.Diff) error {

	return nil
}

func (r *RegistryJob) delete(c client.Client, patchExreg *regv1.ExternalRegistry) error {
	if err := c.Delete(context.TODO(), r.job); err != nil {
		r.logger.Error(err, "Unknown error delete deployment")
		return err
	}

	return nil
}
