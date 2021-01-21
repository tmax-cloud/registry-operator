package harbor

import (
	"context"

	"github.com/tmax-cloud/registry-operator/internal/common/config"
	exv1beta1 "k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	DefaultHarborCoreIngress   = "tmax-harbor-ingress"
	DefaultHarborNotaryIngress = "tmax-harbor-ingress-notary"
	DefaultHarborNamespace     = "harbor"
)

var logger = logf.Log.WithName("registry_harbor")

func IsHarbor(c client.Client, serverURL string) bool {
	regIng, err := Ingress(c)
	if err != nil {
		logger.Error(err, "")
		return false
	}

	if regIng.ResourceVersion != "" && len(regIng.Spec.Rules) == 1 && serverURL == regIng.Spec.Rules[0].Host {
		return true
	}

	return false
}

func Ingress(c client.Client) (*exv1beta1.Ingress, error) {
	regIng := &exv1beta1.Ingress{}
	harborNamespace := config.Config.GetString("harbor.namespace")
	if harborNamespace == "" {
		harborNamespace = DefaultHarborNamespace
	}

	harborCoreIngress := config.Config.GetString("harbor.core.ingress")
	if harborCoreIngress == "" {
		harborCoreIngress = DefaultHarborCoreIngress
	}

	if err := c.Get(context.Background(), types.NamespacedName{Name: harborCoreIngress, Namespace: harborNamespace}, regIng); err != nil {
		logger.Error(err, "")
		return nil, err
	}

	return regIng, nil
}
