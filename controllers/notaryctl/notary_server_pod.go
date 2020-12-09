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

// Handle is to create notary server pod.
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

// Ready is to check if the pod is ready and to set the condition
func (nt *NotaryServer) Ready(c client.Client, notary *regv1.Notary, patchNotary *regv1.Notary, useGet bool) error {
	var err error = nil
	condition := &status.Condition{
		Status: corev1.ConditionFalse,
		Type:   regv1.ConditionTypeNotaryServerPod,
	}

	defer utils.SetCondition(err, patchNotary, condition)

	if useGet {
		err = nt.get(c, notary)
		if err != nil {
			nt.logger.Error(err, "get pod error")
			return err
		}
	}

	// TODO: check if not ready

	nt.logger.Info("Ready")
	condition.Status = corev1.ConditionTrue
	return nil
}

func (nt *NotaryServer) create(c client.Client, notary *regv1.Notary, patchNotary *regv1.Notary, scheme *runtime.Scheme) error {
	var err error = nil
	condition := &status.Condition{
		Status: corev1.ConditionFalse,
		Type:   regv1.ConditionTypeNotaryServerPod,
	}

	defer utils.SetCondition(err, patchNotary, condition)

	if err = controllerutil.SetControllerReference(notary, nt.pod, scheme); err != nil {
		nt.logger.Error(err, "SetOwnerReference Failed")

		return nil
	}

	nt.logger.Info("Create notary server pod")
	if err = c.Create(context.TODO(), nt.pod); err != nil {
		nt.logger.Error(err, "Creating notary server pod is failed.")
		return nil
	}

	return nil
}

func (nt *NotaryServer) get(c client.Client, notary *regv1.Notary) error {
	nt.pod = schemes.NotaryServerPod(notary)
	nt.logger = utils.NewRegistryLogger(*nt, nt.pod.Namespace, nt.pod.Name)

	req := types.NamespacedName{Name: nt.pod.Name, Namespace: nt.pod.Namespace}

	if err := c.Get(context.TODO(), req, nt.pod); err != nil {
		nt.logger.Error(err, "Get notary server pod is failed")
		return err

	}

	return nil
}

func (nt *NotaryServer) delete(c client.Client, patchNotary *regv1.Notary) error {
	var err error = nil
	condition := &status.Condition{
		Status: corev1.ConditionFalse,
		Type:   regv1.ConditionTypeNotaryServerPod,
	}

	defer utils.SetCondition(err, patchNotary, condition)

	if err = c.Delete(context.TODO(), nt.pod); err != nil {
		nt.logger.Error(err, "Unknown error delete pod")
		return err
	}

	return nil
}
