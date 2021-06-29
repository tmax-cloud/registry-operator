package regctl

import (
	"context"
	"fmt"
	"path"
	"time"

	"github.com/tmax-cloud/registry-operator/internal/common/config"
	"github.com/tmax-cloud/registry-operator/internal/schemes"
	"github.com/tmax-cloud/registry-operator/internal/utils"

	regv1 "github.com/tmax-cloud/registry-operator/api/v1"
	"github.com/tmax-cloud/registry-operator/controllers/keycloakctl"

	"github.com/operator-framework/operator-lib/status"
	appsv1 "k8s.io/api/apps/v1"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	readOnlyDiffKey      = "ReadOnly"
	mountPathDiffKey     = "MountPath"
	pvcNameDiffKey       = "PvcName"
	imageDiffKey         = "Image"
	limitCPUDiffKey      = "limitCPU"
	limitMemoryDiffKey   = "limitMemory"
	requestCPUDiffKey    = "requestCPU"
	requestMemoryDiffKey = "requestMemory"
)

// NewRegistryDeployment creates new registry deployment controller
// deps: pvc, svc, cm
func NewRegistryDeployment(client client.Client, scheme *runtime.Scheme, kcCli *keycloakctl.KeycloakClient, deps ...Dependent) *RegistryDeployment {
	return &RegistryDeployment{
		c:      client,
		scheme: scheme,
		deps:   deps,
		KcCli:  kcCli,
	}
}

// RegistryDeployment contains things to handle deployment resource
type RegistryDeployment struct {
	c      client.Client
	scheme *runtime.Scheme
	deps   []Dependent
	KcCli  *keycloakctl.KeycloakClient
	deploy *appsv1.Deployment
	logger *utils.RegistryLogger
}

// Handle makes deployment to be in the desired state
func (r *RegistryDeployment) CreateIfNotExist(reg *regv1.Registry, patchReg *regv1.Registry) error {
	for _, dep := range r.deps {
		if !dep.IsSuccessfullyCompleted(reg) {
			err := fmt.Errorf("unable to handle %s: %s condition is not satisfied", r.Condition(), dep.Condition())
			return err
		}
	}

	if err := r.get(reg); err != nil {
		r.notReady(patchReg, err)
		if errors.IsNotFound(err) {
			if err := r.create(reg, patchReg); err != nil {
				r.logger.Error(err, "create Deployment error")
				r.notReady(patchReg, err)
				return err
			}
			r.logger.Info("Create Succeeded")
		} else {
			r.logger.Error(err, "Deployment error")
			return err
		}
		return nil
	}

	r.logger.Info("Check if patch exists.")
	diff := r.compare(reg)
	if diff == nil {
		r.logger.Error(nil, "Invalid deployment!!!")
		r.notReady(patchReg, nil)
		if err := r.delete(patchReg); err != nil {
			r.notReady(patchReg, appsv1.ErrIntOverflowGenerated)
			return err
		}
	} else if len(diff) > 0 {
		r.notReady(patchReg, nil)
		if err := r.patch(reg, patchReg, diff); err != nil {
			r.notReady(patchReg, err)
			return err
		}
	}

	return nil
}

// Ready checks that deployment is ready
func (r *RegistryDeployment) IsReady(reg *regv1.Registry, patchReg *regv1.Registry, useGet bool) error {
	var err error = nil
	condition := &status.Condition{
		Status: corev1.ConditionFalse,
		Type:   regv1.ConditionTypeDeployment,
	}
	defer utils.SetErrorConditionIfChanged(patchReg, reg, condition, err)
	if useGet {
		err = r.get(reg)
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

	diff := r.compare(reg)
	if diff == nil {
		r.logger.Error(nil, "Invalid deployment!!!")
		if err := r.delete(patchReg); err != nil {
			return err
		}
	} else if len(diff) > 0 {
		r.logger.Info("NotReady")
		err = regv1.MakeRegistryError("NotReady")
		return err
	}

	reg.Status.ReadOnly = reg.Spec.ReadOnly

	r.logger.Info("Ready")
	condition.Status = corev1.ConditionTrue
	return nil
}

func (r *RegistryDeployment) create(reg *regv1.Registry, patchReg *regv1.Registry) error {
	if err := controllerutil.SetControllerReference(reg, r.deploy, r.scheme); err != nil {
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
	err := r.c.Create(context.TODO(), r.deploy)
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

func (r *RegistryDeployment) get(reg *regv1.Registry) error {
	r.logger = utils.NewRegistryLogger(*r, reg.Namespace, schemes.SubresourceName(reg, schemes.SubTypeRegistryDeployment))
	deploy, err := schemes.Deployment(reg, r.getAuthConfig())
	if err != nil {
		r.logger.Error(err, "Get regsitry deployment scheme is failed")
		return err
	}
	r.deploy = deploy

	req := types.NamespacedName{Name: r.deploy.Name, Namespace: r.deploy.Namespace}
	if err := r.c.Get(context.TODO(), req, r.deploy); err != nil {
		r.logger.Error(err, "Get regsitry deployment is failed")
		return err
	}

	return nil
}

func (r *RegistryDeployment) patch(reg *regv1.Registry, patchReg *regv1.Registry, diff []utils.Diff) error {
	target := r.deploy.DeepCopy()
	originObject := client.MergeFrom(r.deploy)
	podSpec := target.Spec.Template.Spec

	r.logger.Info("Get", "Patch", fmt.Sprintf("%+v\n", diff))

	// Get registry container
	var deployContainer *corev1.Container = nil
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
		case readOnlyDiffKey:
			switch d.Type {
			case utils.Add:
				deployContainer.Env = append(deployContainer.Env, corev1.EnvVar{
					Name:  schemes.RegistryEnvKeyStorageMaintenance,
					Value: schemes.RegistryEnvValueStorageMaintenance,
				})

			case utils.Replace:
				for i, env := range deployContainer.Env {
					if env.Name == schemes.RegistryEnvKeyStorageMaintenance {
						deployContainer.Env[i].Value = schemes.RegistryEnvValueStorageMaintenance
						break
					}
				}

			case utils.Remove:
				for i, env := range deployContainer.Env {
					if env.Name == schemes.RegistryEnvKeyStorageMaintenance {
						if i == len(deployContainer.Env)-1 {
							deployContainer.Env = deployContainer.Env[:i]
							break
						}

						deployContainer.Env = append(deployContainer.Env[:i], deployContainer.Env[i+1:]...)
						break
					}
				}
			}

		case imageDiffKey:
			deployContainer.Image = d.Value.(string)

		case mountPathDiffKey:
			found := false
			for i, vm := range deployContainer.VolumeMounts {
				if vm.Name == "registry" {
					found = true
					deployContainer.VolumeMounts[i].MountPath = d.Value.(string)
					break
				}
			}

			if !found {
				r.logger.Error(regv1.MakeRegistryError(regv1.PvcVolumeMountNotFound), "registry pvc volume mount is nil")
			}

		case pvcNameDiffKey:
			found := false
			for i, vol := range podSpec.Volumes {
				if vol.Name == "registry" {
					found = true
					podSpec.Volumes[i].PersistentVolumeClaim.ClaimName = d.Value.(string)
					break
				}
			}

			if !found {
				r.logger.Error(regv1.MakeRegistryError(regv1.PvcVolumeNotFound), "registry pvc volume is nil")
			}

		case limitCPUDiffKey:
			if deployContainer.Resources.Limits == nil {
				deployContainer.Resources.Limits = corev1.ResourceList{}
			}
			deployContainer.Resources.Limits[corev1.ResourceCPU] = d.Value.(resource.Quantity)

		case limitMemoryDiffKey:
			if deployContainer.Resources.Limits == nil {
				deployContainer.Resources.Limits = corev1.ResourceList{}
			}
			deployContainer.Resources.Limits[corev1.ResourceMemory] = d.Value.(resource.Quantity)

		case requestCPUDiffKey:
			if deployContainer.Resources.Requests == nil {
				deployContainer.Resources.Requests = corev1.ResourceList{}
			}
			deployContainer.Resources.Requests[corev1.ResourceCPU] = d.Value.(resource.Quantity)

		case requestMemoryDiffKey:
			if deployContainer.Resources.Requests == nil {
				deployContainer.Resources.Requests = corev1.ResourceList{}
			}
			deployContainer.Resources.Requests[corev1.ResourceMemory] = d.Value.(resource.Quantity)
		}
	}

	// Patch
	if err := r.c.Patch(context.TODO(), target, originObject); err != nil {
		r.logger.Error(err, "Unknown error patch")
		return err
	}
	return nil
}

func (r *RegistryDeployment) delete(patchReg *regv1.Registry) error {
	if err := r.c.Delete(context.TODO(), r.deploy); err != nil {
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
	podSpec := r.deploy.Spec.Template.Spec

	// Get registry container
	var deployContainer *corev1.Container = nil
	for _, cont := range podSpec.Containers {
		if cont.Name == "registry" {
			deployContainer = &cont
			break
		}
	}

	if deployContainer == nil {
		r.logger.Error(regv1.MakeRegistryError(regv1.ContainerNotFound), "registry container is nil")
		return nil
	}

	// Diff ReadOnly
	if reg.Spec.ReadOnly {
		var env corev1.EnvVar
		found := false
		for _, env = range deployContainer.Env {
			if env.Name == schemes.RegistryEnvKeyStorageMaintenance {
				found = true
				break
			}
		}

		if !found {
			diff = append(diff, utils.Diff{Type: utils.Add, Key: readOnlyDiffKey})
		} else if env.Value != schemes.RegistryEnvValueStorageMaintenance {
			diff = append(diff, utils.Diff{Type: utils.Replace, Key: readOnlyDiffKey, Value: schemes.RegistryEnvValueStorageMaintenance})
		}

	} else {
		found := false
		for _, env := range deployContainer.Env {
			if env.Name == schemes.RegistryEnvKeyStorageMaintenance {
				found = true
				break
			}
		}

		if found {
			diff = append(diff, utils.Diff{Type: utils.Remove, Key: readOnlyDiffKey})
		}
	}

	// Diff Image
	regImage := reg.Spec.Image
	if regImage == "" {
		regImage = config.Config.GetString(config.ConfigRegistryImage)
	}

	if regImage != deployContainer.Image {
		diff = append(diff, utils.Diff{Type: utils.Replace, Key: imageDiffKey, Value: regImage})
	}

	// Diff volumes
	volumeName := ""
	if reg.Spec.PersistentVolumeClaim.Exist != nil {
		volumeName = reg.Spec.PersistentVolumeClaim.Exist.PvcName
	}
	if volumeName == "" {
		volumeName = schemes.SubresourceName(reg, schemes.SubTypeRegistryPVC)
	}

	var deployVolume *corev1.Volume = nil
	for i, vol := range podSpec.Volumes {
		if vol.Name == "registry" {
			deployVolume = &podSpec.Volumes[i]
			break
		}
	}

	if deployVolume == nil {
		r.logger.Error(regv1.MakeRegistryError(regv1.PvcVolumeNotFound), "registry pvc volume mount is nil ")
	} else if deployVolume.VolumeSource.PersistentVolumeClaim.ClaimName != volumeName {
		diff = append(diff, utils.Diff{Type: utils.Replace, Key: pvcNameDiffKey, Value: volumeName})
	}

	// Diff Volume Mount
	mountPath := reg.Spec.PersistentVolumeClaim.MountPath
	if mountPath == "" {
		mountPath = schemes.RegistryPVCMountPath
	}

	var contPvcVM *corev1.VolumeMount = nil
	for i, vm := range deployContainer.VolumeMounts {
		if vm.Name == "registry" {
			contPvcVM = &deployContainer.VolumeMounts[i]
			break
		}
	}
	if contPvcVM == nil {
		r.logger.Error(regv1.MakeRegistryError(regv1.PvcVolumeMountNotFound), "registry pvc volume mount is nil ")
	} else if contPvcVM.MountPath != mountPath {
		diff = append(diff, utils.Diff{Type: utils.Replace, Key: mountPathDiffKey, Value: mountPath})
	}

	// Diff Resource Requirement
	regLitmitCPU := *reg.Spec.RegistryDeployment.Resources.Limits.Cpu()
	regLitmitMemory := *reg.Spec.RegistryDeployment.Resources.Limits.Memory()
	regRequestCPU := *reg.Spec.RegistryDeployment.Resources.Requests.Cpu()
	regRequestMemory := *reg.Spec.RegistryDeployment.Resources.Requests.Memory()

	if regLitmitCPU.IsZero() {
		regLitmitCPU = resource.MustParse(config.Config.GetString(config.ConfigRegistryCPU))
	}
	if regLitmitMemory.IsZero() {
		regLitmitMemory = resource.MustParse(config.Config.GetString(config.ConfigRegistryMemory))
	}
	if regRequestCPU.IsZero() {
		regRequestCPU = resource.MustParse(config.Config.GetString(config.ConfigRegistryCPU))
	}
	if regRequestMemory.IsZero() {
		regRequestMemory = resource.MustParse(config.Config.GetString(config.ConfigRegistryMemory))
	}

	if !deployContainer.Resources.Limits.Cpu().Equal(regLitmitCPU) {
		diff = append(diff, utils.Diff{Type: utils.Replace, Key: limitCPUDiffKey, Value: regLitmitCPU})
	}
	if !deployContainer.Resources.Limits.Memory().Equal(regLitmitMemory) {
		diff = append(diff, utils.Diff{Type: utils.Replace, Key: limitMemoryDiffKey, Value: regLitmitMemory})
	}
	if !deployContainer.Resources.Requests.Cpu().Equal(regRequestCPU) {
		diff = append(diff, utils.Diff{Type: utils.Replace, Key: requestCPUDiffKey, Value: regRequestCPU})
	}
	if !deployContainer.Resources.Requests.Memory().Equal(regRequestMemory) {
		diff = append(diff, utils.Diff{Type: utils.Replace, Key: requestMemoryDiffKey, Value: regRequestMemory})
	}

	return diff
}

// IsSuccessfullyCompleted returns true if condition is satisfied
func (r *RegistryDeployment) IsSuccessfullyCompleted(reg *regv1.Registry) bool {
	cond := reg.Status.Conditions.GetCondition(regv1.ConditionTypeDeployment)
	if cond == nil {
		return false
	}

	return cond.IsTrue()
}

func (r *RegistryDeployment) notReady(patchReg *regv1.Registry, err error) {
	condition := &status.Condition{
		Status: corev1.ConditionFalse,
		Type:   regv1.ConditionTypeDeployment,
	}
	utils.SetCondition(err, patchReg, condition)
}

// Condition returns dependent subresource's condition type
func (r *RegistryDeployment) Condition() string {
	return string(regv1.ConditionTypeDeployment)
}

// ModifiedTime returns the modified time of the subresource condition
func (r *RegistryDeployment) ModifiedTime(patchReg *regv1.Registry) []time.Time {
	cond := patchReg.Status.Conditions.GetCondition(regv1.ConditionTypeDeployment)
	if cond == nil {
		return nil
	}

	return []time.Time{cond.LastTransitionTime.Time}
}
