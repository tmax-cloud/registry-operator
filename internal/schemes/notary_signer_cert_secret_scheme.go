package schemes

import (
	regv1 "github.com/tmax-cloud/registry-operator/api/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NotarySignerSecret(notary *regv1.Notary, c client.Client) (*corev1.Secret, error) {
	tlsData := map[string][]byte{}

	cert, err := NewCertFactory(c).CreateCertPair(notary, certTypeNotarySigner)
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

	tlsData[TLSCert] = certPem
	tlsData[TLSKey] = keyPem

	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      SubresourceName(notary, SubTypeNotarySignerSecret),
			Namespace: notary.Namespace,
			Labels: map[string]string{
				"notary-signer": "tls",
			},
		},
		Type: corev1.SecretTypeTLS,
		Data: tlsData,
	}, nil
}
