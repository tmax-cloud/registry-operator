package schemes

import (
	regv1 "github.com/tmax-cloud/registry-operator/api/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	DBImage = "tmaxcloudck/notary_mysql:0.6.2-rc1"
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

	pod := &corev1.Pod{
		ObjectMeta: v1.ObjectMeta{
			Name:      resName,
			Namespace: notary.Namespace,
			Labels:    labels,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				corev1.Container{
					Name:  "notary-server",
					Image: DBImage,
					Args:  []string{"mysqld", "--innodb_file_per_table"},
					Env: []corev1.EnvVar{
						corev1.EnvVar{
							Name:  "TERM",
							Value: "dumb",
						},
						// TODO: set password
						corev1.EnvVar{
							Name:  "MYSQL_ALLOW_EMPTY_PASSWORD",
							Value: "true",
						},
					},
					Ports: []corev1.ContainerPort{
						corev1.ContainerPort{
							ContainerPort: 4443,
						},
					},
					VolumeMounts: []corev1.VolumeMount{
						corev1.VolumeMount{
							Name:      "data",
							MountPath: "/var/lib/mysql",
						},
					},
				},
			},
			Volumes: []corev1.Volume{
				corev1.Volume{
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

	return pod
}
