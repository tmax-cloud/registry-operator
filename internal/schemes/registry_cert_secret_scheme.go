package schemes

import (
	regv1 "github.com/tmax-cloud/registry-operator/api/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func CredentialSecret(reg *regv1.Registry) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      SubresourceName(reg, SubTypeRegistryOpaqueSecret),
			Namespace: reg.Namespace,
			Labels: map[string]string{
				"secret": "data",
			},
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"ID":     []byte(reg.Spec.LoginID),
			"PASSWD": []byte(reg.Spec.LoginPassword),
		},
	}
}

func TlsSecret(reg *regv1.Registry, c client.Client) (*corev1.Secret, error) {

	cert, err := NewCertFactory(c).CreateCertPair(reg, certTypeRegistry)
	if err != nil {
		return nil, err
	}

	certPem, err := cert.CertDataToPem()
	if err != nil {
		return nil, err
	}

	keyPem, err := cert.KeyToPem()
	if err != nil {
		return nil, err
	}

	return &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      SubresourceName(reg, SubTypeRegistryTLSSecret),
				Namespace: reg.Namespace,
				Labels: map[string]string{
					"secret": "tls",
				},
			},
			Type: corev1.SecretTypeTLS,
			Data: map[string][]byte{
				corev1.TLSCertKey:       certPem,
				corev1.TLSPrivateKeyKey: keyPem,
			},
		},
		nil
}
