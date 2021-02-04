package controllers

import (
	"fmt"

	"github.com/tmax-cloud/registry-operator/internal/utils"
	regApi "github.com/tmax-cloud/registry-operator/registry"

	regv1 "github.com/tmax-cloud/registry-operator/api/v1"
	"github.com/tmax-cloud/registry-operator/controllers/regctl"
	"github.com/tmax-cloud/registry-operator/controllers/repoctl"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func sweepImages(c client.Client, reg *regv1.Registry, repo *regv1.Repository) error {
	repoName := repo.Spec.Name
	logz := log.WithValues("repository_namespace", repo.Namespace, "repository_name", repoName)
	images := repo.Spec.Versions
	delTags := []string{}

	for _, image := range images {
		if image.Delete {
			delTags = append(delTags, image.Version)
		}
	}

	if len(delTags) == 0 {
		logz.Info("repository_is_up_to_date")
		return nil
	}

	deletedTags, err := deleteImagesInRepo(c, reg, repoName, delTags)
	if err != nil {
		return err
	}

	logz.Info("patch_repository_cr")
	if err := patchRepo(c, reg, repo, deletedTags); err != nil {
		return err
	}

	logz.Info("garbage_collect")
	if err := garbageCollect(c, reg); err != nil {
		return err
	}

	return nil
}

func sweepRegistryRepo(c client.Client, reg *regv1.Registry, repoName string) error {
	ra := regApi.NewRegistryApi(reg)
	if ra == nil {
		return fmt.Errorf("couldn't get registry api caller")
	}

	tags := ra.Tags(repoName).Tags
	if tags == nil {
		return nil
	}

	log.Info("delete_images")
	deletedTags, err := deleteImagesInRepo(c, reg, repoName, tags)
	if err != nil {
		return err
	}

	for _, tag := range deletedTags {
		log.Info("delete", "repository_namespace", reg.Namespace, "repository_name", repoName, "tag", tag)
	}

	log.Info("garbage_collect")
	if err := garbageCollect(c, reg); err != nil {
		return err
	}

	log.Info("sweep_repo_success")

	return nil
}

func deleteImagesInRepo(c client.Client, reg *regv1.Registry, repoName string, tags []string) ([]string, error) {
	ra := regApi.NewRegistryApi(reg)
	if ra == nil {
		return nil, fmt.Errorf("couldn't get registry api caller")
	}
	deletedTags := []string{}

	for _, tag := range tags {
		log.Info("repository", "tag", tag)
		digest, err := ra.DockerContentDigest(repoName, tag)
		if err != nil {
			log.Error(err, "")
			return deletedTags, err
		}
		log.Info("get", "digest", digest)

		if err := ra.DeleteManifest(repoName, digest); err != nil {
			log.Error(err, "")
			return deletedTags, err
		}
		deletedTags = append(deletedTags, tag)
	}

	return deletedTags, nil
}

func garbageCollect(c client.Client, reg *regv1.Registry) error {
	podName, err := regctl.PodName(c, reg)
	if err != nil {
		return err
	}

	cmder := regApi.NewCommander(podName, reg.Namespace)
	out, err := cmder.GarbageCollect()
	if err != nil {
		log.Error(err, "exec")
		return err
	}

	log.Info("exec", "stdout", out.Outbuf.String(), "stderr", out.Errbuf.String())
	return nil
}

func patchRepo(c client.Client, reg *regv1.Registry, repo *regv1.Repository, deletedTags []string) error {
	repoctl := &repoctl.RegistryRepository{}
	patchRepo := repo.DeepCopy()
	patchImageList := []regv1.ImageVersion{}

	for _, image := range repo.Spec.Versions {
		if !utils.Contains(deletedTags, image.Version) {
			patchImageList = append(patchImageList, image)
		}
	}

	patchRepo.Spec.Versions = patchImageList
	if err := repoctl.Patch(c, repo, patchRepo); err != nil {
		return err
	}

	return nil
}
