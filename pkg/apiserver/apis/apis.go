package apis

import (
	"fmt"
	"net/http"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/tmax-cloud/registry-operator/internal/utils"
	"github.com/tmax-cloud/registry-operator/internal/wrapper"
)

const (
	ApiGroup   = "registry.tmax.io"
	ApiVersion = "v1"
)

var AddApiFuncs []func(*wrapper.RouterWrapper) error

func AddApis(parent *wrapper.RouterWrapper) error {
	apiWrapper := wrapper.New("/apis", nil, apisHandler)
	if err := parent.Add(apiWrapper); err != nil {
		return err
	}

	for _, f := range AddApiFuncs {
		if err := f(apiWrapper); err != nil {
			return err
		}
	}

	return nil
}

func apisHandler(w http.ResponseWriter, _ *http.Request) {
	groupVersion := metav1.GroupVersionForDiscovery{
		GroupVersion: fmt.Sprintf("%s/%s", ApiGroup, ApiVersion),
		Version:      ApiVersion,
	}

	group := metav1.APIGroup{}
	group.Kind = "APIGroup"
	group.Name = ApiGroup
	group.PreferredVersion = groupVersion
	group.Versions = append(group.Versions, groupVersion)
	group.ServerAddressByClientCIDRs = append(group.ServerAddressByClientCIDRs, metav1.ServerAddressByClientCIDR{
		ClientCIDR:    "0.0.0.0/0",
		ServerAddress: "",
	})

	apiGroupList := &metav1.APIGroupList{}
	apiGroupList.Kind = "APIGroupList"
	apiGroupList.Groups = append(apiGroupList.Groups, group)

	_ = utils.RespondJSON(w, apiGroupList)
}
