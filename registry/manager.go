package registry

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/tmax-cloud/registry-operator/controllers/repoctl"
	"github.com/tmax-cloud/registry-operator/internal/schemes"
	"github.com/tmax-cloud/registry-operator/internal/utils"

	regv1 "github.com/tmax-cloud/registry-operator/api/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func getRegistryList(c client.Client) (*regv1.RegistryList, error) {
	regList := &regv1.RegistryList{}
	if err := c.List(context.TODO(), regList); err != nil {
		if strings.Contains(err.Error(), "the cache is not started") {
			logger.Info("the cache is not started, can not read objects")
			return nil, err
		}

		logger.Error(err, "failed to get regsitry list")
		return nil, err
	}

	return regList, nil
}

func printSyncedRegistry(syncedRegistry map[string]bool) {
	for registry, synced := range syncedRegistry {
		logger.Info(fmt.Sprintf("%-20s registry synced: %v", registry, synced))
	}
}

func syncAllRegistry(c client.Client, regList *regv1.RegistryList, syncedRegistry map[string]bool, scheme *runtime.Scheme) {
	for _, reg := range regList.Items {
		if syncedRegistry[reg.Name] {
			continue
		}

		logger.Info("synchronize registry", "name", reg.Name, "namespace", reg.Namespace)
		ra := NewRegistryApi(&reg)
		if ra == nil {
			logger.Error(fmt.Errorf("couldn't get registry api caller"), "failed to registry api caller")
			continue
		}

		if err := SyncRegistryImage(ra, c, &reg, scheme); err != nil {
			logger.Error(err, "failed to sync registry")
			continue
		}

		syncedRegistry[reg.Name] = true
	}
}

func allSynced(syncedRegistry map[string]bool) bool {
	for _, synced := range syncedRegistry {
		if !synced {
			return false
		}
	}

	return true
}

func SyncAllRegistry(c client.Client, scheme *runtime.Scheme) error {
	const MaxRetryCount = 10
	syncedRegistry := map[string]bool{}
	var regList *regv1.RegistryList
	var err error

	for retry := 0; retry < MaxRetryCount; retry++ {
		if regList, err = getRegistryList(c); err != nil {
			time.Sleep(1 * time.Second)
			logger.Info("retry to get registry list")
			continue
		}

		break
	}

	for _, reg := range regList.Items {
		syncedRegistry[reg.Name] = false
	}

	printSyncedRegistry(syncedRegistry)

	logger.Info("start to synchronize registries")
	for retry := 0; retry < MaxRetryCount; retry++ {
		if !allSynced(syncedRegistry) {
			syncAllRegistry(c, regList, syncedRegistry, scheme)
			printSyncedRegistry(syncedRegistry)
			continue
		}

		return nil
	}

	if !allSynced(syncedRegistry) {
		return errors.New("failed to synchronize all registies")
	}

	return nil
}

func getCRRepositories(c client.Client, reg *regv1.Registry) (*regv1.RepositoryList, error) {
	reposCR := &regv1.RepositoryList{}

	label := map[string]string{}
	label["registry"] = reg.Name
	labelSelector := labels.SelectorFromSet(labels.Set(label))
	listOps := &client.ListOptions{
		Namespace:     reg.Namespace,
		LabelSelector: labelSelector,
	}

	if err := c.List(context.TODO(), reposCR, listOps); err != nil {
		logger.Error(err, "failed to list repository")
		return nil, err
	}

	return reposCR, nil
}

func SyncRegistryImage(r *RegistryApi, c client.Client, reg *regv1.Registry, scheme *runtime.Scheme) error {
	crImages := []regv1.Repository{}
	crImageNames := []string{}
	syncLog := logger.WithValues("registry_name", reg.Name, "registry_ns", reg.Namespace)

	reposCR, err := getCRRepositories(c, reg)
	if err != nil {
		syncLog.Error(err, "failed to get cr repositories")
		return err
	}

	for _, image := range reposCR.Items {
		syncLog.Info("CR Repository", "Name", image.Spec.Name)
		crImages = append(crImages, image)
		crImageNames = append(crImageNames, image.Spec.Name)
	}

	repositories := r.Catalog()
	regImageNames := []string{}
	if repositories != nil {
		for _, imageName := range repositories.Repositories {
			syncLog.Info("Repository", "Name", imageName)
			regImageNames = append(regImageNames, imageName)
		}
	}

	// Comparing ImageName and get New Images Name & Deleted Images Name
	newRepositories, deletedRepositories, existRepositories := []string{}, []string{}, []regv1.Repository{}
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

	// For New Image, Insert Image and Versions Data from Repository
	repoCtl := &repoctl.RegistryRepository{}
	for _, newImageName := range newRepositories {
		syncLog.Info("create new repository cr", "name", schemes.RepositoryName(newImageName, reg.Name))
		newRepo := r.Tags(newImageName)
		if err := repoCtl.Create(c, reg, newRepo.Name, newRepo.Tags, scheme); err != nil {
			syncLog.Error(err, "failed to create repository")
			return err
		}
	}

	// For Deleted Image, Delete Image Data from Repository
	for _, deletedImageName := range deletedRepositories {
		syncLog.Info("delete repository cr", "name", schemes.RepositoryName(deletedImageName, reg.Name))
		if err := repoCtl.Delete(c, reg, deletedImageName, scheme); err != nil {
			syncLog.Error(err, "failed to delete image")
			return err
		}
	}

	// For Exist Image, Compare tags List, Insert Version Data from Repository
	for i, existRepo := range existRepositories {
		repoLog := syncLog.WithValues("repo", existRepo.Name)
		imageVersions := []regv1.ImageVersion{}
		curExistImageVersions := []string{}
		patchRepo := existRepo.DeepCopy()
		regVersions := r.Tags(existRepo.Spec.Name).Tags

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
				imageVersions = append(imageVersions, regv1.ImageVersion{Version: regVersion, CreatedAt: metav1.Now(), Delete: false})
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
