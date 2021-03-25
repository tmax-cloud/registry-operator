package schemes

import (
	regv1 "github.com/tmax-cloud/registry-operator/api/v1"
	"github.com/tmax-cloud/registry-operator/internal/common/config"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func NotaryDBPod(notary *regv1.Notary) *corev1.Pod {
	var pvcName string
	labels := make(map[string]string)
	resName := SubresourceName(notary, SubTypeNotaryDBPod)
	labels["app"] = "notary-db"
	labels["apps"] = resName
	labels[resName] = "lb"

	if notary.Spec.PersistentVolumeClaim.Exist != nil {
		pvcName = notary.Spec.PersistentVolumeClaim.Exist.PvcName
	} else {
		pvcName = SubresourceName(notary, SubTypeNotaryDBPVC)
	}

	DBImage := config.Config.GetString(config.ConfigNotaryDBImage)

	litmitCPU := *notary.Spec.DB.Resources.Limits.Cpu()
	litmitMemory := *notary.Spec.DB.Resources.Limits.Memory()
	requestCPU := *notary.Spec.DB.Resources.Requests.Cpu()
	requestMemory := *notary.Spec.DB.Resources.Requests.Memory()

	if litmitCPU.IsZero() {
		litmitCPU = resource.MustParse(config.Config.GetString(config.ConfigNotaryDBCPU))
	}
	if litmitMemory.IsZero() {
		litmitMemory = resource.MustParse(config.Config.GetString(config.ConfigNotaryDBMemory))
	}
	if requestCPU.IsZero() {
		requestCPU = resource.MustParse(config.Config.GetString(config.ConfigNotaryDBCPU))
	}
	if requestMemory.IsZero() {
		requestMemory = resource.MustParse(config.Config.GetString(config.ConfigNotaryDBMemory))
	}

	pod := &corev1.Pod{
		ObjectMeta: v1.ObjectMeta{
			Name:      resName,
			Namespace: notary.Namespace,
			Labels:    labels,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "notary-server",
					Image: DBImage,
					Args:  []string{"mysqld", "--innodb_file_per_table"},
					Resources: corev1.ResourceRequirements{
						Limits: corev1.ResourceList{
							corev1.ResourceCPU:    litmitCPU,
							corev1.ResourceMemory: litmitMemory,
						},
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    requestCPU,
							corev1.ResourceMemory: requestMemory,
						},
					},
					Env: []corev1.EnvVar{
						{
							Name:  "TERM",
							Value: "dumb",
						},
						// TODO: set password
						{
							Name:  "MYSQL_ALLOW_EMPTY_PASSWORD",
							Value: "true",
						},
					},
					Ports: []corev1.ContainerPort{
						{
							ContainerPort: 4443,
						},
					},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "data",
							MountPath: "/var/lib/mysql",
						},
					},
				},
			},
			Volumes: []corev1.Volume{
				{
					Name: "data",
					VolumeSource: corev1.VolumeSource{
						PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
							ClaimName: pvcName,
						},
					},
				},
			},
		},
	}

	// set image pull secret
	if config.Config.GetString(config.ConfigNotaryDBImagePullSecret) != "" {
		pod.Spec.ImagePullSecrets = append(pod.Spec.ImagePullSecrets, corev1.LocalObjectReference{Name: config.Config.GetString(config.ConfigNotaryDBImagePullSecret)})
	}

	return pod
}
