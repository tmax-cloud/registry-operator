package v1

import (
	"context"
	"encoding/json"
	"fmt"

	regv1 "github.com/tmax-cloud/registry-operator/api/v1"
	"k8s.io/api/admission/v1beta1"
	v1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func ImageSignRequest(ar *v1beta1.AdmissionReview, c client.Client) *v1beta1.AdmissionResponse {
	req := ar.Request

	// AdmissionReview for Kind=tmax.io/v1, Kind=ImageSigner, Namespace= Name=yun  UID=685e6c98-a47c-4fb5-b2c5-8d8140eb0ffd patchOperation=CREATE UserInfo={admin@tmax.co.kr  [system:authenticated] map[]}
	log.Info(fmt.Sprintf("AdmissionReview for Kind=%v, Namespace=%v Name=%v  UID=%v patchOperation=%v UserInfo=%v",
		req.Kind, req.Namespace, req.Name, req.UID, req.Operation, req.UserInfo))

	isr := &regv1.ImageSignRequest{}
	if err := json.Unmarshal(req.Object.Raw, isr); err != nil {
		log.Error(err, "unable to unmarshal imagesignrequest", "name", req.Name)
		return &v1beta1.AdmissionResponse{
			Result: &metav1.Status{
				Message: err.Error(),
			},
		}
	}

	// get clusterrolebinding by signer
	crbList := &v1.ClusterRoleBindingList{}
	label := map[string]string{}
	label["object"] = "imagesigner"
	label["signer"] = isr.Spec.Signer
	labelSelector := labels.SelectorFromSet(labels.Set(label))
	listOps := &client.ListOptions{
		LabelSelector: labelSelector,
	}

	if err := c.List(context.TODO(), crbList, listOps); err != nil {
		log.Error(err, "unable to unmarshal imagesignrequest", "name", req.Name)
		return &v1beta1.AdmissionResponse{
			Result: &metav1.Status{
				Message: err.Error(),
			},
		}
	}

	// check if clusterrolebinding's subject name equals request's username
	for _, crb := range crbList.Items {
		for _, s := range crb.Subjects {
			log.Info("check allowed user", "request user", req.UserInfo.Username, "allowed user", s.Name)
			if s.Name == req.UserInfo.Username {
				return &v1beta1.AdmissionResponse{
					Allowed: true,
				}
			}
		}
	}

	return &v1beta1.AdmissionResponse{
		Result: &metav1.Status{
			Message: fmt.Sprintf("%s user is not allowed to sign image from %s signer", req.UserInfo.Username, isr.Spec.Signer),
		},
	}

}
