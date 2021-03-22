package replicatectl

import (
	"context"
	"errors"
	"path"

	"github.com/operator-framework/operator-lib/status"
	regv1 "github.com/tmax-cloud/registry-operator/api/v1"
	"github.com/tmax-cloud/registry-operator/internal/schemes"
	"github.com/tmax-cloud/registry-operator/internal/utils"
	"github.com/tmax-cloud/registry-operator/pkg/registry"
	corev1 "k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// NewImageSignRequest ...
func NewImageSignRequest(dependentJob *RegistryJob) *ImageSignRequest {
	return &ImageSignRequest{dependentJob: dependentJob}
}

// ImageSignRequest ...
type ImageSignRequest struct {
	dependentJob *RegistryJob
	isr          *regv1.ImageSignRequest
	logger       *utils.RegistryLogger
}

// Handle is to create image sign request.
func (r *ImageSignRequest) Handle(c client.Client, repl *regv1.ImageReplicate, patchExreg *regv1.ImageReplicate, scheme *runtime.Scheme) error {
	if !r.dependentJob.IsSuccessfullyCompleted(c, repl) {
		return errors.New("ImageSignRequest: registry job is not completed succesfully")
	}

	if err := r.get(c, repl); err != nil {
		if k8serr.IsNotFound(err) {
			if err := r.create(c, repl, patchExreg, scheme); err != nil {
				r.logger.Error(err, "create image replicate image sign request error")
				return err
			}
		} else {
			r.logger.Error(err, "image replicate image sign request error")
			return err
		}
	}

	return nil
}

// Ready is to check if image sign request is ready
func (r *ImageSignRequest) Ready(c client.Client, repl *regv1.ImageReplicate, patchRepl *regv1.ImageReplicate, useGet bool) error {
	var existErr error = nil
	existCondition := &status.Condition{
		Status: corev1.ConditionFalse,
		Type:   regv1.ConditionTypeImageReplicateImageSignRequestExist,
	}
	condition1 := &status.Condition{}
	condition2 := &status.Condition{}

	if useGet {
		if existErr = r.get(c, repl); existErr != nil {
			r.logger.Error(existErr, "get image sign request error")
			return existErr
		}
	}

	defer utils.SetCondition(existErr, patchRepl, existCondition)
	if r.isr == nil {
		existErr = errors.New("image sign request is not found")
		return existErr
	}
	existCondition.Status = corev1.ConditionTrue

	switch repl.Status.State {
	case regv1.ImageReplicatePending, regv1.ImageReplicateProcessing:
		if r.isr.Status.ImageSignResponse == nil {
			err := errors.New("ImageSignResponse is nil")
			r.logger.Error(err, "")
			return err
		}

		if r.isr.Status.ImageSignResponse.Result == regv1.ResponseResultSigning ||
			r.isr.Status.ImageSignResponse.Result == regv1.ResponseResultSuccess ||
			r.isr.Status.ImageSignResponse.Result == regv1.ResponseResultFail {
			condition1.Status = corev1.ConditionTrue
			condition1.Type = regv1.ConditionTypeImageReplicateImageSigning
			defer utils.SetCondition(nil, patchRepl, condition1)
		}

		condition2.Status = corev1.ConditionUnknown
		condition2.Type = regv1.ConditionTypeImageReplicateImageSigningSuccess
		defer utils.SetCondition(nil, patchRepl, condition2)
		if r.isr.Status.ImageSignResponse.Result == regv1.ResponseResultSuccess {
			condition2.Status = corev1.ConditionTrue
			break
		}
		if r.isr.Status.ImageSignResponse.Result == regv1.ResponseResultFail {
			condition2.Status = corev1.ConditionFalse
			break
		}
	}

	return nil
}

func (r *ImageSignRequest) create(c client.Client, repl *regv1.ImageReplicate, patchRepl *regv1.ImageReplicate, scheme *runtime.Scheme) error {
	if r.isr == nil {
		image, err := r.getImageFullName(c, repl)
		if err != nil {
			r.logger.Error(err, "failed to get image full name")
			return err
		}

		reg := types.NamespacedName{Namespace: repl.Spec.ToImage.RegistryNamespace, Name: repl.Spec.ToImage.RegistryName}
		imagePullSecret, err := registry.GetLoginSecret(c, reg, repl.Spec.ToImage.RegistryType)
		if err != nil {
			r.logger.Error(err, "failed to get login secret")
			return err
		}
		certificate, err := registry.GetCertSecret(c, reg, repl.Spec.ToImage.RegistryType)
		if err != nil {
			r.logger.Error(err, "failed to get certificate")
			return err
		}

		r.isr = schemes.ImageReplicateImageSignRequest(repl, image, imagePullSecret, certificate)
	}

	if err := controllerutil.SetControllerReference(repl, r.isr, scheme); err != nil {
		r.logger.Error(err, "SetOwnerReference Failed")
		return err
	}

	r.logger.Info("Create image replicate image sign request")
	if err := c.Create(context.TODO(), r.isr); err != nil {
		r.logger.Error(err, "Creating image replicate image sign request is failed.")
		return err
	}

	patchRepl.Status.ImageSignRequestName = r.isr.Name

	return nil
}

func (r *ImageSignRequest) get(c client.Client, repl *regv1.ImageReplicate) error {
	r.logger = utils.NewRegistryLogger(*r, repl.Namespace, schemes.SubresourceName(repl, schemes.SubTypeImageReplicateImageSignRequest))
	image, err := r.getImageFullName(c, repl)
	if err != nil {
		r.logger.Error(err, "failed to get image full name")
		return err
	}

	reg := types.NamespacedName{Namespace: repl.Spec.ToImage.RegistryNamespace, Name: repl.Spec.ToImage.RegistryName}
	imagePullSecret, err := registry.GetLoginSecret(c, reg, repl.Spec.ToImage.RegistryType)
	if err != nil {
		r.logger.Error(err, "failed to get login secret")
		return err
	}
	r.logger.Info("get", "imagePullSecret", imagePullSecret, "namespace", reg.Namespace)
	certificate, err := registry.GetCertSecret(c, reg, repl.Spec.ToImage.RegistryType)
	if err != nil {
		r.logger.Error(err, "failed to get certificate")
		return err
	}
	r.logger.Info("get", "certificate", certificate, "namespace", reg.Namespace)

	r.isr = schemes.ImageReplicateImageSignRequest(repl, image, imagePullSecret, certificate)

	req := types.NamespacedName{Name: r.isr.Name, Namespace: r.isr.Namespace}
	if err := c.Get(context.TODO(), req, r.isr); err != nil {
		r.logger.Error(err, "Get image replicate image sign request is failed")
		r.isr = nil
		return err
	}

	return nil
}

func (r *ImageSignRequest) getImageFullName(c client.Client, repl *regv1.ImageReplicate) (string, error) {
	reg := types.NamespacedName{Name: repl.Spec.ToImage.RegistryName, Namespace: repl.Spec.ToImage.RegistryNamespace}
	url, err := registry.GetURL(c, reg, repl.Spec.ToImage.RegistryType)
	if err != nil {
		r.logger.Error(err, "failed to get url", "registryType", repl.Spec.ToImage.RegistryType, "registryName", reg.Name, "registryNamespace", reg.Namespace)
		return "", err
	}

	url = utils.TrimHTTPScheme(url)
	image := path.Join(url, repl.Spec.ToImage.Image)
	return image, nil
}
