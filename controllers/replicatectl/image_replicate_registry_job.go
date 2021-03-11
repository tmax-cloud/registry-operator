package replicatectl

import (
	"context"
	"errors"

	"github.com/operator-framework/operator-lib/status"
	regv1 "github.com/tmax-cloud/registry-operator/api/v1"
	"github.com/tmax-cloud/registry-operator/internal/schemes"
	"github.com/tmax-cloud/registry-operator/internal/utils"
	corev1 "k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type RegistryJob struct {
	job    *regv1.RegistryJob
	logger *utils.RegistryLogger
}

// Handle is to create image replicate job.
func (r *RegistryJob) Handle(c client.Client, repl *regv1.ImageReplicate, patchExreg *regv1.ImageReplicate, scheme *runtime.Scheme) error {
	if err := r.get(c, repl); err != nil {
		if k8serr.IsNotFound(err) {
			if err := r.create(c, repl, patchExreg, scheme); err != nil {
				r.logger.Error(err, "create image replicate registry job error")
				return err
			}
		} else {
			r.logger.Error(err, "image replicate registry job error")
			return err
		}
	}

	return nil
}

// Ready is to check if the image replicate registry job is ready
func (r *RegistryJob) Ready(c client.Client, repl *regv1.ImageReplicate, patchRepl *regv1.ImageReplicate, useGet bool) error {
	var existErr error = nil
	existCondition := &status.Condition{
		Status: corev1.ConditionFalse,
		Type:   regv1.ConditionTypeImageReplicateRegistryJobExist,
	}
	condition := &status.Condition{}

	if useGet {
		if existErr = r.get(c, repl); existErr != nil {
			r.logger.Error(existErr, "get image replicate registry job error")
			return existErr
		}
	}

	defer utils.SetCondition(existErr, patchRepl, existCondition)
	if r.job == nil {
		existErr = errors.New("registry job is not found")
		return existErr
	}
	existCondition.Status = corev1.ConditionTrue

	switch repl.Status.State {
	case regv1.ImageReplicatePending:
		if r.job.Status.State == regv1.RegistryJobStateRunning ||
			r.job.Status.State == regv1.RegistryJobStateCompleted ||
			r.job.Status.State == regv1.RegistryJobStateFailed {
			condition.Status = corev1.ConditionTrue
			condition.Type = regv1.ConditionTypeImageReplicateRegistryJobProcessing
			defer utils.SetCondition(nil, patchRepl, condition)
		}

	case regv1.ImageReplicateProcessing:
		condition.Status = corev1.ConditionUnknown
		condition.Type = regv1.ConditionTypeImageReplicateRegistryJobSuccess
		defer utils.SetCondition(nil, patchRepl, condition)
		if r.job.Status.State == regv1.RegistryJobStateCompleted {
			condition.Status = corev1.ConditionTrue
			break
		}
		if r.job.Status.State == regv1.RegistryJobStateFailed {
			condition.Status = corev1.ConditionFalse
			break
		}
	}

	return nil
}

func (r *RegistryJob) create(c client.Client, repl *regv1.ImageReplicate, patchRepl *regv1.ImageReplicate, scheme *runtime.Scheme) error {
	r.job = schemes.ImageReplicateJob(repl)
	if err := controllerutil.SetControllerReference(repl, r.job, scheme); err != nil {
		r.logger.Error(err, "SetOwnerReference Failed")
		return err
	}

	r.logger.Info("Create image replicate registry job")
	if err := c.Create(context.TODO(), r.job); err != nil {
		r.logger.Error(err, "Creating image replicate registry job is failed.")
		return err
	}

	return nil
}

func (r *RegistryJob) get(c client.Client, repl *regv1.ImageReplicate) error {
	r.job = schemes.ImageReplicateJob(repl)
	r.logger = utils.NewRegistryLogger(*r, r.job.Namespace, r.job.Name)

	req := types.NamespacedName{Name: r.job.Name, Namespace: r.job.Namespace}
	err := c.Get(context.TODO(), req, r.job)
	if err != nil {
		r.logger.Error(err, "Get image replicate registry job is failed")
		r.job = nil
		return err
	}

	return nil
}

func (r *RegistryJob) IsSuccessfullyCompleted(c client.Client, repl *regv1.ImageReplicate) bool {
	if err := r.get(c, repl); err != nil {
		r.logger.Error(err, "image replicate registry job error")
		return false
	}

	if r.job == nil {
		return false
	}

	if r.job.Status.State != regv1.RegistryJobStateCompleted {
		return false
	}

	return true
}
