package registry

import (
	"context"
	"fmt"
	"strings"

	"github.com/tmax-cloud/registry-operator/controllers/repoctl"
	"github.com/tmax-cloud/registry-operator/internal/utils"

	regv1 "github.com/tmax-cloud/registry-operator/api/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var logz = log.Log.WithName("registry-api")

func AllRegistrySync(c client.Client, scheme *runtime.Scheme) error {
	// c, err := client.New(config.GetConfigOrDie(), client.Options{})
	// if err != nil {
	// 	logz.Error(err, "Unknown error")
	// }

	regList := &regv1.RegistryList{}

	if err := c.List(context.TODO(), regList, &client.ListOptions{}); err != nil {
		if strings.Contains(err.Error(), "the cache is not started") {
			logz.Info("the cache is not started, can not read objects")
			return err
		}

		logz.Error(err, "Get regsitry list is failed")
		return err
	}

	logz.Info("Registry list")
	for _, reg := range regList.Items {
		logz.Info("Registry", "Name", reg.Name, "Namespace", reg.Namespace)
		ra := NewRegistryApi(&reg)
		if ra == nil {
			return fmt.Errorf("couldn't get registry api caller")
		}
		logz.Info("Synchronize registry repositories")
		SyncRegistryImage(ra, c, &reg, scheme)
	}

	return nil
}

func SyncRegistryImage(r *RegistryApi, c client.Client, reg *regv1.Registry, scheme *runtime.Scheme) error {
	reposCR := &regv1.RepositoryList{}
	crImages := []regv1.Repository{}
	crImageNames := []string{}
	c.List(context.TODO(), reposCR)
	for _, image := range reposCR.Items {
		logger.Info("CR Repository", "Name", image.Spec.Name)
		crImages = append(crImages, image)
		crImageNames = append(crImageNames, image.Spec.Name)
	}

	repositories := r.Catalog()
	regImageNames := []string{}
	for _, imageName := range repositories.Repositories {
		logger.Info("Repository", "Name", imageName)
		regImageNames = append(regImageNames, imageName)
	}

	newRepositories, deletedRepositories, existRepositories := []string{}, []string{}, []regv1.Repository{}
	logger.Info("Comparing ImageName and get New Images Name & Deleted Images Name")
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

	repoCtl := &repoctl.RegistryRepository{}
	logger.Info("For New Image, Insert Image and Versions Data from Repository")
	for _, newImageName := range newRepositories {
		newRepo := r.Tags(newImageName)
		repoCtl.Create(c, reg, newRepo.Name, newRepo.Tags, scheme)
	}

	logger.Info("For Deleted Image, Delete Image Data from Repository")
	for _, deletedImageName := range deletedRepositories {
		repoCtl.Delete(c, reg, deletedImageName, scheme)
	}

	logger.Info("For Exist Image, Compare tags List, Insert Version Data from Repository")

	for i, existRepo := range existRepositories {
		imageVersions := []regv1.ImageVersion{}
		existImageVersions := []string{}
		patchRepo := existRepo.DeepCopy()
		regVersions := r.Tags(existRepo.Spec.Name).Tags

		for _, ver := range existRepo.Spec.Versions {
			if utils.Contains(regVersions, ver.Version) {
				logger.Info("exist", "version", ver)
				imageVersions = append(imageVersions, regv1.ImageVersion{Version: ver.Version, CreatedAt: ver.CreatedAt})
			}
		}

		for _, ver := range existRepo.Spec.Versions {
			existImageVersions = append(existImageVersions, ver.Version)
		}

		for _, regVersion := range regVersions {
			if !utils.Contains(existImageVersions, regVersion) {
				logger.Info("new", "version", regVersion)
				imageVersions = append(imageVersions, regv1.ImageVersion{Version: regVersion, CreatedAt: metav1.Now()})
			}
		}

		patchRepo.Spec.Versions = imageVersions

		repoCtl.Patch(c, &existRepositories[i], patchRepo)

	}

	return nil
}
