package controllers

import (
	"context"

	tmaxiov1 "github.com/tmax-cloud/registry-operator/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func updateSignerStatus(c client.Client, signer *tmaxiov1.ImageSigner) error {
	if err := c.Status().Update(context.TODO(), signer); err != nil {
		return err
	}

	return nil
}

func makeSignerStatus(signer *tmaxiov1.ImageSigner, created bool, reason, message string, key *tmaxiov1.TrustKey) {
	signer.Status.SignerKeyState = &tmaxiov1.SignerKeyState{}
	if created {
		signer.Status.Created = true
		signer.Status.CreatedAt = metav1.Now()
		signer.Status.RootKeyID = key.ID
	} else {
		signer.Status.Created = false
		signer.Status.Reason = reason
		signer.Status.Message = message
	}
}

func response(c client.Client, signReq *tmaxiov1.ImageSignRequest) error {
	if err := c.Status().Update(context.TODO(), signReq); err != nil {
		return err
	}

	return nil
}

func makeResponse(signReq *tmaxiov1.ImageSignRequest, result bool, reason, message string) {
	signReq.Status.ImageSignResponse = &tmaxiov1.ImageSignResponse{}
	if result {
		signReq.Status.Result = tmaxiov1.ResponseResultSuccess
	} else {
		signReq.Status.Result = tmaxiov1.ResponseResultFail
	}
	signReq.Status.Reason = reason
	signReq.Status.Message = message
}
