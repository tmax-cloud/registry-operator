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

type NotaryServerSecret struct {
	secret *corev1.Secret
	logger *utils.RegistryLogger
}

// Handle is to create notary server secret.
func (nt *NotaryServerSecret) Handle(c client.Client, notary *regv1.Notary, patchNotary *regv1.Notary, scheme *runtime.Scheme) error {
	if err := nt.get(c, notary); err != nil {
		if errors.IsNotFound(err) {
			if err := nt.create(c, notary, patchNotary, scheme); err != nil {
				nt.logger.Error(err, "create secret error")
				return err
			}
		} else {
			nt.logger.Error(err, "secret error")
			return err
		}
	}

	return nil
}

// Ready is to check if the secret is ready and to set the condition
func (nt *NotaryServerSecret) Ready(c client.Client, notary *regv1.Notary, patchNotary *regv1.Notary, useGet bool) error {
	var err error = nil
	condition := &status.Condition{
		Status: corev1.ConditionFalse,
		Type:   regv1.ConditionTypeNotaryServerSecret,
	}

	defer utils.SetCondition(err, patchNotary, condition)

	if useGet {
		err = nt.get(c, notary)
		if err != nil {
			nt.logger.Error(err, "get secret error")
			return err
		}
	}

	nt.logger.Info("Ready")
	condition.Status = corev1.ConditionTrue
	return nil
}

func (nt *NotaryServerSecret) create(c client.Client, notary *regv1.Notary, patchNotary *regv1.Notary, scheme *runtime.Scheme) error {
	var err error = nil
	condition := &status.Condition{
		Status: corev1.ConditionFalse,
		Type:   regv1.ConditionTypeNotaryServerSecret,
	}

	defer utils.SetCondition(err, patchNotary, condition)

	if err = controllerutil.SetControllerReference(notary, nt.secret, scheme); err != nil {
		nt.logger.Error(err, "SetOwnerReference Failed")

		return nil
	}

	nt.logger.Info("Create notary server secret")
	if err = c.Create(context.TODO(), nt.secret); err != nil {
		nt.logger.Error(err, "Creating notary server secret is failed.")
		return nil
	}

	return nil
}

func (nt *NotaryServerSecret) get(c client.Client, notary *regv1.Notary) error {
	nt.logger = utils.NewRegistryLogger(*nt, notary.Namespace, schemes.SubresourceName(notary, schemes.SubTypeNotaryServerSecret))
	secret, err := schemes.NotaryServerSecret(notary, c)
	if err != nil {
		nt.logger.Error(err, "create scheme failed")
		return err
	}
	nt.secret = secret

	req := types.NamespacedName{Name: nt.secret.Name, Namespace: nt.secret.Namespace}

	if err := c.Get(context.TODO(), req, nt.secret); err != nil {
		nt.logger.Error(err, "Get notary server secret is failed")
		return err

	}

	return nil
}

func (nt *NotaryServerSecret) delete(c client.Client, patchNotary *regv1.Notary) error {
	var err error = nil
	condition := &status.Condition{
		Status: corev1.ConditionFalse,
		Type:   regv1.ConditionTypeNotaryServerSecret,
	}

	defer utils.SetCondition(err, patchNotary, condition)

	if err = c.Delete(context.TODO(), nt.secret); err != nil {
		nt.logger.Error(err, "Unknown error delete secret")
		return err
	}

	return nil
}
