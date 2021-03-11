package schemes

import (
	regv1 "github.com/tmax-cloud/registry-operator/api/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ImageReplicateImageSignRequest is a scheme of image replicate job
func ImageReplicateImageSignRequest(repl *regv1.ImageReplicate, image, loginSecret, certSecret string) *regv1.ImageSignRequest {
	labels := make(map[string]string)
	resName := SubresourceName(repl, SubTypeImageReplicateImageSignRequest)
	labels["app"] = "image-replicate-image-sign-request"
	labels["apps"] = resName

	return &regv1.ImageSignRequest{
		ObjectMeta: v1.ObjectMeta{
			Name:      resName,
			Namespace: repl.Namespace,
			Labels:    labels,
		},
		Spec: regv1.ImageSignRequestSpec{
			Image:  image,
			Signer: repl.Spec.Signer,
			RegistrySecret: regv1.RegistrySecret{
				DcjSecretName:  loginSecret,
				CertSecretName: certSecret,
			},
		},
	}
}
