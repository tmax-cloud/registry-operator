package schemes

import (
	"fmt"
	regv1 "github.com/tmax-cloud/registry-operator/api/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func Secrets(reg *regv1.Registry, c client.Client) (*corev1.Secret, *corev1.Secret, error) {
	if !regBodyCheckForSecrets(reg) {
		return nil, nil, fmt.Errorf("failed to generate manifest: not yet assigned registry service IP")
	}
	data := map[string][]byte{}
	tlsData := map[string][]byte{}

	data["ID"] = []byte(reg.Spec.LoginID)
	data["PASSWD"] = []byte(reg.Spec.LoginPassword)

	cert, err := NewCertFactory(c).CreateCertPair(reg, certTypeRegistry)
	if err != nil {
		return nil, nil, err
	}

	certPem, err := cert.CertDataToPem()
	if err != nil {
		return nil, nil, err
	}

	keyPem, err := cert.KeyToPem()
	if err != nil {
		return nil, nil, err
	}

	tlsData[TLSCert] = certPem
	tlsData[TLSKey] = keyPem

	return &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      SubresourceName(reg, SubTypeRegistryOpaqueSecret),
				Namespace: reg.Namespace,
				Labels: map[string]string{
					"secret": "data",
				},
			},
			Type: corev1.SecretTypeOpaque,
			Data: data,
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      SubresourceName(reg, SubTypeRegistryTLSSecret),
				Namespace: reg.Namespace,
				Labels: map[string]string{
					"secret": "tls",
				},
			},
			Type: corev1.SecretTypeTLS,
			Data: tlsData,
		},
		nil
}

func regBodyCheckForSecrets(reg *regv1.Registry) bool {
	regService := reg.Spec.RegistryService
	if reg.Status.ClusterIP == "" {
		return false
	}
	if regService.ServiceType == regv1.RegServiceTypeLoadBalancer && reg.Status.LoadBalancerIP == "" {
		return false
	}
	return true
}
