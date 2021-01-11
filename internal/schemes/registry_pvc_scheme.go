package schemes

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	regv1 "github.com/tmax-cloud/registry-operator/api/v1"
)

func PersistentVolumeClaim(reg *regv1.Registry) *corev1.PersistentVolumeClaim {
	var resName string
	label := map[string]string{}
	label["app"] = "registry"
	label["apps"] = regv1.K8sPrefix + reg.Name

	var accessModes []corev1.PersistentVolumeAccessMode
	var vm corev1.PersistentVolumeMode = corev1.PersistentVolumeFilesystem

	if reg.Spec.PersistentVolumeClaim.Exist != nil {
		resName = reg.Spec.PersistentVolumeClaim.Exist.PvcName
		return &corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      resName,
				Namespace: reg.Namespace,
				Labels:    label,
			},
		}
	}

	resName = regv1.K8sPrefix + reg.Name

	for _, mode := range reg.Spec.PersistentVolumeClaim.Create.AccessModes {
		accessModes = append(accessModes, corev1.PersistentVolumeAccessMode(mode))
	}

	return &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      resName,
			Namespace: reg.Namespace,
			Labels:    label,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: accessModes,
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse(reg.Spec.PersistentVolumeClaim.Create.StorageSize),
				},
				Limits: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse(reg.Spec.PersistentVolumeClaim.Create.StorageSize),
				},
			},
			StorageClassName: &reg.Spec.PersistentVolumeClaim.Create.StorageClassName,
			VolumeMode:       &vm,
		},
	}
}
