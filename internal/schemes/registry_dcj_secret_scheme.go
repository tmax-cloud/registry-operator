package schemes

import (
	"encoding/base64"
	"encoding/json"

	regv1 "github.com/tmax-cloud/registry-operator/api/v1"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type DockerConfig struct {
	Auths map[string]AuthValue `json:"auths"`
}

type AuthValue struct {
	Auth string `json:"auth"`
}

func DCJSecret(reg *regv1.Registry) *corev1.Secret {
	if !regBodyCheckForDCJSecret(reg) {
		return nil
	}
	serviceType := reg.Spec.RegistryService.ServiceType
	var domainList []string
	data := map[string][]byte{}
	if serviceType == regv1.RegServiceTypeLoadBalancer {
		// port = reg.Spec.RegistryService.LoadBalancer.Port
		domainList = append(domainList, reg.Status.LoadBalancerIP)
	} else {
		domainList = append(domainList, RegistryDomainName(reg))
	}
	domainList = append(domainList, reg.Status.ClusterIP)

	config := DockerConfig{
		Auths: map[string]AuthValue{},
	}
	for _, domain := range domainList {
		config.Auths[domain] = AuthValue{base64.StdEncoding.EncodeToString([]byte(reg.Spec.LoginID + ":" + reg.Spec.LoginPassword))}
	}

	configBytes, _ := json.Marshal(config)
	data[corev1.DockerConfigJsonKey] = configBytes

	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      SubresourceName(reg, SubTypeRegistryDCJSecret),
			Namespace: reg.Namespace,
			Labels: map[string]string{
				"secret": "docker",
			},
		},
		Type: corev1.SecretTypeDockerConfigJson,
		Data: data,
	}
}

func regBodyCheckForDCJSecret(reg *regv1.Registry) bool {
	regService := reg.Spec.RegistryService
	if reg.Status.ClusterIP == "" {
		return false
	}
	if regService.ServiceType == regv1.RegServiceTypeLoadBalancer && reg.Status.LoadBalancerIP == "" {
		return false
	}
	return true
}
