package v1

import (
	"fmt"
	tmaxiov1 "github.com/tmax-cloud/registry-operator/api/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	authorization "k8s.io/client-go/kubernetes/typed/authorization/v1"
	"net/http"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/tmax-cloud/registry-operator/internal/utils"
	"github.com/tmax-cloud/registry-operator/internal/wrapper"
)

const (
	ApiGroup   = "registry.tmax.io"
	ApiVersion = "v1"
	SignerKind = "imagesigners"

	NamespaceParamKey = "namespace"
	ResourceParamKey  = "resourceName"
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

	cli, err := utils.Client(opt)
	if err != nil {
		logger.Error(err, "")
		os.Exit(1)
	}
	k8sClient = cli
}

func AddV1Apis(parent *wrapper.RouterWrapper) error {
	versionWrapper := wrapper.New(fmt.Sprintf("/%s/%s", ApiGroup, ApiVersion), nil, versionHandler)
	if err := parent.Add(versionWrapper); err != nil {
		return err
	}

	namespaceWrapper := wrapper.New(fmt.Sprintf("/namespaces/{%s}", NamespaceParamKey), nil, nil)
	if err := versionWrapper.Add(namespaceWrapper); err != nil {
		return err
	}

	// Image scan request
	if err := AddScanRequest(namespaceWrapper); err != nil {
		return err
	}

	// Image scan result
	if err := AddScanResult(namespaceWrapper); err != nil {
		return err
	}

	return nil
}

func versionHandler(w http.ResponseWriter, _ *http.Request) {
	apiResourceList := &metav1.APIResourceList{}
	apiResourceList.Kind = "APIResourceList"
	apiResourceList.GroupVersion = fmt.Sprintf("%s/%s", ApiGroup, ApiVersion)
	apiResourceList.APIVersion = ApiVersion

	apiResourceList.APIResources = []metav1.APIResource{
		{
			Name:       fmt.Sprintf("%s/keys", SignerKind),
			Namespaced: true,
		},
	}

	_ = utils.RespondJSON(w, apiResourceList)
}
