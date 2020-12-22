package http

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/tmax-cloud/registry-operator/internal/schemes"

	corev1 "k8s.io/api/core/v1"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

var logger logr.Logger = logf.Log.WithName("common http")

func CAData() []byte {
	c, err := client.New(config.GetConfigOrDie(), client.Options{})
	if err != nil {
		logger.Error(err, "Unknown error")
		return nil
	}

	secret := &corev1.Secret{}
	err = c.Get(context.TODO(), types.NamespacedName{Name: schemes.RootCASecretName, Namespace: schemes.RootCASecretNamespace}, secret)
	if err != nil {
		logger.Error(err, "Unknown error")
		return nil
	}

	data := secret.Data
	cacrt, exist := data[schemes.RootCACert]
	if !exist {
		logger.Info("CA is not found")
		return nil
	}

	return cacrt
}
