package schemes

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	regv1 "github.com/tmax-cloud/registry-operator/api/v1"
)

func RegistryRole(reg *regv1.Registry, data map[string]string) *corev1.ConfigMap {
	var resName string
	label := map[string]string{}
	label["app"] = "registry"
	label["apps"] = SubresourceName(reg, SubTypeRegistryConfigmap)

	if len(reg.Spec.CustomConfigYml) != 0 {
		resName = reg.Spec.CustomConfigYml
	} else {
		resName = SubresourceName(reg, SubTypeRegistryConfigmap)
	}

	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      resName,
			Namespace: reg.Namespace,
			Labels:    label,
		},
		Data: data,
	}
}
