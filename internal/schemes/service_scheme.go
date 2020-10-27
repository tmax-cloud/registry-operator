package schemes

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	regv1 "github.com/tmax-cloud/registry-operator/api/v1"
)

const (
	RegistryTargetPort   = 443
	RegistryPortProtocol = "TCP"
	RegistryPortName     = "tls"
)

func Service(reg *regv1.Registry) *corev1.Service {
	regServiceName := regv1.K8sPrefix + reg.Name
	label := map[string]string{}
	label["app"] = "registry"
	label["apps"] = regv1.K8sPrefix + reg.Name
	port := 443

	serviceName := reg.Spec.RegistryService.ServiceType
	if serviceName == regv1.RegServiceTypeLoadBalancer {
		port = reg.Spec.RegistryService.LoadBalancer.Port
	} else {
		serviceName = regv1.RegServiceTypeIngress
	}

	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      regServiceName,
			Namespace: reg.Namespace,
			Labels:    label,
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceType(serviceName),
			Selector: map[string]string{
				regv1.K8sPrefix + reg.Name: "lb",
			},
			Ports: []corev1.ServicePort{
				{
					Name:     RegistryPortName,
					Protocol: RegistryPortProtocol,
					Port:     int32(port),
				},
			},
		},
	}
}
