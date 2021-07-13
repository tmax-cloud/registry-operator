package schemes

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	regv1 "github.com/tmax-cloud/registry-operator/api/v1"
)

func Service(reg *regv1.Registry) *corev1.Service {
	serviceName := reg.Spec.RegistryService.ServiceType
	if serviceName != regv1.RegServiceTypeLoadBalancer {
		serviceName = regv1.RegServiceTypeIngress
	}

	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      SubresourceName(reg, SubTypeRegistryService),
			Namespace: reg.Namespace,
			Labels: map[string]string{
				"app":  "registry",
				"apps": SubresourceName(reg, SubTypeRegistryService),
			},
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceType(serviceName),
			Selector: map[string]string{
				SubresourceName(reg, SubTypeRegistryService): "lb",
			},
			Ports: []corev1.ServicePort{
				{
					Name:     "tls",
					Protocol: "TCP",
					Port:     443,
				},
			},
		},
	}
}
