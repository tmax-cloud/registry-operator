package v1

import (
	"encoding/json"
	"fmt"

	regv1 "github.com/tmax-cloud/registry-operator/api/v1"
	"k8s.io/api/admission/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type patchOperation struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value,omitempty"`
}

func Mutate(ar *v1beta1.AdmissionReview, client client.Client) *v1beta1.AdmissionResponse {
	req := ar.Request

	// AdmissionReview for Kind=tmax.io/v1, Kind=ImageSigner, Namespace= Name=yun  UID=685e6c98-a47c-4fb5-b2c5-8d8140eb0ffd patchOperation=CREATE UserInfo={admin@tmax.co.kr  [system:authenticated] map[]}
	log.Info(fmt.Sprintf("AdmissionReview for Kind=%v, Namespace=%v Name=%v  UID=%v patchOperation=%v UserInfo=%v",
		req.Kind, req.Namespace, req.Name, req.UID, req.Operation, req.UserInfo))

	signer := &regv1.ImageSigner{}
	if err := json.Unmarshal(req.Object.Raw, signer); err != nil {
		log.Error(err, "unable to unmarshal imagesigner", "name", req.Name)
		return &v1beta1.AdmissionResponse{
			Result: &metav1.Status{
				Message: err.Error(),
			},
		}
	}

	var patch []patchOperation

	patch = append(patch, patchOperation{
		Op:    "add",
		Path:  "/status/owner",
		Value: req.UserInfo.Username,
	})

	patchBytes, err := json.Marshal(patch)
	if err != nil {
		return &v1beta1.AdmissionResponse{
			Result: &metav1.Status{
				Message: err.Error(),
			},
		}
	}

	return &v1beta1.AdmissionResponse{
		Allowed: true,
		Patch:   patchBytes,
		PatchType: func() *v1beta1.PatchType {
			pt := v1beta1.PatchTypeJSONPatch
			return &pt
		}(),
	}
}
