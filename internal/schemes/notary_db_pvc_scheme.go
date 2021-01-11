package schemes

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	regv1 "github.com/tmax-cloud/registry-operator/api/v1"
)

func NotaryDBPVC(notary *regv1.Notary) *corev1.PersistentVolumeClaim {
	labels := make(map[string]string)
	resName := SubresourceName(notary, SubTypeNotaryDBPVC)
	labels["app"] = "notary-db"
	labels["apps"] = resName

	var accessModes []corev1.PersistentVolumeAccessMode
	var vm corev1.PersistentVolumeMode = corev1.PersistentVolumeFilesystem

	if notary.Spec.PersistentVolumeClaim.Exist != nil {
		resName = notary.Spec.PersistentVolumeClaim.Exist.PvcName
		return &corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      resName,
				Namespace: notary.Namespace,
				Labels:    labels,
			},
		}
	}

	for _, mode := range notary.Spec.PersistentVolumeClaim.Create.AccessModes {
		accessModes = append(accessModes, corev1.PersistentVolumeAccessMode(mode))
	}

	return &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      resName,
			Namespace: notary.Namespace,
			Labels:    labels,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: accessModes,
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse(notary.Spec.PersistentVolumeClaim.Create.StorageSize),
				},
				Limits: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse(notary.Spec.PersistentVolumeClaim.Create.StorageSize),
				},
			},
			StorageClassName: &notary.Spec.PersistentVolumeClaim.Create.StorageClassName,
			VolumeMode:       &vm,
		},
	}
}
