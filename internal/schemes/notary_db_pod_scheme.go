package schemes

import (
	regv1 "github.com/tmax-cloud/registry-operator/api/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	DBImage = "mariadb:10.4"
)

func NotaryDBPod(notary *regv1.Notary) *corev1.Pod {
	labels := make(map[string]string)
	resName := SubresourceName(notary, SubTypeNotaryDBPod)
	labels["app"] = "notary-db"
	labels["apps"] = resName
	labels[resName] = "lb"

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
				},
			},
		},
	}

	return pod
}
