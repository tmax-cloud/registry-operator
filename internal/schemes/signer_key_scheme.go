package schemes

import (
	apiv1 "github.com/tmax-cloud/registry-operator/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func SignerKey(signer *apiv1.ImageSigner) *apiv1.SignerKey {
	return &apiv1.SignerKey{
		ObjectMeta: metav1.ObjectMeta{
			Name: signer.Name,
		},
	}
}
