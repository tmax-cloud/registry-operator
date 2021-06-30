package regctl

import (
	"context"
	"github.com/go-logr/logr"
	"path"
	"time"

	"github.com/operator-framework/operator-lib/status"
	regv1 "github.com/tmax-cloud/registry-operator/api/v1"
	"github.com/tmax-cloud/registry-operator/controllers/keycloakctl"
	"github.com/tmax-cloud/registry-operator/internal/common/config"
	"github.com/tmax-cloud/registry-operator/internal/schemes"
	"github.com/tmax-cloud/registry-operator/internal/utils"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// NewRegistryNotary creates new registry notary controller
func NewRegistryNotary(client client.Client, scheme *runtime.Scheme, cond status.ConditionType, logger logr.Logger, kcCtl *keycloakctl.KeycloakController) *RegistryNotary {
	return &RegistryNotary{
		c:      client,
		scheme: scheme,
		cond:   cond,
		logger: logger.WithName("Notary"),
		kcCtl:  kcCtl,
	}
}

// RegistryNotary contains things to handle notary resource
type RegistryNotary struct {
	c      client.Client
	scheme *runtime.Scheme
	cond   status.ConditionType
	kcCtl  *keycloakctl.KeycloakController
	not    *regv1.Notary
	logger logr.Logger
}

func (r *RegistryNotary) mustCreated(reg *regv1.Registry) bool {
	return reg.Status.Conditions.GetCondition(regv1.ConditionTypeNotary) != nil
}

// Handle makes notary to be in the desired state
func (r *RegistryNotary) CreateIfNotExist(reg *regv1.Registry, patchReg *regv1.Registry) error {
	if !r.mustCreated(reg) {
		if err := r.get(reg); err != nil {
			return nil
		}
		if err := r.delete(reg); err != nil {
			r.logger.Error(err, "failed to delete notary")
			return err
		}
		return nil
	}

	if err := r.get(reg); err != nil {
		r.notReady(patchReg, err)
		if errors.IsNotFound(err) {
			if err := r.create(reg, patchReg); err != nil {
				r.logger.Error(err, "create notary error")
				r.notReady(patchReg, err)
				return err
			}
			r.logger.Info("Create Succeeded")
		} else {
			r.logger.Error(err, "notary is error")
			return err
		}
		return nil
	}

	r.logger.Info("Check if patch exists.")
	diff := r.compare(reg)
	if len(diff) > 0 {
		r.logger.Info("patch exists.")
		r.notReady(patchReg, nil)
		if err := r.patch(reg, patchReg, diff); err != nil {
			r.notReady(patchReg, err)
			return err
		}
	}

	return nil
}

// Ready checks that notary is ready
func (r *RegistryNotary) IsReady(reg *regv1.Registry, patchReg *regv1.Registry, useGet bool) error {
	if !r.mustCreated(reg) {
		return nil
	}

	var err error = nil
	condition := &status.Condition{
		Status: corev1.ConditionFalse,
		Type:   regv1.ConditionTypeNotary,
	}

	defer utils.SetErrorConditionIfChanged(patchReg, reg, condition, err)

	if r.not == nil || useGet {
		err := r.get(reg)
		if err != nil {
			r.logger.Error(err, "notary error")
			return err
		}
	}

	if r.not.Status.NotaryURL == "" {
		return regv1.MakeRegistryError("NotReady")
	}

	patchReg.Status.NotaryURL = r.not.Status.NotaryURL
	condition.Status = corev1.ConditionTrue

	r.logger.Info("Ready")
	return nil
}

func (r *RegistryNotary) create(reg *regv1.Registry, patchReg *regv1.Registry) error {
	if reg.Spec.PersistentVolumeClaim.Exist != nil {
		r.logger.Info("Use exist registry notary. Need not to create notary.")
		return nil
	}

	if reg.Spec.PersistentVolumeClaim.Create.DeleteWithPvc {
		if err := controllerutil.SetControllerReference(reg, r.not, r.scheme); err != nil {
			r.logger.Error(err, "SetOwnerReference Failed")
			condition := status.Condition{
				Status:  corev1.ConditionFalse,
				Type:    regv1.ConditionTypeNotary,
				Message: err.Error(),
			}

			patchReg.Status.Conditions.SetCondition(condition)
			return err
		}
	}

	r.logger.Info("Create registry notary")
	err := r.c.Create(context.TODO(), r.not)
	if err != nil {
		condition := status.Condition{
			Status:  corev1.ConditionFalse,
			Type:    regv1.ConditionTypeNotary,
			Message: err.Error(),
		}

		patchReg.Status.Conditions.SetCondition(condition)
		r.logger.Error(err, "Creating registry notary is failed.")
		return err
	}

	return nil
}

func (r *RegistryNotary) getAuthConfig() *regv1.AuthConfig {
	auth := &regv1.AuthConfig{}
	KeycloakServer := config.Config.GetString(config.ConfigKeycloakService)
	auth.Realm = KeycloakServer + "/" + path.Join("auth", "realms", r.kcCtl.GetRealmName(), "protocol", "docker-v2", "auth")
	auth.Service = r.kcCtl.GetDockerV2ClientName()
	auth.Issuer = KeycloakServer + "/" + path.Join("auth", "realms", r.kcCtl.GetRealmName())

	return auth
}

func (r *RegistryNotary) get(reg *regv1.Registry) error {
	not, err := schemes.Notary(reg, r.getAuthConfig())
	if err != nil {
		r.logger.Error(err, "Get regsitry notary is failed")
		return err
	}
	r.not = not

	req := types.NamespacedName{Name: r.not.Name, Namespace: r.not.Namespace}
	if err := r.c.Get(context.TODO(), req, r.not); err != nil {
		r.logger.Error(err, "Get regsitry notary is failed")
		return err
	}

	return nil
}

func (r *RegistryNotary) patch(reg *regv1.Registry, patchReg *regv1.Registry, diff []utils.Diff) error {
	return nil
}

func (r *RegistryNotary) delete(patchReg *regv1.Registry) error {
	if err := r.c.Delete(context.TODO(), r.not); err != nil {
		r.logger.Error(err, "Unknown error delete notary")
		return err
	}

	return nil
}

func (r *RegistryNotary) compare(reg *regv1.Registry) []utils.Diff {
	return nil
}

// IsSuccessfullyCompleted returns true if condition is satisfied
func (r *RegistryNotary) IsSuccessfullyCompleted(reg *regv1.Registry) bool {
	cond := reg.Status.Conditions.GetCondition(regv1.ConditionTypeNotary)
	if cond == nil {
		return false
	}

	return cond.IsTrue()
}

func (r *RegistryNotary) notReady(patchReg *regv1.Registry, err error) {
	condition := &status.Condition{
		Status: corev1.ConditionFalse,
		Type:   regv1.ConditionTypeNotary,
	}
	utils.SetCondition(err, patchReg, condition)
}

// Condition returns dependent subresource's condition type
func (r *RegistryNotary) Condition() string {
	return string(regv1.ConditionTypeNotary)
}

// ModifiedTime returns the modified time of the subresource condition
func (r *RegistryNotary) ModifiedTime(patchReg *regv1.Registry) []time.Time {
	cond := patchReg.Status.Conditions.GetCondition(regv1.ConditionTypeNotary)
	if cond == nil {
		return nil
	}

	return []time.Time{cond.LastTransitionTime.Time}
}
