package registry

import (
	"context"
	"strings"

	apiv1 "github.com/tmax-cloud/registry-operator/api/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func getRegistry(c client.Client, regName, namespace string) (*apiv1.Registry, error) {
	reg := &apiv1.Registry{}
	if err := c.Get(context.TODO(), types.NamespacedName{Name: regName, Namespace: namespace}, reg); err != nil {
		return nil, err
	}

	return reg, nil
}

type RegCtl struct {
	client client.Client
	reg    *apiv1.Registry
}

// NewRegCtl is a controller for registry
// if registryName or registryNamespace is empty string, RegCtl is nil
func NewRegCtl(c client.Client, regName, namespace string) *RegCtl {
	if len(regName) == 0 || len(namespace) == 0 {
		return nil
	}

	reg, err := getRegistry(c, regName, namespace)
	if err != nil {
		return nil
	}

	return &RegCtl{
		client: c,
		reg:    reg,
	}
}

func (r *RegCtl) GetHostname() string {
	return strings.TrimPrefix(r.GetEndpoint(), "https://")
}

func (r *RegCtl) GetEndpoint() string {
	return r.reg.Status.ServerURL
}

func (r *RegCtl) GetNotaryEndpoint() string {
	return r.reg.Status.NotaryURL
}
