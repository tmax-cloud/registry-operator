package v1

import (
	"github.com/go-logr/logr"
	"net/http"
	"sigs.k8s.io/controller-runtime/pkg/client"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/tmax-cloud/registry-operator/internal/utils"
)

const (
	NamespaceParamKey = "namespace"
)

var logger = ctrl.Log.WithName("signer-apis")

type RegistryAPI struct {
	c      client.Client
	logger logr.Logger
}

func NewRegistryAPI(c client.Client, logger logr.Logger) *RegistryAPI {
	return &RegistryAPI{
		c:      c,
		logger: logger,
	}
}

func (h RegistryAPI) ApisHandler(w http.ResponseWriter, _ *http.Request) {
	groupVersion := metav1.GroupVersionForDiscovery{
		GroupVersion: "registry.tmax.io/v1",
		Version:      "v1",
	}
	_ = utils.RespondJSON(w, &metav1.APIGroupList{
		TypeMeta: metav1.TypeMeta{
			Kind: "APIGroupList",
		},
		Groups: []metav1.APIGroup{
			{
				TypeMeta: metav1.TypeMeta{
					Kind:       "APIGroup",
					APIVersion: "",
				},
				Name:             "registry.tmax.io",
				Versions:         []metav1.GroupVersionForDiscovery{groupVersion},
				PreferredVersion: groupVersion,
				ServerAddressByClientCIDRs: []metav1.ServerAddressByClientCIDR{
					{
						ClientCIDR:    "0.0.0.0/0",
						ServerAddress: "",
					},
				},
			},
		},
	})
}

func (h RegistryAPI) VersionHandler(w http.ResponseWriter, _ *http.Request) {
	_ = utils.RespondJSON(w, &metav1.APIResourceList{
		TypeMeta: metav1.TypeMeta{
			Kind: "APIResourceList",
		},
		GroupVersion: "registry.tmax.io/v1",
		APIResources: []metav1.APIResource{
			{
				Name:       "imagesigners/keys",
				Namespaced: true,
			},
		},
	})
}
