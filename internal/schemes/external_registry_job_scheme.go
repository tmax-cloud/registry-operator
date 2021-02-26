package schemes

import (
	regv1 "github.com/tmax-cloud/registry-operator/api/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ExternalRegistryJob is a scheme of external registry job
func ExternalRegistryJob(exreg *regv1.ExternalRegistry) *regv1.RegistryJob {
	labels := make(map[string]string)
	resName := SubresourceName(exreg, SubTypeExternalRegistryJob)
	labels["app"] = "external-registry-job"
	labels["apps"] = resName

	return &regv1.RegistryJob{
		ObjectMeta: v1.ObjectMeta{
			Name:      resName,
			Namespace: exreg.Namespace,
			Labels:    labels,
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
