package regctl

import (
	"context"
	"hypercloud-operator-go/internal/schemes"
	"hypercloud-operator-go/internal/utils"

	regv1 "hypercloud-operator-go/pkg/apis/tmax/v1"

	"github.com/go-logr/logr"
	"github.com/operator-framework/operator-sdk/pkg/status"
	appsv1 "k8s.io/api/apps/v1"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type RegistryDeployment struct {
	deploy *appsv1.Deployment
	logger logr.Logger
}

func (r *RegistryDeployment) Create(c client.Client, reg *regv1.Registry, conditions *status.Conditions, scheme *runtime.Scheme, useGet bool) error {
	if r.deploy == nil || useGet {
		err := r.get(c, reg, conditions)
		if err != nil && !errors.IsNotFound(err) {
			r.logger.Error(err, "Deployment error")
			return err
		} else if err == nil {
			return err
		}
	}

	if err := controllerutil.SetControllerReference(reg, r.deploy, scheme); err != nil {
		r.logger.Error(err, "SetOwnerReference Failed")
		return err
	}

	r.logger.Info("Create registry deployment")
	err := c.Create(context.TODO(), r.deploy)
	if err != nil {
		condition := status.Condition{
			Status:  corev1.ConditionFalse,
			Type:    regv1.ConditionTypeDeployment,
			Message: err.Error(),
		}

		conditions.SetCondition(condition)
		r.logger.Error(err, "Creating registry deployment is failed.")
		return err
	}

	return nil
}

func (r *RegistryDeployment) get(c client.Client, reg *regv1.Registry, conditions *status.Conditions) error {
	r.deploy = schemes.Deployment(reg)
	r.logger = utils.GetRegistryLogger(*r, r.deploy.Namespace, r.deploy.Name)

	req := types.NamespacedName{Name: r.deploy.Name, Namespace: r.deploy.Namespace}

	err := c.Get(context.TODO(), req, r.deploy)
	if err != nil {
		r.logger.Error(err, "Get regsitry deployment is failed")
		return err
	}

	return nil
}

func (r *RegistryDeployment) GetTypeName() string {
	return string(regv1.ConditionTypeDeployment)
}

func (r *RegistryDeployment) Patch(c client.Client, reg *regv1.Registry, useGet bool) error {
	return nil
}

func (r *RegistryDeployment) Ready(c client.Client, reg *regv1.Registry, conditions *status.Conditions, useGet bool) error {
	if r.deploy == nil || useGet {
		err := r.get(c, reg, conditions)
		if err != nil {
			r.logger.Error(err, "Deployment error")
			return err
		}
	}

	condition := status.Condition{
		Status: corev1.ConditionTrue,
		Type:   regv1.ConditionTypeDeployment,
	}

	conditions.SetCondition(condition)
	return nil
}

func (r *RegistryDeployment) StatusPatch(c client.Client, reg *regv1.Registry, conditions *status.Conditions, useGet bool) error {
	if r.deploy == nil || useGet {
		err := r.get(c, reg, conditions)
		if err != nil {
			r.logger.Error(err, "Deployment error")
			return err
		}
	}

	patch := client.MergeFrom(reg) // Set original obeject
	target := reg.DeepCopy()       // Target to Patch object

	for _, condition := range *conditions {
		r.logger.Info("patch condition", "type", string(condition.Type))
		target.Status.Conditions.SetCondition(condition)
	}

	err := c.Status().Patch(context.TODO(), target, patch)
	if err != nil {
		r.logger.Error(err, "Unknown error patching status")
		return err
	}
	return nil
}

func (r *RegistryDeployment) StatusUpdate(c client.Client, reg *regv1.Registry, conditions *status.Conditions, useGet bool) error {
	return nil
}

func (r *RegistryDeployment) Update(c client.Client, reg *regv1.Registry, useGet bool) error {
	return nil
}
