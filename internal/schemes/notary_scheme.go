package schemes

import (
	regv1 "github.com/tmax-cloud/registry-operator/api/v1"
	"github.com/tmax-cloud/registry-operator/internal/common/certs"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Notary(reg *regv1.Registry, auth *regv1.AuthConfig) (*regv1.Notary, error) {
	labels := make(map[string]string)
	resName := SubresourceName(reg, SubTypeRegistryNotary)
	labels["app"] = "notary"
	labels["apps"] = resName

	if _, err := certs.GetRootCert(reg.Namespace); err != nil {
		return nil, err
	}

	notary := &regv1.Notary{
		ObjectMeta: metav1.ObjectMeta{
			Name:      resName,
			Namespace: reg.Namespace,
			Labels:    labels,
		},
		Spec: regv1.NotarySpec{
			RootCASecret: certs.RootCASecretName,
			AuthConfig: regv1.AuthConfig{
				Issuer:  auth.Issuer,
				Realm:   auth.Realm,
				Service: auth.Service,
			},
			ServiceType: reg.Spec.Notary.ServiceType,
			Server:      reg.Spec.Notary.Server,
			Signer:      reg.Spec.Notary.Signer,
			DB:          reg.Spec.Notary.DB,
		},
	}

	if reg.Spec.Notary.PersistentVolumeClaim.Exist != nil {
		notary.Spec.PersistentVolumeClaim.Exist = reg.Spec.Notary.PersistentVolumeClaim.Exist.DeepCopy()
	} else {
		notary.Spec.PersistentVolumeClaim.Create = reg.Spec.Notary.PersistentVolumeClaim.Create.DeepCopy()
	}

	return notary, nil
}
