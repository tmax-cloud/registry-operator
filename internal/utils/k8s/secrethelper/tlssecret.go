package secrethelper

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
)

func GetCert(secret *corev1.Secret, certKey string) ([]byte, error) {
	if secret.Type == corev1.SecretTypeTLS {
		cert, ok := secret.Data[corev1.TLSCertKey]
		if !ok {
			return nil, fmt.Errorf("Not found cert key: %s\n", corev1.TLSCertKey)
		}
		return cert, nil

	} else if secret.Type == corev1.SecretTypeOpaque {
		cert, ok := secret.Data[certKey]
		if !ok {
			return nil, fmt.Errorf("Not found cert key: %s\n", certKey)
		}
		return cert, nil
	}

	return nil, fmt.Errorf("Unsupported secret type")

}

func GetPrivateKey(secret *corev1.Secret, keyKey string) ([]byte, error) {
	if secret.Type == corev1.SecretTypeTLS {
		key, ok := secret.Data[corev1.TLSPrivateKeyKey]
		if !ok {
			return nil, fmt.Errorf("Failed to get dockerconfig from TlsSecret")
		}
		return key, nil

	} else if secret.Type == corev1.SecretTypeOpaque {
		key, ok := secret.Data[keyKey]
		if !ok {
			return nil, fmt.Errorf("Failed to get dockerconfig from TlsSecret")
		}
		return key, nil
	} else {
		return nil, fmt.Errorf("Unsupported secret type")
	}
}
