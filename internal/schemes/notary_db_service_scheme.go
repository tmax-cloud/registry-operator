package schemes

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	regv1 "github.com/tmax-cloud/registry-operator/api/v1"
)

const (
	NotaryDBDefaultPort = 3306
)

func NotaryDBService(notary *regv1.Notary) *corev1.Service {
	labels := make(map[string]string)
	resName := SubresourceName(notary, SubTypeNotaryDBService)
	labels["app"] = "notary-db"
	labels["apps"] = resName
	port := NotaryDBDefaultPort

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
				{
					Name:     "port",
					Protocol: "TCP",
					Port:     int32(port),
				},
			},
		},
	}
}
