package notaryctl

import (
	"context"

	"github.com/operator-framework/operator-lib/status"
	regv1 "github.com/tmax-cloud/registry-operator/api/v1"
	"github.com/tmax-cloud/registry-operator/internal/schemes"
	"github.com/tmax-cloud/registry-operator/internal/utils"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type NotaryServerIngress struct {
	ingress *v1beta1.Ingress
	logger  *utils.RegistryLogger
}

// Handle is to create notary server ingress.
func (nt *NotaryServerIngress) Handle(c client.Client, notary *regv1.Notary, patchNotary *regv1.Notary, scheme *runtime.Scheme) error {
	if err := nt.get(c, notary); err != nil {
		if errors.IsNotFound(err) {
			if err := nt.create(c, notary, patchNotary, scheme); err != nil {
				nt.logger.Error(err, "create ingress error")
				return err
			}
		} else {
			nt.logger.Error(err, "ingress error")
			return err
		}
	}

	return nil
}

// Ready is to check if the ingress is ready and to set the condition
func (nt *NotaryServerIngress) Ready(c client.Client, notary *regv1.Notary, patchNotary *regv1.Notary, useGet bool) error {
	var err error = nil
	condition := &status.Condition{
		Status: corev1.ConditionFalse,
		Type:   regv1.ConditionTypeNotaryServerIngress,
	}

	defer utils.SetCondition(err, patchNotary, condition)

	if useGet {
		err = nt.get(c, notary)
		if err != nil {
			nt.logger.Error(err, "get ingress error")
			return err
		}
	}

	if _, ok := nt.ingress.Annotations["kubernetes.io/ingress.class"]; !ok {
		err = regv1.MakeRegistryError("Ingress Error")
		return err
	}
	if _, ok := nt.ingress.Annotations["nginx.ingress.kubernetes.io/proxy-connect-timeout"]; !ok {
		err = regv1.MakeRegistryError("Ingress Error")
		return err
	}
	if _, ok := nt.ingress.Annotations["nginx.ingress.kubernetes.io/proxy-read-timeout"]; !ok {
		err = regv1.MakeRegistryError("Ingress Error")
		return err
	}
	if _, ok := nt.ingress.Annotations["nginx.ingress.kubernetes.io/ssl-redirect"]; !ok {
		err = regv1.MakeRegistryError("Ingress Error")
		return err
	}
	if val, ok := nt.ingress.Annotations["nginx.ingress.kubernetes.io/backend-protocol"]; ok {
		if val != "HTTPS" {
			err = regv1.MakeRegistryError("Ingress Error")
			return err
		}
	} else {
		err = regv1.MakeRegistryError("Ingress Error")
		return err
	}
	if _, ok := nt.ingress.Annotations["nginx.ingress.kubernetes.io/proxy-body-size"]; !ok {
		err = regv1.MakeRegistryError("Ingress Error")
		return err
	}

	if len(nt.ingress.Spec.TLS) > 0 {
		for _, host := range nt.ingress.Spec.TLS[0].Hosts {
			patchNotary.Status.NotaryURL = "https://" + host + ":443"
		}
	}

	condition.Status = corev1.ConditionTrue
	nt.logger.Info("Ready")
	return nil
}

func (nt *NotaryServerIngress) create(c client.Client, notary *regv1.Notary, patchNotary *regv1.Notary, scheme *runtime.Scheme) error {
	var err error = nil
	condition := &status.Condition{
		Status: corev1.ConditionFalse,
		Type:   regv1.ConditionTypeNotaryServerIngress,
	}

	defer utils.SetCondition(err, patchNotary, condition)

	if err = controllerutil.SetControllerReference(notary, nt.ingress, scheme); err != nil {
		nt.logger.Error(err, "SetOwnerReference Failed")

		return nil
	}

	nt.logger.Info("Create notary server ingress")
	if err = c.Create(context.TODO(), nt.ingress); err != nil {
		nt.logger.Error(err, "Creating notary server ingress is failed.")
		return nil
	}

	return nil
}

func (nt *NotaryServerIngress) get(c client.Client, notary *regv1.Notary) error {
	nt.ingress = schemes.NotaryServerIngress(notary)
	nt.logger = utils.NewRegistryLogger(*nt, nt.ingress.Namespace, nt.ingress.Name)

	req := types.NamespacedName{Name: nt.ingress.Name, Namespace: nt.ingress.Namespace}

	if err := c.Get(context.TODO(), req, nt.ingress); err != nil {
		nt.logger.Error(err, "Get notary server ingress is failed")
		return err

	}

	return nil
}

func (nt *NotaryServerIngress) delete(c client.Client, patchNotary *regv1.Notary) error {
	var err error = nil
	condition := &status.Condition{
		Status: corev1.ConditionFalse,
		Type:   regv1.ConditionTypeNotaryServerIngress,
	}

	defer utils.SetCondition(err, patchNotary, condition)

	if err = c.Delete(context.TODO(), nt.ingress); err != nil {
		nt.logger.Error(err, "Unknown error delete ingress")
		return err
	}

	return nil
}
