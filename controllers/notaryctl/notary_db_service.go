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

type NotaryDBService struct {
	svc    *corev1.Service
	logger *utils.RegistryLogger
}

// Handle is to create notary db service.
func (nt *NotaryDBService) Handle(c client.Client, notary *regv1.Notary, patchNotary *regv1.Notary, scheme *runtime.Scheme) error {
	if err := nt.get(c, notary); err != nil {
		if errors.IsNotFound(err) {
			if err := nt.create(c, notary, patchNotary, scheme); err != nil {
				nt.logger.Error(err, "create service error")
				return err
			}
		} else {
			nt.logger.Error(err, "service error")
			return err
		}
	}

	return nil
}

// Ready is to check if the service is ready and to set the condition
func (nt *NotaryDBService) Ready(c client.Client, notary *regv1.Notary, patchNotary *regv1.Notary, useGet bool) error {
	var err error = nil
	condition := &status.Condition{
		Status: corev1.ConditionFalse,
		Type:   regv1.ConditionTypeNotaryDBService,
	}

	defer utils.SetCondition(err, patchNotary, condition)

	if useGet {
		err = nt.get(c, notary)
		if err != nil {
			nt.logger.Error(err, "get service error")
			return err
		}
	}

	nt.logger.Info("Ready")
	condition.Status = corev1.ConditionTrue
	return nil
}

func (nt *NotaryDBService) create(c client.Client, notary *regv1.Notary, patchNotary *regv1.Notary, scheme *runtime.Scheme) error {
	var err error = nil
	condition := &status.Condition{
		Status: corev1.ConditionFalse,
		Type:   regv1.ConditionTypeNotaryDBService,
	}

	defer utils.SetCondition(err, patchNotary, condition)

	if err = controllerutil.SetControllerReference(notary, nt.svc, scheme); err != nil {
		nt.logger.Error(err, "SetOwnerReference Failed")

		return nil
	}

	nt.logger.Info("Create notary db service")
	if err = c.Create(context.TODO(), nt.svc); err != nil {
		nt.logger.Error(err, "Creating notary db service is failed.")
		return nil
	}

	return nil
}

func (nt *NotaryDBService) get(c client.Client, notary *regv1.Notary) error {
	nt.svc = schemes.NotaryDBService(notary)
	nt.logger = utils.NewRegistryLogger(*nt, nt.svc.Namespace, nt.svc.Name)

	req := types.NamespacedName{Name: nt.svc.Name, Namespace: nt.svc.Namespace}

	if err := c.Get(context.TODO(), req, nt.svc); err != nil {
		nt.logger.Error(err, "Get notary db service is failed")
		return err

	}

	return nil
}

func (nt *NotaryDBService) delete(c client.Client, patchNotary *regv1.Notary) error {
	var err error = nil
	condition := &status.Condition{
		Status: corev1.ConditionFalse,
		Type:   regv1.ConditionTypeNotaryDBService,
	}

	defer utils.SetCondition(err, patchNotary, condition)

	if err = c.Delete(context.TODO(), nt.svc); err != nil {
		nt.logger.Error(err, "Unknown error delete service")
		return err
	}

	return nil
}
