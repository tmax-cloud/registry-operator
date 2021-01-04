package v1

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	regv1 "github.com/tmax-cloud/registry-operator/api/v1"
	"k8s.io/api/admission/v1beta1"
	authorization "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func ImageSignRequest(ar *v1beta1.AdmissionReview, w http.ResponseWriter, r *http.Request) *v1beta1.AdmissionResponse {
	req := ar.Request

	// AdmissionReview for Kind=tmax.io/v1, Kind=ImageSigner, Namespace= Name=yun  UID=685e6c98-a47c-4fb5-b2c5-8d8140eb0ffd patchOperation=CREATE UserInfo={admin@tmax.co.kr  [system:authenticated] map[]}
	log.Info(fmt.Sprintf("AdmissionReview for Kind=%v, Namespace=%v Name=%v  UID=%v patchOperation=%v UserInfo=%v",
		req.Kind, req.Namespace, req.Name, req.UID, req.Operation, req.UserInfo))

	if err := reviewAccessImageSigner(req); err != nil {
		log.Error(err, "image signer is not allowed. or failed to check subject's authorization")
		return &v1beta1.AdmissionResponse{
			Result: &metav1.Status{
				Message: fmt.Sprintf("image signer is not allowed. or failed to check subject's authorization: %s", err.Error()),
			},
		}
	}

	return &v1beta1.AdmissionResponse{
		Allowed: true,
	}
}

func reviewAccessImageSigner(req *v1beta1.AdmissionRequest) error {
	userName := req.UserInfo.Username

	isr := &regv1.ImageSignRequest{}
	if err := json.Unmarshal(req.Object.Raw, isr); err != nil {
		log.Error(err, "unable to unmarshal imagesignrequest", "name", req.Name)
		return err
	}

	resourceName := isr.Spec.Signer

	r := &authorization.SubjectAccessReview{
		Spec: authorization.SubjectAccessReviewSpec{
			User: userName,
			ResourceAttributes: &authorization.ResourceAttributes{
				Name:     resourceName,
				Group:    "tmax.io",
				Version:  ApiVersion,
				Resource: "signerkeys",
				Verb:     "get",
			},
		},
	}

	log.Info("SubjectAccessReview", "spec", fmt.Sprintf("%+v", r.Spec))
	result, err := authClient.SubjectAccessReviews().Create(context.TODO(), r, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	if result.Status.Allowed {
		return nil
	}

	return fmt.Errorf(result.Status.Reason)
}
