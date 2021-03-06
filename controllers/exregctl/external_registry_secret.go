package exregctl

import (
	"context"
	"errors"

	"github.com/operator-framework/operator-lib/status"
	regv1 "github.com/tmax-cloud/registry-operator/api/v1"
	"github.com/tmax-cloud/registry-operator/internal/schemes"
	"github.com/tmax-cloud/registry-operator/internal/utils"
	"github.com/tmax-cloud/registry-operator/pkg/image"
	corev1 "k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	newLoginSecretDiffKey = "NewLoginSecret"
)

// LoginSecret ...
type LoginSecret struct {
	secret *corev1.Secret
	logger *utils.RegistryLogger
}

// Handle creates login secret if not exists
func (r *LoginSecret) Handle(c client.Client, exreg *regv1.ExternalRegistry, patchExreg *regv1.ExternalRegistry, scheme *runtime.Scheme) error {
	if err := r.get(c, exreg); err != nil {
		if k8serr.IsNotFound(err) {
			_, _ = r.initCondition(c, exreg, patchExreg, false)
			if err := r.create(c, exreg, patchExreg, scheme); err != nil {
				r.logger.Error(err, "create external registry login secret error")
				return err
			}
		} else {
			r.logger.Error(err, "external registry login secret error")
			return err
		}
	}

	r.logger.Info("Check if patch exists.")
	diff := r.compare(exreg)
	if len(diff) > 0 {
		if err := r.patch(c, exreg, patchExreg, diff); err != nil {
			return err
		}
	}

	return nil
}

// Ready is to check if the external registry login secret is ready
func (r *LoginSecret) Ready(c client.Client, exreg *regv1.ExternalRegistry, patchExreg *regv1.ExternalRegistry, useGet bool) error {
	condition, err := r.initCondition(c, exreg, patchExreg, false)
	if err != nil {
		return err
	}

	defer utils.SetCondition(err, patchExreg, condition)

	r.logger.Info("Ready")

	patchExreg.Spec.LoginID = ""
	patchExreg.Spec.LoginPassword = ""
	if exreg.Spec.RegistryType == regv1.RegistryTypeDockerHub {
		patchExreg.Spec.RegistryURL = image.DefaultServer
	}
	if r.secret != nil {
		patchExreg.Status.LoginSecret = r.secret.Name
	}

	condition.Status = corev1.ConditionTrue
	return nil
}

func (r *LoginSecret) create(c client.Client, exreg *regv1.ExternalRegistry, patchExreg *regv1.ExternalRegistry, scheme *runtime.Scheme) error {
	if exreg.Spec.LoginID == "" && exreg.Spec.LoginPassword == "" {
		return errors.New("login info is empty")
	}

	secret, err := schemes.ExternalRegistryLoginSecret(exreg)
	if err != nil {
		r.logger.Error(err, "failed to get secret scheme")
		return err
	}
	r.secret = secret

	if err := controllerutil.SetControllerReference(exreg, r.secret, scheme); err != nil {
		r.logger.Error(err, "SetOwnerReference Failed")
		return err
	}

	r.logger.Info("Create external registry secret")
	if err := c.Create(context.TODO(), r.secret); err != nil {
		r.logger.Error(err, "Creating external registry login secret is failed.")
		return err
	}

	return nil
}

func (r *LoginSecret) get(c client.Client, exreg *regv1.ExternalRegistry) error {
	r.logger = utils.NewRegistryLogger(*r, exreg.Namespace, schemes.SubresourceName(exreg, schemes.SubTypeExternalRegistryLoginSecret))
	secret, err := schemes.ExternalRegistryLoginSecret(exreg)
	if err != nil {
		r.logger.Error(err, "failed to get secret")
		return err
	}
	r.secret = secret

	req := types.NamespacedName{Name: r.secret.Name, Namespace: r.secret.Namespace}
	if err := c.Get(context.TODO(), req, r.secret); err != nil {
		r.logger.Error(err, "failed to get secret")
		r.secret = nil
		return err
	}

	return nil
}

func (r *LoginSecret) compare(exreg *regv1.ExternalRegistry) []utils.Diff {
	diff := []utils.Diff{}
	if exreg.Spec.LoginID != "" && exreg.Spec.LoginPassword != "" {
		diff = append(diff, utils.Diff{Type: utils.Replace, Key: newLoginSecretDiffKey})
	}

	return diff
}

func (r *LoginSecret) patch(c client.Client, exreg *regv1.ExternalRegistry, patchExreg *regv1.ExternalRegistry, diff []utils.Diff) error {
	target := r.secret.DeepCopy()
	originObject := client.MergeFrom(r.secret)

	for _, d := range diff {
		switch d.Key {
		case newLoginSecretDiffKey:
			switch d.Type {
			case utils.Replace:
				secret, err := schemes.ExternalRegistryLoginSecret(exreg)
				if err != nil {
					r.logger.Error(err, "failed to get secret scheme")
					return err
				}
				target.Data = secret.Data
			}
		}
	}

	// Patch
	if err := c.Patch(context.TODO(), target, originObject); err != nil {
		r.logger.Error(err, "Unknown error patch")
		return err
	}

	return nil
}

func (r *LoginSecret) delete(c client.Client, patchExreg *regv1.ExternalRegistry) error {
	if err := c.Delete(context.TODO(), r.secret); err != nil {
		r.logger.Error(err, "Unknown error delete deployment")
		return err
	}

	return nil
}

func (r *LoginSecret) initCondition(c client.Client, exreg *regv1.ExternalRegistry, patchExreg *regv1.ExternalRegistry, useGet bool) (*status.Condition, error) {
	var err error = nil
	condition := &status.Condition{
		Status: corev1.ConditionFalse,
		Type:   regv1.ConditionTypeExRegistryLoginSecretExist,
	}

	defer utils.SetCondition(err, patchExreg, condition)

	if useGet {
		if err = r.get(c, exreg); err != nil {
			r.logger.Error(err, "get external registry login secret error")
			return condition, err
		}
	}

	if r.secret == nil && (exreg.Spec.LoginID == "" || exreg.Spec.LoginPassword == "") {
		err = errors.New("login secret is not found. must enter loginId and loginPassword in spec field")
		r.logger.Error(err, "")
		patchExreg.Status.LoginSecret = ""
		return condition, err
	}

	return condition, nil
}
