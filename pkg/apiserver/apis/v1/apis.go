package v1

import (
	tmaxiov1 "github.com/tmax-cloud/registry-operator/api/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	authorization "k8s.io/client-go/kubernetes/typed/authorization/v1"
	"net/http"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/tmax-cloud/registry-operator/internal/utils"
)

const (
	NamespaceParamKey = "namespace"
)

var logger = ctrl.Log.WithName("signer-apis")
var authClient *authorization.AuthorizationV1Client
var k8sClient client.Client

func Initiate() {
	// Auth Client
	authCli, err := utils.AuthClient()
	if err != nil {
		logger.Error(err, "")
		os.Exit(1)
	}
	authClient = authCli

	// K8s Client
	opt := client.Options{Scheme: runtime.NewScheme()}
	utilruntime.Must(tmaxiov1.AddToScheme(opt.Scheme))
	utilruntime.Must(corev1.AddToScheme(opt.Scheme))

	cli, err := utils.Client(opt)
	if err != nil {
		logger.Error(err, "")
		os.Exit(1)
	}
	k8sClient = cli
}

func ApisHandler(w http.ResponseWriter, _ *http.Request) {
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

func VersionHandler(w http.ResponseWriter, _ *http.Request) {
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
