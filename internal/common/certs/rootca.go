package certs

import (
	"context"
	"strings"

	regv1 "github.com/tmax-cloud/registry-operator/api/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

const (
	RootCASecretName = regv1.K8sPrefix + regv1.K8sRegistryPrefix + "rootca"
)

// GetRootCert returns registry's root ca certificate secret.
// If not exist, create root ca secret as registry-ca secret in operator namespace
func GetRootCert(namespace string) (*corev1.Secret, error) {
	c, err := client.New(config.GetConfigOrDie(), client.Options{})
	if err != nil {
		return nil, err
	}

	secret, err := getRootCASecret(c, namespace)
	if err != nil && !errors.IsNotFound(err) {
		return nil, err
	}

	secret, err = createRootCASecret(c, namespace)
	if err != nil {
		return nil, err
	}

	return secret, nil
}

func getRootCASecret(c client.Client, namespace string) (*corev1.Secret, error) {
	secret := &corev1.Secret{}
	if err := c.Get(context.TODO(), types.NamespacedName{Name: RootCASecretName, Namespace: namespace}, secret); err != nil {
		return nil, err
	}

	return secret, nil
}

func createRootCASecret(c client.Client, namespace string) (*corev1.Secret, error) {
	sysRegCA := &corev1.Secret{}
	if err := c.Get(context.TODO(), types.NamespacedName{Name: regv1.RegistryRootCASecretName, Namespace: regv1.OperatorNamespace}, sysRegCA); err != nil {
		return nil, err
	}

	var crtData []byte
	for k, v := range sysRegCA.Data {
		if strings.Contains(k, ".crt") {
			crtData = v
			break
		}
	}

	secret := &corev1.Secret{
		ObjectMeta: v1.ObjectMeta{
			Name:      RootCASecretName,
			Namespace: namespace,
		},
		Data: map[string][]byte{"ca.crt": crtData},
	}

	if err := c.Create(context.TODO(), secret); err != nil {
		return nil, err
	}

	return secret, nil
}
