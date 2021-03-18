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

// NewRegistrySyncJob ...
func NewRegistrySyncJob(dependentJob *RegistryJob) *RegistrySyncJob {
	return &RegistrySyncJob{dependentJob: dependentJob}
}

// RegistrySyncJob is a registry job to synchronize external registry repository list
type RegistrySyncJob struct {
	dependentJob *RegistryJob
	job          *regv1.RegistryJob
	logger       *utils.RegistryLogger
}

// Handle is to create image replicate job.
func (r *RegistrySyncJob) Handle(c client.Client, repl *regv1.ImageReplicate, patchExreg *regv1.ImageReplicate, scheme *runtime.Scheme) error {
	if repl.Status.Conditions.GetCondition(regv1.ConditionTypeImageReplicateSynchronized).Status == corev1.ConditionTrue {
		return nil
	}

	if !r.dependentJob.IsSuccessfullyCompleted(c, repl) {
		return errors.New("RegistrySyncJob: registry job is not completed succesfully")
	}

	if err := r.get(c, repl); err != nil {
		if k8serr.IsNotFound(err) {
			if err := r.create(c, repl, patchExreg, scheme); err != nil {
				r.logger.Error(err, "create image replicate registry sync job error")
				return err
			}
		} else {
			r.logger.Error(err, "image replicate registry sync job error")
			return err
		}
	}

	return nil
}

// Ready is to check if the image replicate registry sync job is ready
func (r *RegistrySyncJob) Ready(c client.Client, repl *regv1.ImageReplicate, patchRepl *regv1.ImageReplicate, _ bool) error {
	if repl.Status.Conditions.GetCondition(regv1.ConditionTypeImageReplicateSynchronized).Status == corev1.ConditionTrue {
		return nil
	}

	var err error = nil
	condition := &status.Condition{
		Status: corev1.ConditionFalse,
		Type:   regv1.ConditionTypeImageReplicateSynchronized,
	}
	defer utils.SetCondition(err, patchRepl, condition)

	if err = r.get(c, repl); err != nil {
		r.logger.Error(err, "get image replicate registry sync job error")
		return err
	}

	condition.Status = corev1.ConditionTrue

	return nil
}

func (r *RegistrySyncJob) create(c client.Client, repl *regv1.ImageReplicate, patchRepl *regv1.ImageReplicate, scheme *runtime.Scheme) error {
	if r.job == nil {
		r.job = schemes.ImageReplicateSyncJob(repl)
	}

	if err := controllerutil.SetControllerReference(repl, r.job, scheme); err != nil {
		r.logger.Error(err, "SetOwnerReference Failed")
		return err
	}

	r.logger.Info("Create image replicate registry sync job")
	if err := c.Create(context.TODO(), r.job); err != nil {
		r.logger.Error(err, "Creating image replicate registry sync job is failed.")
		return err
	}

	return nil
}

func (r *RegistrySyncJob) get(c client.Client, repl *regv1.ImageReplicate) error {
	r.job = schemes.ImageReplicateSyncJob(repl)
	r.logger = utils.NewRegistryLogger(*r, r.job.Namespace, r.job.Name)

	req := types.NamespacedName{Name: r.job.Name, Namespace: r.job.Namespace}
	err := c.Get(context.TODO(), req, r.job)
	if err != nil {
		r.logger.Error(err, "Get image replicate registry sync job is failed")
		r.job = nil
		return err
	}

	return nil
}
