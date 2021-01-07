package schemes

import (
	"strings"

	regv1 "github.com/tmax-cloud/registry-operator/api/v1"
	"github.com/tmax-cloud/registry-operator/internal/common/ingress"

	"k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func Ingress(reg *regv1.Registry) *v1beta1.Ingress {
	if !regBodyCheckForIngress(reg) {
		schemeLogger.Info("failed to check registry body for ingress")
		return nil
	}
	registryDomain := RegistryDomainName(reg)
	if registryDomain == "" {
		return nil
	}

	ingressTLS := v1beta1.IngressTLS{
		Hosts:      []string{registryDomain},
		SecretName: SubresourceName(reg, SubTypeRegistryTLSSecret),
	}
	httpIngressPath := v1beta1.HTTPIngressPath{
		Path: "/",
		Backend: v1beta1.IngressBackend{
			ServiceName: SubresourceName(reg, SubTypeRegistryService),
			ServicePort: intstr.FromInt(443),
		},
	}

	rule := v1beta1.IngressRule{
		Host: registryDomain,
		IngressRuleValue: v1beta1.IngressRuleValue{
			HTTP: &v1beta1.HTTPIngressRuleValue{
				Paths: []v1beta1.HTTPIngressPath{
					httpIngressPath,
				},
			},
		},
	}

	return &v1beta1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      SubresourceName(reg, SubTypeRegistryIngress),
			Namespace: reg.Namespace,
			Labels: map[string]string{
				"app":  "registry",
				"apps": SubresourceName(reg, SubTypeRegistryIngress),
			},
			Annotations: map[string]string{
				"kubernetes.io/ingress.class":                       "nginx-shd",
				"nginx.ingress.kubernetes.io/proxy-connect-timeout": "3600",
				"nginx.ingress.kubernetes.io/proxy-read-timeout":    "3600",
				"nginx.ingress.kubernetes.io/ssl-redirect":          "true",
				"nginx.ingress.kubernetes.io/backend-protocol":      "HTTPS",
				"nginx.ingress.kubernetes.io/proxy-body-size":       "0",
			},
		},
		Spec: v1beta1.IngressSpec{
			TLS: []v1beta1.IngressTLS{
				ingressTLS,
			},
			Rules: []v1beta1.IngressRule{
				rule,
			},
		},
	}
}

func regBodyCheckForIngress(reg *regv1.Registry) bool {
	regService := reg.Spec.RegistryService
	return regService.ServiceType == "Ingress"
}

func RegistryDomainName(reg *regv1.Registry) string {
	icIP := ingress.GetIngressControllerIP()
	if icIP == "" {
		return ""
	}

	return strings.Join([]string{reg.Namespace, reg.Name, "registry", icIP, "nip", "io"}, ".")
}
