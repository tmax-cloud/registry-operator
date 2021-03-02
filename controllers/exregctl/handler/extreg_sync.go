package handler

import (
	"context"
	"errors"

	v1 "github.com/tmax-cloud/registry-operator/api/v1"
	"github.com/tmax-cloud/registry-operator/internal/common/http"
	"github.com/tmax-cloud/registry-operator/internal/utils"
	"github.com/tmax-cloud/registry-operator/pkg/registry/base"
	"github.com/tmax-cloud/registry-operator/pkg/registry/ext/factory"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var logger = log.Log.WithName("extregctl-handler")

// NewExternalRegistrySyncHandler returns a new handler to synchronize external registry
func NewExternalRegistrySyncHandler(k8sClient client.Client, scheme *runtime.Scheme) *ExternalRegistrySyncHandler {
	return &ExternalRegistrySyncHandler{
		k8sClient: k8sClient,
		scheme:    scheme,
	}
}

// ExternalRegistrySyncHandler contains objects to use in handle function
type ExternalRegistrySyncHandler struct {
	k8sClient client.Client
	scheme    *runtime.Scheme
}

// Handle synchronizes external registry repository list
func (h *ExternalRegistrySyncHandler) Handle(object types.NamespacedName) error {
	log := logger.WithValues("namespace", object.Namespace, "name", object.Name)
	// get external registry
	exreg := &v1.ExternalRegistry{}
	exregNamespacedName := object
	if err := h.k8sClient.Get(context.TODO(), exregNamespacedName, exreg); err != nil {
		log.Error(err, "")
	}

	username, password := "", ""
	if exreg.Spec.ImagePullSecret != "" {
		basic, err := utils.GetBasicAuth(exreg.Spec.ImagePullSecret, exreg.Namespace, exreg.Spec.RegistryURL)
		if err != nil {
			log.Error(err, "failed to get basic auth")
		}

		username, password = utils.DecodeBasicAuth(basic)
	}

	var ca []byte
	if exreg.Spec.CertificateSecret != "" {
		data, err := utils.GetCAData(exreg.Spec.CertificateSecret, exreg.Namespace)
		if err != nil {
			log.Error(err, "failed to get ca data")
		}
		ca = data
	}

	syncFactory := factory.NewRegistryFactory(
		h.k8sClient,
		exregNamespacedName,
		h.scheme,
		http.NewHTTPClient(
			exreg.Spec.RegistryURL,
			username, password,
			ca,
			exreg.Spec.Insecure,
		),
	)

	syncClient, ok := syncFactory.Create(exreg.Spec.RegistryType).(base.Synchronizable)
	if !ok {
		err := errors.New("unable to convert to synchronizable")
		log.Error(err, "failed to create sync client")
		return err
	}
	if err := syncClient.Synchronize(); err != nil {
		log.Error(err, "failed to synchronize external registry")
		return err
	}
	return nil
}
