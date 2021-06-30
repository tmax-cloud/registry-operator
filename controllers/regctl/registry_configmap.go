package regctl

import (
	"context"
	"github.com/go-logr/logr"
	"time"

	"github.com/tmax-cloud/registry-operator/internal/schemes"
	"github.com/tmax-cloud/registry-operator/internal/utils"

	regv1 "github.com/tmax-cloud/registry-operator/api/v1"

	"github.com/operator-framework/operator-lib/status"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// RegistryConfigMap contains things to handle deployment resource
type RegistryConfigMap struct {
	c      client.Client
	scheme *runtime.Scheme
	reg    *regv1.Registry
	cm     *corev1.ConfigMap
	logger logr.Logger
}

// NewRegistryConfigMap creates new registry configmap controller
func NewRegistryConfigMap(client client.Client, scheme *runtime.Scheme, reg *regv1.Registry, logger logr.Logger) *RegistryConfigMap {
	return &RegistryConfigMap{
		c:      client,
		scheme: scheme,
		reg:    reg,
		logger: logger.WithName("Configmap"),
	}
}

// Handle makes configmap to be in the desired state
func (r *RegistryConfigMap) CreateIfNotExist(reg *regv1.Registry, patchReg *regv1.Registry) error {
	if err := r.get(reg); err != nil {
		r.notReady(patchReg, err)
		if errors.IsNotFound(err) {
			if err := r.create(reg, patchReg); err != nil {
				r.logger.Error(err, "create configmap error")
				r.notReady(patchReg, err)
				return err
			}
			r.logger.Info("Create Succeeded")
		} else {
			r.logger.Error(err, "configmap error")
			return err
		}
		return nil
	}

	return nil
}

// Ready checks that configmap is ready
func (r *RegistryConfigMap) IsReady(reg *regv1.Registry, patchReg *regv1.Registry, useGet bool) error {
	var err error = nil
	condition := &status.Condition{
		Status: corev1.ConditionFalse,
		Type:   regv1.ConditionTypeConfigMap,
	}
	defer utils.SetErrorConditionIfChanged(patchReg, reg, condition, err)

	if useGet {
		err = r.get(reg)
		if err != nil {
			r.logger.Error(err, "PersistentVolumeClaim error")
			return err
		}
	}

	_, exist := r.cm.Data["config.yml"]
	if !exist {
		r.logger.Info("NotReady")
		err = regv1.MakeRegistryError("NotReady")
		return err
	}

	r.logger.Info("Ready")
	condition.Status = corev1.ConditionTrue
	return nil
}

func (r *RegistryConfigMap) create(reg *regv1.Registry, patchReg *regv1.Registry) error {
	if len(reg.Spec.CustomConfigYml) > 0 {
		r.logger.Info("Use exist registry configmap. Need not to create configmap. (Configmap: " + reg.Spec.CustomConfigYml + ")")
		return nil
	}

	defaultCm := &corev1.ConfigMap{}
	defaultCmType := schemes.DefaultConfigMapType()

	// Read Default ConfigMap
	if err := r.c.Get(context.TODO(), *defaultCmType, defaultCm); err != nil {
		r.logger.Error(err, "get default configmap error")
		return nil
	}

	r.cm = schemes.ConfigMap(reg, defaultCm.Data)

	if err := controllerutil.SetControllerReference(reg, r.cm, r.scheme); err != nil {
		r.logger.Error(err, "SetOwnerReference Failed")
		condition := status.Condition{
			Status:  corev1.ConditionFalse,
			Type:    regv1.ConditionTypeConfigMap,
			Message: err.Error(),
		}

		patchReg.Status.Conditions.SetCondition(condition)
		return nil
	}

	r.logger.Info("Create registry configmap")
	err := r.c.Create(context.TODO(), r.cm)
	if err != nil {
		condition := status.Condition{
			Status:  corev1.ConditionFalse,
			Type:    regv1.ConditionTypeConfigMap,
			Message: err.Error(),
		}

		patchReg.Status.Conditions.SetCondition(condition)
		r.logger.Error(err, "Creating registry configmap is failed.")
		return nil
	}

	return nil
}

func (r *RegistryConfigMap) get(reg *regv1.Registry) error {
	r.cm = schemes.ConfigMap(reg, map[string]string{})

	req := types.NamespacedName{Name: r.cm.Name, Namespace: r.cm.Namespace}
	err := r.c.Get(context.TODO(), req, r.cm)
	if err != nil {
		r.logger.Error(err, "Get regsitry configmap is failed")
		return err
	}

	return nil
}

func (r *RegistryConfigMap) patch(reg *regv1.Registry, patchReg *regv1.Registry, diff []utils.Diff) error {
	return nil
}

func (r *RegistryConfigMap) delete(patchReg *regv1.Registry) error {
	if err := r.c.Delete(context.TODO(), r.cm); err != nil {
		r.logger.Error(err, "Unknown error delete configmap")
		return err
	}
	condition := status.Condition{
		Type:   regv1.ConditionTypeConfigMap,
		Status: corev1.ConditionFalse,
	}

	patchReg.Status.Conditions.SetCondition(condition)
	return nil
}

func (r *RegistryConfigMap) compare(reg *regv1.Registry) []utils.Diff {
	return nil
}

// IsSuccessfullyCompleted returns true if condition is satisfied
func (r *RegistryConfigMap) IsSuccessfullyCompleted(reg *regv1.Registry) bool {
	cond := reg.Status.Conditions.GetCondition(regv1.ConditionTypeConfigMap)
	if cond == nil {
		return false
	}

	return cond.IsTrue()
}

func (r *RegistryConfigMap) notReady(patchReg *regv1.Registry, err error) {
	condition := &status.Condition{
		Status: corev1.ConditionFalse,
		Type:   regv1.ConditionTypeConfigMap,
	}
	utils.SetCondition(err, patchReg, condition)
}

// Condition returns dependent subresource's condition type
func (r *RegistryConfigMap) Condition() string {
	return string(regv1.ConditionTypeConfigMap)
}

// ModifiedTime returns the modified time of the subresource condition
func (r *RegistryConfigMap) ModifiedTime(patchReg *regv1.Registry) []time.Time {
	cond := patchReg.Status.Conditions.GetCondition(regv1.ConditionTypeConfigMap)
	if cond == nil {
		return nil
	}

	return []time.Time{cond.LastTransitionTime.Time}
}
