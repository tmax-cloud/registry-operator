package sync

import (
	"context"

	regv1 "github.com/tmax-cloud/registry-operator/api/v1"
	"github.com/tmax-cloud/registry-operator/controllers/repoctl"
	"github.com/tmax-cloud/registry-operator/internal/schemes"
	"github.com/tmax-cloud/registry-operator/internal/utils"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var logger = log.Log.WithName("sync-registry")

// Registry synchronizes custom resource repository based on all repositories in registry server
func Registry(c client.Client, registry, namespace string, scheme *runtime.Scheme, repos *regv1.APIRepositoryList) error {
	syncLog := logger.WithValues("registry_name", registry, "registry_ns", namespace)

	crImages, crImageNames, err := crImages(c, registry, namespace)
	if err != nil {
		syncLog.Error(err, "failed to get cr")
		return err
	}

	regImageNames := []string{}
	if repos != nil {
		for _, repo := range *repos {
			syncLog.Info("Repository", "Name", repo.Name)
			regImageNames = append(regImageNames, repo.Name)
		}
	}
	// Comparing ImageName and get New Images Name & Deleted Images Name
	newRepositories, deletedRepositories, existRepositories := compareRepositories(regImageNames, crImageNames, crImages)

	// For New Image, Insert Image and Versions Data from Repository
	repoCtl := &repoctl.RegistryRepository{}
	reg := &regv1.Registry{}
	if err := c.Get(context.TODO(), types.NamespacedName{Name: registry, Namespace: namespace}, reg); err != nil {
		syncLog.Error(err, "")
	}

	for _, newImageName := range newRepositories {
		syncLog.Info("create new repository cr", "name", schemes.RepositoryName(newImageName, registry))
		newRepo := repos.GetRepository(newImageName)
		if err := repoCtl.Create(c, reg, newRepo.Name, newRepo.Tags, scheme); err != nil {
			syncLog.Error(err, "failed to create repository")
			return err
		}
	}

	// For Deleted Image, Delete Image Data from Repository
	for _, deletedImageName := range deletedRepositories {
		syncLog.Info("delete repository cr", "name", schemes.RepositoryName(deletedImageName, registry))
		if err := repoCtl.Delete(c, reg, deletedImageName, scheme); err != nil {
			syncLog.Error(err, "failed to delete image")
			return err
		}
	}

	// For Exist Image, Compare tags List, Insert Version Data from Repository
	if err := patchRepository(c, registry, namespace, existRepositories, repos); err != nil {
		syncLog.Error(err, "failed to patch repository")
		return err
	}

	return nil
}

// ExternalRegistry synchronizes external registry repository list
func ExternalRegistry(c client.Client, registry, namespace string, scheme *runtime.Scheme, repos *regv1.APIRepositoryList) error {
	syncLog := logger.WithValues("registry_name", registry, "registry_ns", namespace)
	crImages, crImageNames, err := crImages(c, registry, namespace)
	if err != nil {
		syncLog.Error(err, "failed to get cr")
		return err
	}

	regImageNames := []string{}
	if repos != nil {
		for _, repo := range *repos {
			syncLog.Info("Repository", "Name", repo.Name)
			regImageNames = append(regImageNames, repo.Name)
		}
	}
	// Comparing ImageName and get New Images Name & Deleted Images Name
	newRepositories, deletedRepositories, existRepositories := compareRepositories(regImageNames, crImageNames, crImages)

	// For New Image, Insert Image and Versions Data from Repository
	repoCtl := &repoctl.RegistryRepository{}
	exreg := &regv1.ExternalRegistry{}
	if err := c.Get(context.TODO(), types.NamespacedName{Name: registry, Namespace: namespace}, exreg); err != nil {
		syncLog.Error(err, "")
	}

	for _, newImageName := range newRepositories {
		syncLog.Info("create new repository cr", "name", schemes.RepositoryName(newImageName, registry))
		newRepo := repos.GetRepository(newImageName)

		if err := repoCtl.ExtCreate(c, exreg, newRepo.Name, newRepo.Tags, scheme); err != nil {
			syncLog.Error(err, "failed to create repository")
			return err
		}
	}

	// For Deleted Image, Delete Image Data from Repository
	for _, deletedImageName := range deletedRepositories {
		syncLog.Info("delete repository cr", "name", schemes.RepositoryName(deletedImageName, registry))
		if err := repoCtl.ExtDelete(c, exreg, deletedImageName, scheme); err != nil {
			syncLog.Error(err, "failed to delete image")
			return err
		}
	}

	// For Exist Image, Compare tags List, Insert Version Data from Repository
	if err := patchRepository(c, registry, namespace, existRepositories, repos); err != nil {
		syncLog.Error(err, "failed to patch repository")
		return err
	}

	return nil
}

func getCRRepositories(c client.Client, registry, namespace string) (*regv1.RepositoryList, error) {
	reposCR := &regv1.RepositoryList{}

	label := map[string]string{}
	label["ext-registry"] = registry
	labelSelector := labels.SelectorFromSet(labels.Set(label))
	listOps := &client.ListOptions{
		Namespace:     namespace,
		LabelSelector: labelSelector,
	}

	if err := c.List(context.TODO(), reposCR, listOps); err != nil {
		logger.Error(err, "failed to list repository")
		return nil, err
	}

	return reposCR, nil
}

func crImages(c client.Client, registry, namespace string) ([]regv1.Repository, []string, error) {
	crImages := []regv1.Repository{}
	crImageNames := []string{}
	syncLog := logger.WithValues("registry_name", registry, "registry_ns", namespace)

	reposCR, err := getCRRepositories(c, registry, namespace)
	if err != nil {
		syncLog.Error(err, "failed to get cr repositories")
		return crImages, crImageNames, err
	}

	for _, image := range reposCR.Items {
		syncLog.Info("CR Repository", "Name", image.Spec.Name)
		crImages = append(crImages, image)
		crImageNames = append(crImageNames, image.Spec.Name)
	}

	return crImages, crImageNames, err
}

func compareRepositories(regImageNames, crImageNames []string, crImages []regv1.Repository) (
	newRepositories []string, deletedRepositories []string, existRepositories []regv1.Repository) {
	newRepositories, deletedRepositories, existRepositories = []string{}, []string{}, []regv1.Repository{}
	for _, regImage := range regImageNames {
		if !utils.Contains(crImageNames, regImage) {
			newRepositories = append(newRepositories, regImage)
		}
	}

	for _, crImage := range crImages {
		if !utils.Contains(regImageNames, crImage.Spec.Name) {
			deletedRepositories = append(deletedRepositories, crImage.Spec.Name)
		} else {
			existRepositories = append(existRepositories, crImage)
		}
	}

	return
}

func patchRepository(c client.Client, registry, namespace string, existRepositories []regv1.Repository, repos *regv1.APIRepositoryList) error {
	syncLog := logger.WithValues("registry_name", registry, "registry_ns", namespace)
	repoCtl := &repoctl.RegistryRepository{}

	for i, existRepo := range existRepositories {
		repoLog := syncLog.WithValues("repo", existRepo.Name)
		imageVersions := []regv1.ImageVersion{}
		curExistImageVersions := []string{}
		patchRepo := existRepo.DeepCopy()
		repo := repos.GetRepository(existRepo.Spec.Name)
		if repo == nil {
			continue
		}
		regVersions := repo.Tags

		for _, ver := range existRepo.Spec.Versions {
			if utils.Contains(regVersions, ver.Version) {
				repoLog.Info("exist", "version", ver)
				imageVersions = append(imageVersions, regv1.ImageVersion{Version: ver.Version, CreatedAt: ver.CreatedAt, Delete: ver.Delete, Signer: ver.Signer})
			}
		}

		for _, ver := range existRepo.Spec.Versions {
			curExistImageVersions = append(curExistImageVersions, ver.Version)
		}

		for _, regVersion := range regVersions {
			if !utils.Contains(curExistImageVersions, regVersion) {
				repoLog.Info("new", "version", regVersion)
				imageVersions = append(imageVersions, regv1.ImageVersion{Version: regVersion, CreatedAt: v1.Now(), Delete: false})
			}
		}

		patchRepo.Spec.Versions = imageVersions

		if err := repoCtl.Patch(c, &existRepositories[i], patchRepo); err != nil {
			repoLog.Error(err, "failed to patch repository")
			return err
		}
	}

	return nil
}
