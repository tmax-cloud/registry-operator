package schemes

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	regv1 "github.com/tmax-cloud/registry-operator/api/v1"
)

const (
	NotarySignerDefaultHTTPPort = 4444
	NotarySignerDefaultGRPCPort = 7899
)

func NotarySignerService(notary *regv1.Notary) *corev1.Service {
	labels := make(map[string]string)
	resName := SubresourceName(notary, SubTypeNotarySignerService)
	labels["app"] = "notary-signer"
	labels["apps"] = resName
	httpPort := NotarySignerDefaultHTTPPort
	grpcPort := NotarySignerDefaultGRPCPort

	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      resName,
			Namespace: notary.Namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeClusterIP,
			Selector: map[string]string{
				resName: "lb",
			},
			Ports: []corev1.ServicePort{
				corev1.ServicePort{
					Name:     "http",
					Protocol: "TCP",
					Port:     int32(httpPort),
				},
				corev1.ServicePort{
					Name:     "grpc",
					Protocol: "TCP",
					Port:     int32(grpcPort),
				},
			},
		},
	}
}
