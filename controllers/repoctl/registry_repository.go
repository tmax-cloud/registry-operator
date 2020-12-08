package repoctl

import (
	"context"

	"github.com/tmax-cloud/registry-operator/internal/schemes"

	regv1 "github.com/tmax-cloud/registry-operator/api/v1"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

type RegistryRepository struct {
}

var logger = logf.Log.WithName("registry_repository")

func (r *RegistryRepository) Create(c client.Client, reg *regv1.Registry, imageName string, tags []string, scheme *runtime.Scheme) error {
	repo := schemes.Repository(reg, imageName, tags)
	if err := controllerutil.SetControllerReference(reg, repo, scheme); err != nil {
		logger.Error(err, "Controller reference failed")
		return err
	}

	if err := c.Create(context.TODO(), repo); err != nil {
		logger.Error(err, "Create failed")
		return err
	}

	logger.Info("Created", "Registry", reg.Name, "Repository", repo.Name, "Namespace", reg.Namespace)
	return nil
}

func (r *RegistryRepository) Patch(c client.Client, repo *regv1.Repository, patchRepo *regv1.Repository) error {
	originObject := client.MergeFrom(repo)

	// Patch
	if err := c.Patch(context.TODO(), patchRepo, originObject); err != nil {
		logger.Error(err, "Unknown error patching status")
		return err
	}

	logger.Info("Patched", "Repository", patchRepo.Name+"/"+patchRepo.Namespace)
	return nil
}

func (r *RegistryRepository) Delete(c client.Client, reg *regv1.Registry, imageName string, scheme *runtime.Scheme) error {
	repo := schemes.Repository(reg, imageName, nil)
	if err := c.Delete(context.TODO(), repo); err != nil {
		logger.Error(err, "Delete failed")
		return err
	}

	logger.Info("Deleted", "Registry", reg.Name, "Repository", repo.Name, "Namespace", reg.Namespace)
	return nil
}