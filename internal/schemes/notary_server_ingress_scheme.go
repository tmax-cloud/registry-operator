package schemes

import (
	"strings"

	regv1 "github.com/tmax-cloud/registry-operator/api/v1"
	"github.com/tmax-cloud/registry-operator/internal/common/ingress"

	"k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func NotaryServerIngress(notary *regv1.Notary) *v1beta1.Ingress {
	notaryDomain := NotaryDomainName(notary)
	if notaryDomain == "" {
		return nil
	}

	ingressTLS := v1beta1.IngressTLS{
		Hosts:      []string{notaryDomain},
		SecretName: SubresourceName(notary, SubTypeNotaryServerSecret),
	}
	httpIngressPath := v1beta1.HTTPIngressPath{
		Path: "/",
		Backend: v1beta1.IngressBackend{
			ServiceName: SubresourceName(notary, SubTypeNotaryServerService),
			ServicePort: intstr.FromInt(NotaryServerDefaultPort),
		},
	}

	rule := v1beta1.IngressRule{
		Host: notaryDomain,
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
			Name:      SubresourceName(notary, SubTypeNotaryServerIngress),
			Namespace: notary.Namespace,
			Labels: map[string]string{
				"app":  "notary-server",
				"apps": SubresourceName(notary, SubTypeNotaryServerIngress),
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

func NotaryDomainName(notary *regv1.Notary) string {
	icIP := ingress.GetIngressControllerIP()
	if icIP == "" {
		return ""
	}

	return strings.Join([]string{notary.Namespace, notary.Name, "notary", icIP, "nip", "io"}, ".")
}
