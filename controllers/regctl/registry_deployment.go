package regctl

import (
	"context"
	"path"
	"strings"

	"github.com/tmax-cloud/registry-operator/internal/common/config"
	"github.com/tmax-cloud/registry-operator/internal/schemes"
	"github.com/tmax-cloud/registry-operator/internal/utils"

	regv1 "github.com/tmax-cloud/registry-operator/api/v1"
	"github.com/tmax-cloud/registry-operator/controllers/keycloakctl"

	"github.com/operator-framework/operator-lib/status"
	appsv1 "k8s.io/api/apps/v1"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	MountPathDiffKey = "MountPath"
	PvcNameDiffKey   = "PvcName"
	ImageDiffKey     = "Image"
)

type RegistryDeployment struct {
	KcCli  *keycloakctl.KeycloakClient
	deploy *appsv1.Deployment
	logger *utils.RegistryLogger
}

func (r *RegistryDeployment) Handle(c client.Client, reg *regv1.Registry, patchReg *regv1.Registry, scheme *runtime.Scheme) error {
	if err := r.get(c, reg); err != nil {
		if errors.IsNotFound(err) {
			if err := r.create(c, reg, patchReg, scheme); err != nil {
				r.logger.Error(err, "create Deployment error")
				return err
			}
		} else {
			r.logger.Error(err, "Deployment error")
			return err
		}
	}

	r.logger.Info("Check if patch exists.")
	diff := r.compare(reg)
	if diff == nil {
		r.logger.Error(nil, "Invalid deployment!!!")
		if err := r.delete(c, patchReg); err != nil {
			return err
		}
	} else if len(diff) > 0 {
		if err := r.patch(c, reg, patchReg, diff); err != nil {
			return err
		}
	}

	return nil
}

func (r *RegistryDeployment) Ready(c client.Client, reg *regv1.Registry, patchReg *regv1.Registry, useGet bool) error {
	var err error = nil
	condition := &status.Condition{
		Status: corev1.ConditionFalse,
		Type:   regv1.ConditionTypeDeployment,
	}
	defer utils.SetCondition(err, patchReg, condition)
	if useGet {
		err = r.get(c, reg)
		if err != nil {
			r.logger.Error(err, "Deployment error")
			return err
		}
	}

	if r.deploy == nil {
		r.logger.Info("NotReady")

		err = regv1.MakeRegistryError("NotReady")
		return err
	}

	r.logger.Info("Ready")
	condition.Status = corev1.ConditionTrue
	return nil
}

func (r *RegistryDeployment) create(c client.Client, reg *regv1.Registry, patchReg *regv1.Registry, scheme *runtime.Scheme) error {
	if err := controllerutil.SetControllerReference(reg, r.deploy, scheme); err != nil {
		r.logger.Error(err, "SetOwnerReference Failed")
		condition := status.Condition{
			Status:  corev1.ConditionFalse,
			Type:    regv1.ConditionTypeDeployment,
			Message: err.Error(),
		}

		patchReg.Status.Conditions.SetCondition(condition)
		return nil
	}

	r.logger.Info("Create registry deployment")
	err := c.Create(context.TODO(), r.deploy)
	if err != nil {
		condition := status.Condition{
			Status:  corev1.ConditionFalse,
			Type:    regv1.ConditionTypeDeployment,
			Message: err.Error(),
		}

		patchReg.Status.Conditions.SetCondition(condition)
		r.logger.Error(err, "Creating registry deployment is failed.")
		return nil
	}

	return nil
}

func (r *RegistryDeployment) getAuthConfig() *regv1.AuthConfig {
	KeycloakServer := config.Config.GetString(config.ConfigKeycloakService)
	auth := &regv1.AuthConfig{}
	auth.Realm = KeycloakServer + "/" + path.Join("auth", "realms", r.KcCli.GetRealm(), "protocol", "docker-v2", "auth")
	auth.Service = r.KcCli.GetService()
	auth.Issuer = KeycloakServer + "/" + path.Join("auth", "realms", r.KcCli.GetRealm())

	return auth
}

// func (r *RegistryDeployment) getToken(c client.Client, reg *regv1.Registry) (string, error) {
// 	// get token
// 	scopes := []string{"registry:catalog:*"}
// 	token, err := r.KcCli.GetToken(scopes)
// 	if err != nil {
// 		return "", err
// 	}

// 	return token, nil
// }

func (r *RegistryDeployment) get(c client.Client, reg *regv1.Registry) error {
	r.logger = utils.NewRegistryLogger(*r, reg.Namespace, schemes.SubresourceName(reg, schemes.SubTypeRegistryDeployment))
	deploy, err := schemes.Deployment(reg, r.getAuthConfig())
	if err != nil {
		r.logger.Error(err, "Get regsitry deployment scheme is failed")
		return err
	}

	r.deploy = deploy

	req := types.NamespacedName{Name: r.deploy.Name, Namespace: r.deploy.Namespace}

	if err := c.Get(context.TODO(), req, r.deploy); err != nil {
		r.logger.Error(err, "Get regsitry deployment is failed")
		return err
	}

	return nil
}

func (r *RegistryDeployment) patch(c client.Client, reg *regv1.Registry, patchReg *regv1.Registry, diff []utils.Diff) error {
	target := r.deploy.DeepCopy()
	originObject := client.MergeFrom(r.deploy)

	var deployContainer *corev1.Container = nil
	// var contPvcVm *corev1.VolumeMount = nil
	volumeMap := map[string]corev1.Volume{}
	podSpec := target.Spec.Template.Spec

	r.logger.Info("Get", "Patch Keys", strings.Join(utils.DiffKeyList(diff), ", "))

	// Get registry container
	for i, cont := range podSpec.Containers {
		if cont.Name == "registry" {
			deployContainer = &podSpec.Containers[i]
			break
		}
	}

	if deployContainer == nil {
		r.logger.Error(regv1.MakeRegistryError(regv1.ContainerNotFound), "registry container is nil")
		return nil
	}

	for _, d := range diff {
		switch d.Key {
		case ImageDiffKey:
			if reg.Spec.Image == "" {
				deployContainer.Image = config.Config.GetString(config.ConfigRegistryImage)
				continue
			}

			deployContainer.Image = reg.Spec.Image

		case MountPathDiffKey:
			found := false
			for i, vm := range deployContainer.VolumeMounts {
				if vm.Name == "registry" {
					found = true

					if len(reg.Spec.PersistentVolumeClaim.MountPath) == 0 {
						deployContainer.VolumeMounts[i].MountPath = schemes.RegistryPVCMountPath
						break
					}

					deployContainer.VolumeMounts[i].MountPath = reg.Spec.PersistentVolumeClaim.MountPath
					break
				}
			}

			if !found {
				r.logger.Error(regv1.MakeRegistryError(regv1.PvcVolumeMountNotFound), "registry pvc volume mount is nil")
				return nil
			}

		case PvcNameDiffKey:
			// Get volumes
			for _, vol := range podSpec.Volumes {
				volumeMap[vol.Name] = vol
			}

			vol := volumeMap["registry"]
			if reg.Spec.PersistentVolumeClaim.Create != nil {
				vol.PersistentVolumeClaim.ClaimName = regv1.K8sPrefix + reg.Name
			} else {
				vol.PersistentVolumeClaim.ClaimName = reg.Spec.PersistentVolumeClaim.Exist.PvcName
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

func (r *RegistryDeployment) delete(c client.Client, patchReg *regv1.Registry) error {
	if err := c.Delete(context.TODO(), r.deploy); err != nil {
		r.logger.Error(err, "Unknown error delete deployment")
		return err
	}

	condition := status.Condition{
		Type:   regv1.ConditionTypeDeployment,
		Status: corev1.ConditionFalse,
	}

	patchReg.Status.Conditions.SetCondition(condition)

	return nil
}

func (r *RegistryDeployment) compare(reg *regv1.Registry) []utils.Diff {
	diff := []utils.Diff{}
	var deployContainer *corev1.Container = nil
	podSpec := r.deploy.Spec.Template.Spec
	volumeMap := map[string]corev1.Volume{}

	// Get registry container
	for _, cont := range podSpec.Containers {
		if cont.Name == "registry" {
			deployContainer = &cont
		}
	}

	if deployContainer == nil {
		r.logger.Error(regv1.MakeRegistryError(regv1.ContainerNotFound), "registry container is nil")
		return nil
	}

	// Get volumes
	for _, vol := range podSpec.Volumes {
		volumeMap[vol.Name] = vol
	}

	if (reg.Spec.Image != "" && reg.Spec.Image != deployContainer.Image) || (reg.Spec.Image == "" && deployContainer.Image != config.Config.GetString(config.ConfigRegistryImage)) {
		diff = append(diff, utils.Diff{Type: utils.Replace, Key: ImageDiffKey})
	}

	if reg.Spec.PersistentVolumeClaim.Create != nil {
		vol, exist := volumeMap["registry"]
		if !exist {
			r.logger.Info("Registry volume is not exist.")
		} else if vol.VolumeSource.PersistentVolumeClaim.ClaimName != (regv1.K8sPrefix + reg.Name) {
			diff = append(diff, utils.Diff{Type: utils.Replace, Key: PvcNameDiffKey})
		}
	} else {
		vol, exist := volumeMap["registry"]
		if !exist {
			r.logger.Info("Registry volume is not exist.")
		} else if vol.VolumeSource.PersistentVolumeClaim.ClaimName != reg.Spec.PersistentVolumeClaim.Exist.PvcName {
			diff = append(diff, utils.Diff{Type: utils.Replace, Key: PvcNameDiffKey})
		}
	}

	var contPvcVm *corev1.VolumeMount = nil
	for i, vm := range deployContainer.VolumeMounts {
		if vm.Name == "registry" {
			contPvcVm = &deployContainer.VolumeMounts[i]
			break
		}
	}

	if contPvcVm == nil {
		r.logger.Error(regv1.MakeRegistryError(regv1.PvcVolumeMountNotFound), "registry pvc volume mount is nil")
		return nil
	}

	if reg.Spec.PersistentVolumeClaim.MountPath != contPvcVm.MountPath {
		diff = append(diff, utils.Diff{Type: utils.Replace, Key: MountPathDiffKey})
	}

	return diff
}
