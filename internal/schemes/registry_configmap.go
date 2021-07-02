package schemes

import (
	regv1 "github.com/tmax-cloud/registry-operator/api/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const DefaultConfigMapName = "registry-config"

func ConfigMap(reg *regv1.Registry, data map[string]string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      SubresourceName(reg, SubTypeRegistryConfigmap),
			Namespace: reg.Namespace,
			Labels: map[string]string{
				"app":  "registry",
				"apps": SubresourceName(reg, SubTypeRegistryConfigmap),
			},
		},
		Data: data,
	}
}
