package notaryctl

import (
	"context"

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

type NotaryServer struct {
	pod    *corev1.Pod
	logger *utils.RegistryLogger
}

func (nt *NotaryServer) Handle(c client.Client, notary *regv1.Notary, patchNotary *regv1.Notary, scheme *runtime.Scheme) error {
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

func (nt *NotaryServer) Ready(c client.Client, notary *regv1.Notary, patchNotary *regv1.Notary, useGet bool) error {
	var err error = nil
	condition := &status.Condition{
		Status: corev1.ConditionFalse,
		Type:   regv1.ConditionTypeDeployment,
	}
	if useGet {
		err = nt.get(c, notary)
		if err != nil {
			nt.logger.Error(err, "Deployment error")
			return err
		}
	}

	if nt.pod == nil {
		nt.logger.Info("NotReady")

		err = regv1.MakeRegistryError("NotReady")
		return err
	}

	nt.logger.Info("Ready")
	condition.Status = corev1.ConditionTrue
	return nil
}

func (nt *NotaryServer) create(c client.Client, reg *regv1.Notary, patchReg *regv1.Notary, scheme *runtime.Scheme) error {
	if err := controllerutil.SetControllerReference(reg, nt.pod, scheme); err != nil {
		nt.logger.Error(err, "SetOwnerReference Failed")
		condition := status.Condition{
			Status:  corev1.ConditionFalse,
			Type:    regv1.ConditionTypeNotaryServerPod,
			Message: err.Error(),
		}

		patchReg.Status.Conditions.SetCondition(condition)
		return nil
	}

	nt.logger.Info("Create notary server pod")
	err := c.Create(context.TODO(), nt.pod)
	if err != nil {
		condition := status.Condition{
			Status:  corev1.ConditionFalse,
			Type:    regv1.ConditionTypeNotaryServerPod,
			Message: err.Error(),
		}

		patchReg.Status.Conditions.SetCondition(condition)
		nt.logger.Error(err, "Creating notary server pod is failed.")
		return nil
	}

	return nil
}

func (nt *NotaryServer) get(c client.Client, notary *regv1.Notary) error {
	nt.pod = schemes.NotaryServerPod(notary)
	nt.logger = utils.NewRegistryLogger(*nt, nt.pod.Namespace, nt.pod.Name)

	req := types.NamespacedName{Name: nt.pod.Name, Namespace: nt.pod.Namespace}

	err := c.Get(context.TODO(), req, nt.pod)
	if err != nil {
		nt.logger.Error(err, "Get notary server pod is failed")
		return err
	}

	return nil
}

func (nt *NotaryServer) delete(c client.Client, patchReg *regv1.Notary) error {
	if err := c.Delete(context.TODO(), nt.pod); err != nil {
		nt.logger.Error(err, "Unknown error delete deployment")
		return err
	}

	condition := status.Condition{
		Type:   regv1.ConditionTypeDeployment,
		Status: corev1.ConditionFalse,
	}

	patchReg.Status.Conditions.SetCondition(condition)

	return nil
}
