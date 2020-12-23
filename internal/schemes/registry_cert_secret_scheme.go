package schemes

import (
	"strings"

	regv1 "github.com/tmax-cloud/registry-operator/api/v1"
	"github.com/tmax-cloud/registry-operator/internal/utils"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func Secrets(reg *regv1.Registry, c client.Client) (*corev1.Secret, *corev1.Secret) {
	logger := utils.GetRegistryLogger(corev1.Secret{}, reg.Namespace, reg.Name+"secret")
	if !regBodyCheckForSecrets(reg) {
		return nil, nil
	}
	data := map[string][]byte{}
	tlsData := map[string][]byte{}

	data["ID"] = []byte(reg.Spec.LoginId)
	data["PASSWD"] = []byte(reg.Spec.LoginPassword)

	cert, err := NewCertFactory(c).CreateCertPair(reg, certTypeRegistry)
	if err != nil {
		logger.Error(err, "")
		return nil, nil
	}

	certPem, err := cert.CertDataToPem()
	if err != nil {
		logger.Error(err, "")
		return nil, nil
	}

	keyPem, err := cert.KeyToPem()
	if err != nil {
		logger.Error(err, "")
		return nil, nil
	}

	tlsData[TLSCert] = certPem
	tlsData[TLSKey] = keyPem

	return &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      regv1.K8sPrefix + strings.ToLower(reg.Name),
				Namespace: reg.Namespace,
				Labels: map[string]string{
					"secret": "data",
				},
			},
			Type: corev1.SecretTypeOpaque,
			Data: data,
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      regv1.K8sPrefix + regv1.TLSPrefix + strings.ToLower(reg.Name),
				Namespace: reg.Namespace,
				Labels: map[string]string{
					"secret": "tls",
				},
			},
			Type: corev1.SecretTypeTLS,
			Data: tlsData,
		}
}

func regBodyCheckForSecrets(reg *regv1.Registry) bool {
	regService := reg.Spec.RegistryService
	if reg.Status.ClusterIP == "" {
		return false
	}
	if regService.ServiceType == regv1.RegServiceTypeLoadBalancer && reg.Status.LoadBalancerIP == "" {
		return false
	} else if regService.ServiceType == regv1.RegServiceTypeIngress && regService.Ingress.DomainName == "" {
		return false
	}
	return true
}
