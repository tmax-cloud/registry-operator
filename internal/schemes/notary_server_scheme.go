package schemes

import (
	regv1 "github.com/tmax-cloud/registry-operator/api/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ServerPrefix = "server-"
	ServerImage  = ""
)

func NotaryServerPod(reg *regv1.Registry) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: v1.ObjectMeta{
			Name:      regv1.K8sPrefix + regv1.K8sNotaryPrefix + ServerPrefix + reg.Name,
			Namespace: reg.Namespace,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				corev1.Container{
					Name:  "notary-server",
					Image: ServerImage,
				},
			},
		},
	}
}
