package schemes

import (
	regv1 "github.com/tmax-cloud/registry-operator/api/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ExternalRegistryJob is a scheme of external registry job
func ExternalRegistryJob(exreg *regv1.ExternalRegistry) *regv1.RegistryJob {
	return &regv1.RegistryJob{
		ObjectMeta: v1.ObjectMeta{
			Name:      SubresourceName(exreg, SubTypeExternalRegistryJob),
			Namespace: exreg.Namespace,
			Labels: map[string]string{
				"app":  "external-registry-job",
				"apps": SubresourceName(exreg, SubTypeExternalRegistryJob),
			},
		},
		Spec: regv1.RegistryJobSpec{
			Priority: 100,
			Claim: &regv1.RegistryJobClaim{
				JobType: regv1.JobTypeSynchronizeExtReg,
				HandleObject: corev1.LocalObjectReference{
					Name: exreg.Name,
				},
			},
		},
	}
}
