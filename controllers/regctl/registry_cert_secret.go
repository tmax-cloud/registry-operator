package regctl

import (
	"context"

	"github.com/operator-framework/operator-lib/status"
	regv1 "github.com/tmax-cloud/registry-operator/api/v1"

	"k8s.io/apimachinery/pkg/api/errors"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/tmax-cloud/registry-operator/internal/schemes"
	"github.com/tmax-cloud/registry-operator/internal/utils"
)

const SecretOpaqueTypeName = regv1.ConditionTypeSecretOpaque
const SecretTLSTypeName = regv1.ConditionTypeSecretTls

type RegistryCertSecret struct {
	secretOpaque *corev1.Secret
	secretTLS    *corev1.Secret
	logger       *utils.RegistryLogger
}

func (r *RegistryCertSecret) Handle(c client.Client, reg *regv1.Registry, patchReg *regv1.Registry, scheme *runtime.Scheme) error {
	err := r.get(c, reg)
	if err != nil && errors.IsNotFound(err) {
		// resource is not exist : have to create
		if createError := r.create(c, reg, patchReg, scheme); createError != nil {
			r.logger.Error(createError, "Create failed in Handle")
			return createError
		}
		// patchReg.Status.PodRecreateRequired = true
	}

	if isValid := r.compare(reg); isValid == nil {
		if deleteError := r.delete(c, patchReg); deleteError != nil {
			r.logger.Error(deleteError, "Delete failed in Handle")
			return deleteError
		}
	}

	r.logger.Info("Succeed")
	return nil
}

func (r *RegistryCertSecret) Ready(c client.Client, reg *regv1.Registry, patchReg *regv1.Registry, useGet bool) error {
	var opaqueErr error = nil
	var err error = nil

	opCondition := &status.Condition{
		Status: corev1.ConditionFalse,
		Type:   SecretOpaqueTypeName,
	}

	tlsCondition := &status.Condition{
		Status: corev1.ConditionFalse,
		Type:   SecretTLSTypeName,
	}

	defer utils.SetCondition(opaqueErr, patchReg, opCondition)

	if useGet {
		if opaqueErr = r.get(c, reg); opaqueErr != nil {
			r.logger.Error(opaqueErr, "Get failed")
			return opaqueErr
		}
	}

	opCondition.Status = corev1.ConditionTrue

	defer utils.SetCondition(err, patchReg, tlsCondition)
	err = regv1.MakeRegistryError("Secret TLS Error")
	if _, ok := r.secretTLS.Data[schemes.TLSCert]; !ok {
		r.logger.Error(err, "No certificate in data")
		return nil
	}

	if _, ok := r.secretTLS.Data[schemes.TLSKey]; !ok {
		r.logger.Error(err, "No private key in data")
		return nil
	}

	tlsCondition.Status = corev1.ConditionTrue
	err = nil
	r.logger.Info("Succeed")
	return nil
}

func (r *RegistryCertSecret) GetUserSecret(c client.Client, reg *regv1.Registry) (username, password string, err error) {
	if opaqueErr := r.get(c, reg); opaqueErr != nil {
		r.logger.Error(opaqueErr, "Get failed")
		err = opaqueErr
		return
	}

	username = string(r.secretOpaque.Data["ID"])
	password = string(r.secretOpaque.Data["PASSWD"])
	return
}

func (r *RegistryCertSecret) create(c client.Client, reg *regv1.Registry, patchReg *regv1.Registry, scheme *runtime.Scheme) error {
	condition := &status.Condition{
		Status: corev1.ConditionFalse,
		Type:   SecretOpaqueTypeName,
	}

	tlsCondition := &status.Condition{
		Status: corev1.ConditionFalse,
		Type:   SecretTLSTypeName,
	}

	if err := controllerutil.SetControllerReference(reg, r.secretOpaque, scheme); err != nil {
		utils.SetCondition(err, patchReg, condition)
		return err
	}

	if err := controllerutil.SetControllerReference(reg, r.secretTLS, scheme); err != nil {
		utils.SetCondition(err, patchReg, tlsCondition)
		return err
	}

	if err := c.Create(context.TODO(), r.secretOpaque); err != nil {
		r.logger.Error(err, "Create failed")
		utils.SetCondition(err, patchReg, condition)
		return err
	}

	if err := c.Create(context.TODO(), r.secretTLS); err != nil {
		r.logger.Error(err, "Create failed")
		utils.SetCondition(err, patchReg, tlsCondition)
		return err
	}

	r.logger.Info("Succeed")
	return nil
}

func (r *RegistryCertSecret) get(c client.Client, reg *regv1.Registry) error {
	r.logger = utils.NewRegistryLogger(*r, reg.Namespace, schemes.SubresourceName(reg, schemes.SubTypeRegistryOpaqueSecret))
	r.secretOpaque, r.secretTLS = schemes.Secrets(reg, c)
	if r.secretOpaque == nil && r.secretTLS == nil {
		return regv1.MakeRegistryError("Registry has no fields Secrets required")
	}

	req := types.NamespacedName{Name: r.secretOpaque.Name, Namespace: r.secretOpaque.Namespace}
	if err := c.Get(context.TODO(), req, r.secretOpaque); err != nil {
		r.logger.Error(err, "Get failed")
		return err
	}

	req = types.NamespacedName{Name: r.secretTLS.Name, Namespace: r.secretTLS.Namespace}
	if err := c.Get(context.TODO(), req, r.secretTLS); err != nil {
		r.logger.Error(err, "Get failed")
		return err
	}

	r.logger.Info("Succeed")
	return nil
}

func (r *RegistryCertSecret) patch(c client.Client, reg *regv1.Registry, patchReg *regv1.Registry, diff []utils.Diff) error {
	// [TODO]
	return nil
}

func (r *RegistryCertSecret) delete(c client.Client, patchReg *regv1.Registry) error {
	condition := &status.Condition{
		Status: corev1.ConditionFalse,
		Type:   SecretOpaqueTypeName,
	}

	tlsCondition := &status.Condition{
		Status: corev1.ConditionFalse,
		Type:   SecretTLSTypeName,
	}

	if err := c.Delete(context.TODO(), r.secretOpaque); err != nil {
		r.logger.Error(err, "Delete failed")
		utils.SetCondition(err, patchReg, condition)
		return err
	}

	if err := c.Delete(context.TODO(), r.secretTLS); err != nil {
		r.logger.Error(err, "Delete failed")
		utils.SetCondition(err, patchReg, tlsCondition)
		return err
	}

	return nil
}

func (r *RegistryCertSecret) compare(reg *regv1.Registry) []utils.Diff {
	return []utils.Diff{}
}
