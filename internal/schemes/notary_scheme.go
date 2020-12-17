package schemes

import (
	regv1 "github.com/tmax-cloud/registry-operator/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Notary(reg *regv1.Registry, auth *regv1.AuthConfig) *regv1.Notary {
	labels := make(map[string]string)
	resName := SubresourceName(reg, SubTypeRegistryNotary)
	labels["app"] = "notary"
	labels["apps"] = resName

	notary := &regv1.Notary{
		ObjectMeta: metav1.ObjectMeta{
			Name:      resName,
			Namespace: reg.Namespace,
			Labels:    labels,
		},
		Spec: regv1.NotarySpec{
			RootCASecret: reg.Spec.Notary.RootCASecret,
			AuthConfig: regv1.AuthConfig{
				Issuer:  auth.Issuer,
				Realm:   auth.Realm,
				Service: auth.Service,
			},
			ServiceType: reg.Spec.Notary.ServiceType,
		},
	}

	if reg.Spec.Notary.PersistentVolumeClaim.Exist != nil {
		notary.Spec.PersistentVolumeClaim.Exist = reg.Spec.Notary.PersistentVolumeClaim.Exist.DeepCopy()
	} else {
		notary.Spec.PersistentVolumeClaim.Create = reg.Spec.Notary.PersistentVolumeClaim.Create.DeepCopy()
	}

	return notary
}
