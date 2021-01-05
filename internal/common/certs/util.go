package certs

import (
	corev1 "k8s.io/api/core/v1"
)

const (
	RootCACert = "ca.crt"
	RootCAPriv = "ca.key"
)

func CAData(secret *corev1.Secret) ([]byte, []byte) {
	return secret.Data[RootCACert], secret.Data[RootCAPriv]
}
