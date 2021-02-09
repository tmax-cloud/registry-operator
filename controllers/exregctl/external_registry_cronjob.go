package exregctl

import (
	"context"

	regv1 "github.com/tmax-cloud/registry-operator/api/v1"
	"github.com/tmax-cloud/registry-operator/internal/schemes"
	"github.com/tmax-cloud/registry-operator/internal/utils"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type RegistryCronJob struct {
	cron   *regv1.RegistryCronJob
	logger *utils.RegistryLogger
}

// Handle is to create external registry cron job.
func (r *RegistryCronJob) Handle(c client.Client, exreg *regv1.ExternalRegistry, patchExreg *regv1.ExternalRegistry, scheme *runtime.Scheme) error {
	if err := r.get(c, exreg); err != nil {
		if errors.IsNotFound(err) {
			if err := r.create(c, exreg, patchExreg, scheme); err != nil {
				r.logger.Error(err, "create external registry cron job error")
				return err
			}
		} else {
			r.logger.Error(err, "external registry cron job error")
			return err
		}
	}

	return nil
}

// Ready is to check if the external registry cron job is ready
func (r *RegistryCronJob) Ready(c client.Client, exreg *regv1.ExternalRegistry, patchExreg *regv1.ExternalRegistry, useGet bool) error {
	if useGet {
		if err := r.get(c, exreg); err != nil {
			r.logger.Error(err, "get external registry cron job error")
			return err
		}
	}

	diff := r.compare(exreg)
	if diff == nil {
		r.logger.Error(nil, "Invalid cron job!!!")
		if err := r.delete(c, patchExreg); err != nil {
			return err
		}
	} else if len(diff) > 0 {
		r.logger.Info("NotReady")
		err := regv1.MakeRegistryError("NotReady")
		return err
	}

	r.logger.Info("Ready")
	patchExreg.Status.State = regv1.ExternalRegistryScheduled
	return nil
}

func (r *RegistryCronJob) create(c client.Client, exreg *regv1.ExternalRegistry, patchExreg *regv1.ExternalRegistry, scheme *runtime.Scheme) error {
	if err := controllerutil.SetControllerReference(exreg, r.cron, scheme); err != nil {
		r.logger.Error(err, "SetOwnerReference Failed")
		return err
	}

	r.logger.Info("Create external registry cron job")
	if err := c.Create(context.TODO(), r.cron); err != nil {
		r.logger.Error(err, "Creating external registry cron job is failed.")
		return err
	}

	return nil
}

func (r *RegistryCronJob) get(c client.Client, exreg *regv1.ExternalRegistry) error {
	r.cron = schemes.ExternalRegistryCronJob(exreg)
	r.logger = utils.NewRegistryLogger(*r, r.cron.Namespace, r.cron.Name)

	req := types.NamespacedName{Name: r.cron.Name, Namespace: r.cron.Namespace}
	err := c.Get(context.TODO(), req, r.cron)
	if err != nil {
		r.logger.Error(err, "Get external registry cron job is failed")
		return err
	}

	return nil
}

func (r *RegistryCronJob) compare(reg *regv1.ExternalRegistry) []utils.Diff {
	diff := []utils.Diff{}

	// TODO

	return diff
}

func (r *RegistryCronJob) patch(c client.Client, exreg *regv1.ExternalRegistry, patchExreg *regv1.ExternalRegistry, diff []utils.Diff) error {

	return nil
}

func (r *RegistryCronJob) delete(c client.Client, patchExreg *regv1.ExternalRegistry) error {
	if err := c.Delete(context.TODO(), r.cron); err != nil {
		r.logger.Error(err, "Unknown error delete deployment")
		return err
	}

	return nil
}
