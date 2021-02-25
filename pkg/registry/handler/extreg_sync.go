package handler

import (
	"context"

	v1 "github.com/tmax-cloud/registry-operator/api/v1"
	"github.com/tmax-cloud/registry-operator/internal/common/http"
	"github.com/tmax-cloud/registry-operator/internal/utils"
	"github.com/tmax-cloud/registry-operator/pkg/registry/ext/factory"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var Logger = log.Log.WithName("registry-handler")

func ExternalRegistrySyncHandleFunc(k8sClient client.Client, object types.NamespacedName, scheme *runtime.Scheme) error {
	// get external registry
	exreg := &v1.ExternalRegistry{}
	exregNamespacedName := object
	if err := k8sClient.Get(context.TODO(), exregNamespacedName, exreg); err != nil {
		Logger.Error(err, "")
	}

	username, password := "", ""
	if exreg.Spec.ImagePullSecret != "" {
		basic, err := utils.GetBasicAuth(exreg.Spec.ImagePullSecret, exreg.Namespace, exreg.Spec.RegistryURL)
		if err != nil {
			Logger.Error(err, "failed to get basic auth")
		}

		username, password = utils.DecodeBasicAuth(basic)
	}

	var ca []byte
	if exreg.Spec.CertificateSecret != "" {
		data, err := utils.GetCAData(exreg.Spec.CertificateSecret, exreg.Namespace)
		if err != nil {
			Logger.Error(err, "failed to get ca data")
		}
		ca = data
	}

	syncFactory := factory.NewSyncRegistryFactory(
		k8sClient,
		exregNamespacedName,
		scheme,
		http.NewHTTPClient(
			exreg.Spec.RegistryURL,
			username, password,
			ca,
			exreg.Spec.Insecure,
		),
	)

	syncClient := syncFactory.Create(exreg.Spec.RegistryType)
	if err := syncClient.Synchronize(); err != nil {
		Logger.Error(err, "failed to synchronize external registry")
		return err
	}
	return nil
}
