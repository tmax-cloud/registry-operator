package regctl

import (
	"context"
	"strings"

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

func (nt *NotaryServer) Handle(c client.Client, reg *regv1.Registry, patchReg *regv1.Registry, scheme *runtime.Scheme) error {
	if err := nt.get(c, reg); err != nil {
		if errors.IsNotFound(err) {
			if err := nt.create(c, reg, patchReg, scheme); err != nil {
				nt.logger.Error(err, "create Deployment error")
				return err
			}
		} else {
			nt.logger.Error(err, "Deployment error")
			return err
		}
	}

	nt.logger.Info("Check if patch exists.")
	diff := nt.compare(reg)
	if diff == nil {
		nt.logger.Error(nil, "Invalid deployment!!!")
		nt.delete(c, patchReg)
	} else if len(diff) > 0 {
		nt.patch(c, reg, patchReg, diff)
	}

	return nil
}

func (nt *NotaryServer) Ready(c client.Client, reg *regv1.Registry, patchReg *regv1.Registry, useGet bool) error {
	var err error = nil
	condition := &status.Condition{
		Status: corev1.ConditionFalse,
		Type:   regv1.ConditionTypeDeployment,
	}
	defer utils.SetError(err, patchReg, condition)
	if useGet {
		err = nt.get(c, reg)
		if err != nil {
			nt.logger.Error(err, "Deployment error")
			return err
		}
	}

	if nt.pod == nil {
		nt.logger.Info("NotReady")

		err = regv1.MakeRegistryError("NotReady")
		return err
	}

	nt.logger.Info("Ready")
	condition.Status = corev1.ConditionTrue
	return nil
}

func (nt *NotaryServer) create(c client.Client, reg *regv1.Registry, patchReg *regv1.Registry, scheme *runtime.Scheme) error {
	if err := controllerutil.SetControllerReference(reg, nt.pod, scheme); err != nil {
		nt.logger.Error(err, "SetOwnerReference Failed")
		condition := status.Condition{
			Status:  corev1.ConditionFalse,
			Type:    regv1.ConditionTypeNotaryServer,
			Message: err.Error(),
		}

		patchReg.Status.Conditions.SetCondition(condition)
		return nil
	}

	nt.logger.Info("Create notary server pod")
	err := c.Create(context.TODO(), nt.pod)
	if err != nil {
		condition := status.Condition{
			Status:  corev1.ConditionFalse,
			Type:    regv1.ConditionTypeNotaryServer,
			Message: err.Error(),
		}

		patchReg.Status.Conditions.SetCondition(condition)
		nt.logger.Error(err, "Creating notary server pod is failed.")
		return nil
	}

	return nil
}

func (nt *NotaryServer) get(c client.Client, reg *regv1.Registry) error {
	nt.pod = schemes.Deployment(reg)
	nt.logger = utils.NewRegistryLogger(*nt, nt.pod.Namespace, nt.pod.Name)

	req := types.NamespacedName{Name: nt.pod.Name, Namespace: nt.pod.Namespace}

	err := c.Get(context.TODO(), req, nt.pod)
	if err != nil {
		nt.logger.Error(err, "Get regsitry deployment is failed")
		return err
	}

	return nil
}

func (nt *NotaryServer) patch(c client.Client, reg *regv1.Registry, patchReg *regv1.Registry, diff []utils.Diff) error {
	target := nt.pod.DeepCopy()
	originObject := client.MergeFrom(nt.pod)

	var deployContainer *corev1.Container = nil
	// var contPvcVm *corev1.VolumeMount = nil
	volumeMap := map[string]corev1.Volume{}
	podSpec := target.Spec.Template.Spec

	nt.logger.Info("Get", "Patch Keys", strings.Join(utils.DiffKeyList(diff), ", "))

	// Get registry container
	for i, cont := range podSpec.Containers {
		if cont.Name == "registry" {
			deployContainer = &podSpec.Containers[i]
			break
		}
	}

	if deployContainer == nil {
		nt.logger.Error(regv1.MakeRegistryError(regv1.ContainerNotFound), "registry container is nil")
		return nil
	}

	for _, d := range diff {
		switch d.Key {
		case ImageDiffKey:
			deployContainer.Image = reg.Spec.Image

		case MountPathDiffKey:
			found := false
			for i, vm := range deployContainer.VolumeMounts {
				if vm.Name == "registry" {
					deployContainer.VolumeMounts[i].MountPath = reg.Spec.PersistentVolumeClaim.MountPath
					found = true
					break
				}
			}

			if !found {
				nt.logger.Error(regv1.MakeRegistryError(regv1.PvcVolumeMountNotFound), "registry pvc volume mount is nil")
				return nil
			}

		case PvcNameDiffKey:
			// Get volumes
			for _, vol := range podSpec.Volumes {
				volumeMap[vol.Name] = vol
			}

			vol, _ := volumeMap["registry"]
			if reg.Spec.PersistentVolumeClaim.Create != nil {
				vol.PersistentVolumeClaim.ClaimName = regv1.K8sPrefix + reg.Name
			} else {
				vol.PersistentVolumeClaim.ClaimName = reg.Spec.PersistentVolumeClaim.Exist.PvcName
			}
		}
	}

	// Patch
	if err := c.Patch(context.TODO(), target, originObject); err != nil {
		nt.logger.Error(err, "Unknown error patch")
		return err
	}
	return nil
}

func (nt *NotaryServer) delete(c client.Client, patchReg *regv1.Registry) error {
	if err := c.Delete(context.TODO(), nt.pod); err != nil {
		nt.logger.Error(err, "Unknown error delete deployment")
		return err
	}

	condition := status.Condition{
		Type:   regv1.ConditionTypeDeployment,
		Status: corev1.ConditionFalse,
	}

	patchReg.Status.Conditions.SetCondition(condition)

	return nil
}

func (nt *NotaryServer) compare(reg *regv1.Registry) []utils.Diff {
	diff := []utils.Diff{}
	var deployContainer *corev1.Container = nil
	podSpec := nt.pod.Spec.Template.Spec
	volumeMap := map[string]corev1.Volume{}

	// Get registry container
	for _, cont := range podSpec.Containers {
		if cont.Name == "registry" {
			deployContainer = &cont
		}
	}

	if deployContainer == nil {
		nt.logger.Error(regv1.MakeRegistryError(regv1.ContainerNotFound), "registry container is nil")
		return nil
	}

	// Get volumes
	for _, vol := range podSpec.Volumes {
		volumeMap[vol.Name] = vol
	}

	if reg.Spec.Image != deployContainer.Image {
		diff = append(diff, utils.Diff{Type: utils.Replace, Key: ImageDiffKey})
	}

	if reg.Spec.PersistentVolumeClaim.Create != nil {
		vol, exist := volumeMap["registry"]
		if !exist {
			nt.logger.Info("Registry volume is not exist.")
		} else if vol.VolumeSource.PersistentVolumeClaim.ClaimName != (regv1.K8sPrefix + reg.Name) {
			diff = append(diff, utils.Diff{Type: utils.Replace, Key: PvcNameDiffKey})
		}
	} else {
		vol, exist := volumeMap["registry"]
		if !exist {
			nt.logger.Info("Registry volume is not exist.")
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
		nt.logger.Error(regv1.MakeRegistryError(regv1.PvcVolumeMountNotFound), "registry pvc volume mount is nil")
		return nil
	}

	if reg.Spec.PersistentVolumeClaim.MountPath != contPvcVm.MountPath {
		diff = append(diff, utils.Diff{Type: utils.Replace, Key: MountPathDiffKey})
	}

	return diff
}
