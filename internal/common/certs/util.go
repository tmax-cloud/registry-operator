package certs

import (
	corev1 "k8s.io/api/core/v1"
)

const (
	RootCACert = "ca.crt"
	RootCAPriv = "ca.key"
)

func CAData(secret *corev1.Secret) ([]byte, []byte) {
	data := secret.Data
	cacrt, exist := data[RootCACert]
	if !exist {
		return nil, nil
	}
	cakey, exist := data[RootCAPriv]
	if !exist {
		return nil, nil
	}

	return cacrt, cakey
}
