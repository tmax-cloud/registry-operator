package schemes

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	regv1 "github.com/tmax-cloud/registry-operator/api/v1"
)

const (
	NotaryServerDefaultPort = 4443
)

func NotaryServerService(notary *regv1.Notary) *corev1.Service {
	labels := make(map[string]string)
	resName := SubresourceName(notary, SubTypeNotaryServerService)
	labels["app"] = "notary-server"
	labels["apps"] = resName
	port := NotaryServerDefaultPort

	svcType := notary.Spec.ServiceType
	if svcType != regv1.RegServiceTypeLoadBalancer {
		svcType = regv1.RegServiceTypeIngress
	}

	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      resName,
			Namespace: notary.Namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceType(svcType),
			Selector: map[string]string{
				resName: "lb",
			},
			Ports: []corev1.ServicePort{
				{
					Name:     "http",
					Protocol: "TCP",
					Port:     int32(port),
				},
			},
		},
	}
}
