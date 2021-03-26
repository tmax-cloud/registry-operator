package regctl

import (
	"context"

	"github.com/tmax-cloud/registry-operator/internal/schemes"
	"github.com/tmax-cloud/registry-operator/internal/utils"

	regv1 "github.com/tmax-cloud/registry-operator/api/v1"

	"github.com/operator-framework/operator-lib/status"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// RegistryRole contains things to handle deployment resource
type RegistryRole struct {
	role   *rbacv1.Role
	logger *utils.RegistryLogger
}

// Handle makes role to be in the desired state
func (r *RegistryRole) Handle(c client.Client, reg *regv1.Registry, patchReg *regv1.Registry, scheme *runtime.Scheme) error {
	if err := r.get(c, reg); err != nil {
		if errors.IsNotFound(err) {
			if err := r.create(c, reg, patchReg, scheme); err != nil {
				r.logger.Error(err, "create role error")
				return err
			}
		} else {
			r.logger.Error(err, "role error")
			return err
		}
	}

	return nil
}

// Ready checks that role is ready
func (r *RegistryRole) Ready(c client.Client, reg *regv1.Registry, patchReg *regv1.Registry, useGet bool) error {
	var err error = nil
	condition := &status.Condition{
		Status: corev1.ConditionFalse,
		Type:   regv1.ConditionTypeConfigMap,
	}
	defer utils.SetCondition(err, patchReg, condition)

	if useGet {
		err = r.get(c, reg)
		if err != nil {
			r.logger.Error(err, "PersistentVolumeClaim error")
			return err
		}
	}

	_, exist := r.role.Data["config.yml"]
	if !exist {
		r.logger.Info("NotReady")
		err = regv1.MakeRegistryError("NotReady")
		return err
	}

	r.logger.Info("Ready")
	condition.Status = corev1.ConditionTrue
	return nil
}

func (r *RegistryRole) create(c client.Client, reg *regv1.Registry, patchReg *regv1.Registry, scheme *runtime.Scheme) error {
	condition := &status.Condition{
		Status: corev1.ConditionFalse,
		Type:   regv1.ConditionTypeRole,
	}

	if err := controllerutil.SetControllerReference(reg, r.role, scheme); err != nil {
		r.logger.Error(err, "Set owner reference failed")
		utils.SetCondition(err, patchReg, condition)
		return err
	}

	if err := c.Create(context.TODO(), r.role); err != nil {
		r.logger.Error(err, "Create failed")
		utils.SetCondition(err, patchReg, condition)
		return err
	}

	r.logger.Info("Succeed")
	return nil
}

func (r *RegistryRole) get(c client.Client, reg *regv1.Registry) error {
	r.role = schemes.ConfigMap(reg, map[string]string{})
	r.logger = utils.NewRegistryLogger(*r, r.role.Namespace, r.role.Name)

	req := types.NamespacedName{Name: r.role.Name, Namespace: r.role.Namespace}
	err := c.Get(context.TODO(), req, r.role)
	if err != nil {
		r.logger.Error(err, "Get regsitry role is failed")
		return err
	}

	return nil
}

func (r *RegistryRole) patch(c client.Client, reg *regv1.Registry, patchReg *regv1.Registry, diff []utils.Diff) error {
	return nil
}

func (r *RegistryRole) delete(c client.Client, patchReg *regv1.Registry) error {
	if err := c.Delete(context.TODO(), r.role); err != nil {
		r.logger.Error(err, "Unknown error delete role")
		return err
	}
	condition := status.Condition{
		Type:   regv1.ConditionTypeConfigMap,
		Status: corev1.ConditionFalse,
	}

	patchReg.Status.Conditions.SetCondition(condition)
	return nil
}

func (r *RegistryRole) compare(reg *regv1.Registry) []utils.Diff {
	return nil
}
