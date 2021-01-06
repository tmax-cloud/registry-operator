package repoctl

import (
	"context"

	"github.com/tmax-cloud/registry-operator/internal/schemes"

	regv1 "github.com/tmax-cloud/registry-operator/api/v1"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

type RegistryRepository struct {
}

func New() *RegistryRepository {
	return &RegistryRepository{}
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

func (r *RegistryRepository) Get(c client.Client, reg *regv1.Registry, imageName string) (*regv1.Repository, error) {
	repo := &regv1.Repository{}

	logger.Info("Get", "Registry", reg.Name, "Repository", schemes.RepositoryName(imageName, reg.Name), "Namespace", reg.Namespace)
	if err := c.Get(context.TODO(), types.NamespacedName{Name: schemes.RepositoryName(imageName, reg.Name), Namespace: reg.Namespace}, repo); err != nil {
		logger.Error(err, "failed to get repository")
		return nil, err
	}

	return repo, nil
}

func (r *RegistryRepository) Patch(c client.Client, repo *regv1.Repository, patchRepo *regv1.Repository) error {
	originObject := client.MergeFrom(repo)

	// Patch
	if err := c.Patch(context.TODO(), patchRepo, originObject); err != nil {
		logger.Error(err, "Unknown error patching repository spec")
		return err
	}

	logger.Info("Patched", "Repository", patchRepo.Name+"/"+patchRepo.Namespace)
	return nil
}

func (r *RegistryRepository) Update(c client.Client, repo *regv1.Repository) error {
	// Update
	if err := c.Update(context.TODO(), repo); err != nil {
		logger.Error(err, "Unknown error updating repository spec")
		return err
	}

	logger.Info("Updated", "Repository", repo.Name+"/"+repo.Namespace)
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
