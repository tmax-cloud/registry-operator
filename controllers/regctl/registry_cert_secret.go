package regctl

import (
	"context"
	"fmt"
	"time"

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

// NewRegistryCertSecret creates new registry cert secret controller
// deps: service
func NewRegistryCertSecret(client client.Client, deps ...Dependent) *RegistryCertSecret {
	return &RegistryCertSecret{
		c:    client,
		deps: deps,
	}
}

// RegistryCertSecret contains things to handle tls and opaque secret resource
type RegistryCertSecret struct {
	c            client.Client
	deps         []Dependent
	secretOpaque *corev1.Secret
	secretTLS    *corev1.Secret
	logger       *utils.RegistryLogger
}

// Handle makes secret to be in the desired state
func (r *RegistryCertSecret) CreateIfNotExist(reg *regv1.Registry, patchReg *regv1.Registry, scheme *runtime.Scheme) error {
	for _, dep := range r.deps {
		if !dep.IsSuccessfullyCompleted(reg) {
			err := fmt.Errorf("unable to handle %s: %s condition is not satisfied", r.Condition(), dep.Condition())
			return err
		}
	}

	if err := r.get(reg); err != nil {
		r.notReady(patchReg, err)
		if errors.IsNotFound(err) {
			if createError := r.create(reg, patchReg, scheme); createError != nil {
				r.logger.Error(createError, "Create failed in CreateIfNotExist")
				r.notReady(patchReg, createError)
				return createError
			}
			r.logger.Info("Create Succeeded")
		} else {
			r.logger.Error(err, "cert secret is error")
			return err
		}
		return nil
	}

	if isValid := r.compare(reg); isValid == nil {
		r.notReady(patchReg, nil)
		if deleteError := r.delete(patchReg); deleteError != nil {
			r.logger.Error(deleteError, "Delete failed in CreateIfNotExist")
			r.notReady(patchReg, deleteError)
			return deleteError
		}
	}

	return nil
}

// Ready checks that secret is ready
func (r *RegistryCertSecret) IsReady(reg *regv1.Registry, patchReg *regv1.Registry, useGet bool) error {
	var opaqueErr error = nil
	var tlsErr error = nil

	opCondition := &status.Condition{
		Status: corev1.ConditionFalse,
		Type:   regv1.ConditionTypeSecretOpaque,
	}

	tlsCondition := &status.Condition{
		Status: corev1.ConditionFalse,
		Type:   regv1.ConditionTypeSecretTLS,
	}

	defer utils.SetErrorConditionIfChanged(patchReg, reg, opCondition, opaqueErr)
	defer utils.SetErrorConditionIfChanged(patchReg, reg, tlsCondition, tlsErr)

	if useGet {
		if opaqueErr = r.get(reg); opaqueErr != nil {
			r.logger.Error(opaqueErr, "Get failed")
			return opaqueErr
		}
	}

	opCondition.Status = corev1.ConditionTrue

	if _, ok := r.secretTLS.Data[schemes.TLSCert]; !ok {
		tlsErr = regv1.MakeRegistryError("Secret TLS Error")
		r.logger.Error(tlsErr, "No certificate in data")
		return nil
	}

	if _, ok := r.secretTLS.Data[schemes.TLSKey]; !ok {
		tlsErr = regv1.MakeRegistryError("Secret TLS Error")
		r.logger.Error(tlsErr, "No private key in data")
		return nil
	}

	tlsCondition.Status = corev1.ConditionTrue
	r.logger.Info("Succeed")
	return nil
}

// GetUserSecret returns username and password
func (r *RegistryCertSecret) GetUserSecret(reg *regv1.Registry) (username, password string, err error) {
	if opaqueErr := r.get(reg); opaqueErr != nil {
		r.logger.Error(opaqueErr, "Get failed")
		err = opaqueErr
		return
	}

	username = string(r.secretOpaque.Data["ID"])
	password = string(r.secretOpaque.Data["PASSWD"])
	return
}

func (r *RegistryCertSecret) create(reg *regv1.Registry, patchReg *regv1.Registry, scheme *runtime.Scheme) error {
	if err := controllerutil.SetControllerReference(reg, r.secretOpaque, scheme); err != nil {
		r.logger.Error(err, "failed to set controller reference")
		return err
	}

	if err := controllerutil.SetControllerReference(reg, r.secretTLS, scheme); err != nil {
		r.logger.Error(err, "failed to set controller reference")
		return err
	}
	if err := r.c.Create(context.TODO(), r.secretOpaque); err != nil {
		r.logger.Error(err, "Create failed")
		return err
	}
	if err := r.c.Create(context.TODO(), r.secretTLS); err != nil {
		r.logger.Error(err, "Create failed")
		return err
	}

	r.logger.Info("Succeed")
	return nil
}

func (r *RegistryCertSecret) get(reg *regv1.Registry) error {
	r.logger = utils.NewRegistryLogger(*r, reg.Namespace, schemes.SubresourceName(reg, schemes.SubTypeRegistryOpaqueSecret))
	r.secretOpaque, r.secretTLS = schemes.Secrets(reg, r.c)
	if r.secretOpaque == nil && r.secretTLS == nil {
		return regv1.MakeRegistryError("Registry has no fields Secrets required")
	}

	req := types.NamespacedName{Name: r.secretOpaque.Name, Namespace: r.secretOpaque.Namespace}
	if err := r.c.Get(context.TODO(), req, r.secretOpaque); err != nil {
		r.logger.Error(err, "Get failed")
		return err
	}

	req = types.NamespacedName{Name: r.secretTLS.Name, Namespace: r.secretTLS.Namespace}
	if err := r.c.Get(context.TODO(), req, r.secretTLS); err != nil {
		r.logger.Error(err, "Get failed")
		return err
	}

	r.logger.Info("Succeed")
	return nil
}

func (r *RegistryCertSecret) patch(reg *regv1.Registry, patchReg *regv1.Registry, diff []utils.Diff) error {
	// [TODO]
	return nil
}

func (r *RegistryCertSecret) delete(patchReg *regv1.Registry) error {
	if err := r.c.Delete(context.TODO(), r.secretOpaque); err != nil {
		r.logger.Error(err, "Delete failed")
		return err
	}

	if err := r.c.Delete(context.TODO(), r.secretTLS); err != nil {
		r.logger.Error(err, "Delete failed")
		return err
	}

	return nil
}

func (r *RegistryCertSecret) compare(reg *regv1.Registry) []utils.Diff {
	diff := []utils.Diff{}
	cond1 := reg.Status.Conditions.GetCondition(regv1.ConditionTypeSecretOpaque)
	cond2 := reg.Status.Conditions.GetCondition(regv1.ConditionTypeSecretTLS)

	for _, dep := range r.deps {
		for _, depTime := range dep.ModifiedTime(reg) {
			if depTime.After(cond1.LastTransitionTime.Time) ||
				depTime.After(cond2.LastTransitionTime.Time) {
				return nil
			}
		}
	}

	return diff
}

// IsSuccessfullyCompleted returns true if condition is satisfied
func (r *RegistryCertSecret) IsSuccessfullyCompleted(reg *regv1.Registry) bool {
	cond1 := reg.Status.Conditions.GetCondition(regv1.ConditionTypeSecretOpaque)
	if cond1 == nil {
		return false
	}

	cond2 := reg.Status.Conditions.GetCondition(regv1.ConditionTypeSecretTLS)
	if cond2 == nil {
		return false
	}

	return cond1.IsTrue() && cond2.IsTrue()
}

func (r *RegistryCertSecret) notReady(patchReg *regv1.Registry, err error) {
	condition := &status.Condition{
		Status: corev1.ConditionFalse,
		Type:   regv1.ConditionTypeSecretOpaque,
	}

	tlsCondition := &status.Condition{
		Status: corev1.ConditionFalse,
		Type:   regv1.ConditionTypeSecretTLS,
	}

	utils.SetCondition(err, patchReg, condition)
	utils.SetCondition(err, patchReg, tlsCondition)
}

// Condition returns dependent subresource's condition type
func (r *RegistryCertSecret) Condition() string {
	return fmt.Sprintf("%s and %s", string(regv1.ConditionTypeSecretOpaque), string(regv1.ConditionTypeSecretTLS))
}

// ModifiedTime returns the modified time of the subresource condition
func (r *RegistryCertSecret) ModifiedTime(patchReg *regv1.Registry) []time.Time {
	cond1 := patchReg.Status.Conditions.GetCondition(regv1.ConditionTypeSecretOpaque)
	if cond1 == nil {
		return nil
	}
	cond2 := patchReg.Status.Conditions.GetCondition(regv1.ConditionTypeSecretTLS)
	if cond2 == nil {
		return nil
	}

	return []time.Time{cond1.LastTransitionTime.Time, cond2.LastTransitionTime.Time}
}
