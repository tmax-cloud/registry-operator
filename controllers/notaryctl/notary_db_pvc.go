package notaryctl

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

type NotaryDBPVC struct {
	pvc    *corev1.PersistentVolumeClaim
	logger *utils.RegistryLogger
}

// Handle is to create notary db pvc.
func (nt *NotaryDBPVC) Handle(c client.Client, notary *regv1.Notary, patchNotary *regv1.Notary, scheme *runtime.Scheme) error {
	if err := nt.get(c, notary); err != nil {
		if errors.IsNotFound(err) {
			if err := nt.create(c, notary, patchNotary, scheme); err != nil {
				nt.logger.Error(err, "create pod error")
				return err
			}
		} else {
			nt.logger.Error(err, "pod error")
			return err
		}
	}

	return nil
}

// Ready is to check if the pvc is ready and to set the condition
func (nt *NotaryDBPVC) Ready(c client.Client, notary *regv1.Notary, patchNotary *regv1.Notary, useGet bool) error {
	var err error = nil
	condition := &status.Condition{
		Status: corev1.ConditionFalse,
		Type:   regv1.ConditionTypeNotaryDBPVC,
	}

	defer utils.SetCondition(err, patchNotary, condition)

	if useGet {
		err = nt.get(c, notary)
		if err != nil {
			nt.logger.Error(err, "get pod error")
			return err
		}
	}

	if string(nt.pvc.Status.Phase) == "pending" {
		nt.logger.Info("NotReady")
		err = regv1.MakeRegistryError("NotReady")
		return err
	}

	nt.logger.Info("Ready")
	condition.Status = corev1.ConditionTrue
	return nil
}

func (nt *NotaryDBPVC) create(c client.Client, notary *regv1.Notary, patchNotary *regv1.Notary, scheme *runtime.Scheme) error {
	var err error = nil
	condition := &status.Condition{
		Status: corev1.ConditionFalse,
		Type:   regv1.ConditionTypeNotaryDBPVC,
	}

	defer utils.SetCondition(err, patchNotary, condition)

	if err = controllerutil.SetControllerReference(notary, nt.pvc, scheme); err != nil {
		nt.logger.Error(err, "SetOwnerReference Failed")

		return nil
	}

	nt.logger.Info("Create notary db pvc")
	if err = c.Create(context.TODO(), nt.pvc); err != nil {
		nt.logger.Error(err, "Creating notary db pvc is failed.")
		return nil
	}

	return nil
}

func (nt *NotaryDBPVC) get(c client.Client, notary *regv1.Notary) error {
	nt.pvc = schemes.NotaryDBPVC(notary)
	nt.logger = utils.NewRegistryLogger(*nt, nt.pvc.Namespace, nt.pvc.Name)

	req := types.NamespacedName{Name: nt.pvc.Name, Namespace: nt.pvc.Namespace}
	err := c.Get(context.TODO(), req, nt.pvc)
	if err != nil {
		nt.logger.Error(err, "Get notary pvc is failed")
		return err
	}

	return nil
}

func (nt *NotaryDBPVC) delete(c client.Client, patchNotary *regv1.Notary) error {
	var err error = nil
	condition := &status.Condition{
		Status: corev1.ConditionFalse,
		Type:   regv1.ConditionTypeNotaryDBPVC,
	}

	defer utils.SetCondition(err, patchNotary, condition)

	if err = c.Delete(context.TODO(), nt.pvc); err != nil {
		nt.logger.Error(err, "Unknown error delete pvc")
		return err
	}

	return nil
}
