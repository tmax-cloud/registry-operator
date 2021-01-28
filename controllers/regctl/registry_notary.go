package regctl

import (
	"context"
	"path"

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

type RegistryNotary struct {
	KcCtl  *keycloakctl.KeycloakController
	not    *regv1.Notary
	logger *utils.RegistryLogger
}

func (r *RegistryNotary) Handle(c client.Client, reg *regv1.Registry, patchReg *regv1.Registry, scheme *runtime.Scheme) error {
	if err := r.get(c, reg); err != nil {
		if errors.IsNotFound(err) {
			if err := r.create(c, reg, patchReg, scheme); err != nil {
				r.logger.Error(err, "create notary error")
				return err
			}
		} else {
			r.logger.Error(err, "notary is error")
			return err
		}
	}

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

func (r *RegistryNotary) Ready(c client.Client, reg *regv1.Registry, patchReg *regv1.Registry, useGet bool) error {
	var err error = nil
	condition := &status.Condition{
		Status: corev1.ConditionFalse,
		Type:   regv1.ConditionTypeNotary,
	}

	defer utils.SetCondition(err, patchReg, condition)

	if r.not == nil || useGet {
		err := r.get(c, reg)
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

func (r *RegistryNotary) create(c client.Client, reg *regv1.Registry, patchReg *regv1.Registry, scheme *runtime.Scheme) error {
	if reg.Spec.PersistentVolumeClaim.Exist != nil {
		r.logger.Info("Use exist registry notary. Need not to create notary.")
		return nil
	}

	if reg.Spec.PersistentVolumeClaim.Create.DeleteWithPvc {
		if err := controllerutil.SetControllerReference(reg, r.not, scheme); err != nil {
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
	err := c.Create(context.TODO(), r.not)
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
	auth.Realm = KeycloakServer + "/" + path.Join("auth", "realms", r.KcCtl.GetRealmName(), "protocol", "docker-v2", "auth")
	auth.Service = r.KcCtl.GetDockerV2ClientName()
	auth.Issuer = KeycloakServer + "/" + path.Join("auth", "realms", r.KcCtl.GetRealmName())

	return auth
}

func (r *RegistryNotary) get(c client.Client, reg *regv1.Registry) error {
	r.logger = utils.NewRegistryLogger(*r, reg.Namespace, schemes.SubresourceName(reg, schemes.SubTypeRegistryNotary))
	not, err := schemes.Notary(reg, r.getAuthConfig())
	if err != nil {
		r.logger.Error(err, "Get regsitry notary is failed")
		return err
	}
	r.not = not

	req := types.NamespacedName{Name: r.not.Name, Namespace: r.not.Namespace}
	if err := c.Get(context.TODO(), req, r.not); err != nil {
		r.logger.Error(err, "Get regsitry notary is failed")
		return err
	}

	return nil
}

func (r *RegistryNotary) patch(c client.Client, reg *regv1.Registry, patchReg *regv1.Registry, diff []utils.Diff) error {
	return nil
}

func (r *RegistryNotary) delete(c client.Client, patchReg *regv1.Registry) error {
	if err := c.Delete(context.TODO(), r.not); err != nil {
		r.logger.Error(err, "Unknown error delete notary")
		return err
	}

	condition := status.Condition{
		Type:   regv1.ConditionTypeNotary,
		Status: corev1.ConditionFalse,
	}

	patchReg.Status.Conditions.SetCondition(condition)
	return nil
}

func (r *RegistryNotary) compare(reg *regv1.Registry) []utils.Diff {
	return nil
}
