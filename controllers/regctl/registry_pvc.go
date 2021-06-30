package regctl

import (
	"context"
	"github.com/go-logr/logr"
	"github.com/tmax-cloud/registry-operator/internal/schemes"
	"strings"
	"time"

	"github.com/tmax-cloud/registry-operator/internal/utils"

	regv1 "github.com/tmax-cloud/registry-operator/api/v1"

	"github.com/operator-framework/operator-lib/status"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// NewRegistryPVC creates new registry pvc controller
func NewRegistryPVC(client client.Client, scheme *runtime.Scheme, reg *regv1.Registry, cond status.ConditionType, logger logr.Logger) *RegistryPVC {
	return &RegistryPVC{
		c:      client,
		scheme: scheme,
		cond:   cond,
		logger: logger.WithName("PVC"),
		pvc:    schemes.PersistentVolumeClaim(reg),
	}
}

// RegistryPVC things to handle pvc resource
type RegistryPVC struct {
	c      client.Client
	scheme *runtime.Scheme
	cond   status.ConditionType
	pvc    *corev1.PersistentVolumeClaim
	logger logr.Logger
}

// Handle makes pvc to be in the desired state
func (r *RegistryPVC) CreateIfNotExist(reg *regv1.Registry, patchReg *regv1.Registry) error {
	logger := r.logger.WithName("CreateIfNotExist")

	if err := r.get(reg); err != nil {
		r.notReady(patchReg, err)
		if errors.IsNotFound(err) {
			if err := r.create(reg, patchReg); err != nil {
				logger.Error(err, "create pvc error")
				r.notReady(patchReg, err)
				return err
			}
			logger.Info("Create Succeeded")
		} else {
			logger.Error(err, "pvc is error")
			return err
		}
		return nil
	}

	logger.Info("Check if patch exists.")
	diff := r.compare(reg)
	if len(diff) > 0 {
		logger.Info("patch exists.")
		r.notReady(patchReg, nil)
		if err := r.patch(reg, patchReg, diff); err != nil {
			logger.Error(err, "failed to patch pvc")
			r.notReady(patchReg, err)
			return err
		}
	}

	return nil
}

// Ready checks that pvc is ready
func (r *RegistryPVC) IsReady(reg *regv1.Registry, patchReg *regv1.Registry, useGet bool) error {
	logger := r.logger.WithName("IsReady")
	var err error = nil
	condition := &status.Condition{
		Status: corev1.ConditionFalse,
		Type:   regv1.ConditionTypePvc,
	}

	defer utils.SetErrorConditionIfChanged(patchReg, reg, condition, err)

	if r.pvc == nil || useGet {
		err := r.get(reg)
		if err != nil {
			logger.Error(err, "pvc error")
			return err
		}
	}

	if string(r.pvc.Status.Phase) == "pending" {
		logger.Info("NotReady")
		err = regv1.MakeRegistryError("NotReady")
		return err
	}

	patchReg.Status.Capacity = r.pvc.Status.Capacity.Storage().String()
	condition.Status = corev1.ConditionTrue

	logger.Info("Ready")
	return nil
}

func (r *RegistryPVC) create(reg *regv1.Registry, patchReg *regv1.Registry) error {
	logger := r.logger.WithName("create")
	if reg.Spec.PersistentVolumeClaim.Exist != nil {
		logger.Info("Use exist registry pvc. Need not to create pvc.")
		return nil
	}

	if reg.Spec.PersistentVolumeClaim.Create.DeleteWithPvc {
		if err := controllerutil.SetControllerReference(reg, r.pvc, r.scheme); err != nil {
			logger.Error(err, "SetOwnerReference Failed")
			return err
		}
	}

	logger.Info("Create registry pvc")
	if err := r.c.Create(context.TODO(), r.pvc); err != nil {
		logger.Error(err, "Creating registry pvc is failed.")
		return err
	}

	return nil
}

func (r *RegistryPVC) get(reg *regv1.Registry) error {
	logger := r.logger.WithName("get")
	req := types.NamespacedName{Name: r.pvc.Name, Namespace: r.pvc.Namespace}
	err := r.c.Get(context.TODO(), req, r.pvc)
	if err != nil {
		logger.Error(err, "Get regsitry pvc is failed")
		return err
	}
	return nil
}

func (r *RegistryPVC) patch(reg *regv1.Registry, patchReg *regv1.Registry, diff []utils.Diff) error {
	logger := r.logger.WithName("patch")
	target := r.pvc.DeepCopy()
	originObject := client.MergeFrom(r.pvc)

	logger.Info("Get", "Patch Keys", strings.Join(utils.DiffKeyList(diff), ","))

	for _, d := range diff {
		switch d.Key {
		case "DeleteWithPvc":
			if d.Type == utils.Remove {
				logger.Info("Remove OwnerReferences")
				target.OwnerReferences = []metav1.OwnerReference{}
			} else {
				logger.Info("Replace or Add OwnerReferences")
				if err := controllerutil.SetControllerReference(reg, target, r.scheme); err != nil {
					logger.Error(err, "SetOwnerReference Failed")
					condition := status.Condition{
						Status:  corev1.ConditionFalse,
						Type:    regv1.ConditionTypePvc,
						Message: err.Error(),
					}

					patchReg.Status.Conditions.SetCondition(condition)
					return err
				}
			}
		}
	}

	// Patch
	if err := r.c.Patch(context.TODO(), target, originObject); err != nil {
		logger.Error(err, "Unknown error patching status")
		return err
	}
	return nil
}

func (r *RegistryPVC) delete(patchReg *regv1.Registry) error {
	logger := r.logger.WithName("delete")
	if err := r.c.Delete(context.TODO(), r.pvc); err != nil {
		logger.Error(err, "Unknown error delete pvc")
		return err
	}
	return nil
}

func (r *RegistryPVC) compare(reg *regv1.Registry) []utils.Diff {
	logger := r.logger.WithName("compare")
	diff := []utils.Diff{}
	regPvc := reg.Spec.PersistentVolumeClaim

	if regPvc.Create != nil {
		if regPvc.Create.DeleteWithPvc {
			if len(r.pvc.OwnerReferences) == 0 {
				diff = append(diff, utils.Diff{Type: utils.Add, Key: "DeleteWithPvc"})
			}
		} else {
			if len(r.pvc.OwnerReferences) != 0 {
				diff = append(diff, utils.Diff{Type: utils.Remove, Key: "DeleteWithPvc"})
			}
		}
	}
	if len(diff) > 0 {
		logger.Info("diff exist")
	}
	return diff
}

func (r *RegistryPVC) IsSuccessfullyCompleted(reg *regv1.Registry) bool {
	cond := reg.Status.Conditions.GetCondition(regv1.ConditionTypePvc)
	if cond == nil {
		return false
	}

	return cond.IsTrue()
}

func (r *RegistryPVC) notReady(patchReg *regv1.Registry, err error) {
	condition := &status.Condition{
		Status: corev1.ConditionFalse,
		Type:   regv1.ConditionTypePvc,
	}
	utils.SetCondition(err, patchReg, condition)
}

// Condition returns dependent subresource's condition type
func (r *RegistryPVC) Condition() string {
	return string(regv1.ConditionTypePvc)
}

// ModifiedTime returns the modified time of the subresource condition
func (r *RegistryPVC) ModifiedTime(patchReg *regv1.Registry) []time.Time {
	cond := patchReg.Status.Conditions.GetCondition(regv1.ConditionTypePvc)
	if cond == nil {
		return nil
	}

	return []time.Time{cond.LastTransitionTime.Time}
}
