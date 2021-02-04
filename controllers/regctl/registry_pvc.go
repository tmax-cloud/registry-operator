package regctl

import (
	"context"
	"strings"

	"github.com/tmax-cloud/registry-operator/internal/utils"

	"github.com/tmax-cloud/registry-operator/internal/schemes"

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

// RegistryPVC things to handle pvc resource
type RegistryPVC struct {
	pvc    *corev1.PersistentVolumeClaim
	logger *utils.RegistryLogger
	scheme *runtime.Scheme
}

// Handle makes pvc to be in the desired state
func (r *RegistryPVC) Handle(c client.Client, reg *regv1.Registry, patchReg *regv1.Registry, scheme *runtime.Scheme) error {
	if err := r.get(c, reg); err != nil {
		if errors.IsNotFound(err) {
			if err := r.create(c, reg, patchReg, scheme); err != nil {
				r.logger.Error(err, "create pvc error")
				return err
			}
		} else {
			r.logger.Error(err, "pvc is error")
			return err
		}
	}

	r.scheme = scheme

	r.logger.Info("Check if patch exists.")
	diff := r.compare(reg)
	if len(diff) > 0 {
		r.logger.Info("patch exists.")
		if err := r.patch(c, reg, patchReg, diff); err != nil {
			return err
		}
	}

	return nil
}

// Ready checks that pvc is ready
func (r *RegistryPVC) Ready(c client.Client, reg *regv1.Registry, patchReg *regv1.Registry, useGet bool) error {
	var err error = nil
	condition := &status.Condition{
		Status: corev1.ConditionFalse,
		Type:   regv1.ConditionTypePvc,
	}

	defer utils.SetCondition(err, patchReg, condition)

	if r.pvc == nil || useGet {
		err := r.get(c, reg)
		if err != nil {
			r.logger.Error(err, "pvc error")
			return err
		}
	}

	if string(r.pvc.Status.Phase) == "pending" {
		r.logger.Info("NotReady")
		err = regv1.MakeRegistryError("NotReady")
		return err
	}

	patchReg.Status.Capacity = r.pvc.Status.Capacity.Storage().String()
	condition.Status = corev1.ConditionTrue

	r.logger.Info("Ready")
	return nil
}

func (r *RegistryPVC) create(c client.Client, reg *regv1.Registry, patchReg *regv1.Registry, scheme *runtime.Scheme) error {
	if reg.Spec.PersistentVolumeClaim.Exist != nil {
		r.logger.Info("Use exist registry pvc. Need not to create pvc.")
		return nil
	}

	if reg.Spec.PersistentVolumeClaim.Create.DeleteWithPvc {
		if err := controllerutil.SetControllerReference(reg, r.pvc, scheme); err != nil {
			r.logger.Error(err, "SetOwnerReference Failed")
			condition := status.Condition{
				Status:  corev1.ConditionFalse,
				Type:    regv1.ConditionTypePvc,
				Message: err.Error(),
			}

			patchReg.Status.Conditions.SetCondition(condition)
			return err
		}
	}

	r.logger.Info("Create registry pvc")
	err := c.Create(context.TODO(), r.pvc)
	if err != nil {
		condition := status.Condition{
			Status:  corev1.ConditionFalse,
			Type:    regv1.ConditionTypePvc,
			Message: err.Error(),
		}

		patchReg.Status.Conditions.SetCondition(condition)
		r.logger.Error(err, "Creating registry pvc is failed.")
		return err
	}

	return nil
}

func (r *RegistryPVC) get(c client.Client, reg *regv1.Registry) error {
	r.pvc = schemes.PersistentVolumeClaim(reg)
	r.logger = utils.NewRegistryLogger(*r, r.pvc.Namespace, r.pvc.Name)

	req := types.NamespacedName{Name: r.pvc.Name, Namespace: r.pvc.Namespace}
	err := c.Get(context.TODO(), req, r.pvc)
	if err != nil {
		r.logger.Error(err, "Get regsitry pvc is failed")
		return err
	}

	return nil
}

func (r *RegistryPVC) patch(c client.Client, reg *regv1.Registry, patchReg *regv1.Registry, diff []utils.Diff) error {
	target := r.pvc.DeepCopy()
	originObject := client.MergeFrom(r.pvc)

	r.logger.Info("Get", "Patch Keys", strings.Join(utils.DiffKeyList(diff), ","))

	for _, d := range diff {
		switch d.Key {
		case "DeleteWithPvc":
			if d.Type == utils.Remove {
				r.logger.Info("Remove OwnerReferences")
				target.OwnerReferences = []metav1.OwnerReference{}
			} else {
				r.logger.Info("Replace or Add OwnerReferences")
				if err := controllerutil.SetControllerReference(reg, target, r.scheme); err != nil {
					r.logger.Error(err, "SetOwnerReference Failed")
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
	if err := c.Patch(context.TODO(), target, originObject); err != nil {
		r.logger.Error(err, "Unknown error patching status")
		return err
	}
	return nil
}

func (r *RegistryPVC) delete(c client.Client, patchReg *regv1.Registry) error {
	if err := c.Delete(context.TODO(), r.pvc); err != nil {
		r.logger.Error(err, "Unknown error delete pvc")
		return err
	}

	condition := status.Condition{
		Type:   regv1.ConditionTypePvc,
		Status: corev1.ConditionFalse,
	}

	patchReg.Status.Conditions.SetCondition(condition)
	return nil
}

func (r *RegistryPVC) compare(reg *regv1.Registry) []utils.Diff {
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

	return diff
}
