package schemes

import (
	regv1 "github.com/tmax-cloud/registry-operator/api/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ImageReplicateSyncJob is a scheme of image replicate sync job
func ImageReplicateSyncJob(repl *regv1.ImageReplicate) *regv1.RegistryJob {
	labels := make(map[string]string)
	resName := SubresourceName(repl, SubTypeImageReplicateSyncJob)
	labels["app"] = "image-replicate-sync-job"
	labels["apps"] = resName

	return &regv1.RegistryJob{
		ObjectMeta: v1.ObjectMeta{
			Name:      resName,
			Namespace: repl.Namespace,
			Labels:    labels,
		},
		Spec: regv1.RegistryJobSpec{
			Priority: 100,
			TTL:      60,
			Claim: &regv1.RegistryJobClaim{
				JobType: regv1.JobTypeSynchronizeExtReg,
				HandleObject: corev1.LocalObjectReference{
					Name: repl.Spec.ToImage.RegistryName,
				},
			},
		},
	}
}
