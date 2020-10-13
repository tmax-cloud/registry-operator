package repository

import (
	"hypercloud-operator-go/internal/utils"
	tmaxv1 "hypercloud-operator-go/pkg/apis/tmax/v1"
	"hypercloud-operator-go/pkg/controller/regctl"
	"hypercloud-operator-go/pkg/controller/repoctl"
	regApi "hypercloud-operator-go/pkg/registry"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

func sweepImages(c client.Client, reg *tmaxv1.Registry, repo *tmaxv1.Repository) error {
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

	logz.Info("restart_registry", "ns/name", reg.Namespace+"/"+reg.Name)
	if err := regctl.DeletePod(c, reg); err != nil {
		return err
	}

	return nil
}

func sweepRegistryRepo(c client.Client, reg *tmaxv1.Registry, repoName string) error {
	ra := regApi.NewRegistryApi(reg)

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

	log.Info("restart", "registry", reg.Namespace+"/"+reg.Name)
	if err := regctl.DeletePod(c, reg); err != nil {
		return err
	}

	log.Info("sweep_repo_success")

	return nil
}

func deleteImagesInRepo(c client.Client, reg *tmaxv1.Registry, repoName string, tags []string) ([]string, error) {
	ra := regApi.NewRegistryApi(reg)
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

func garbageCollect(c client.Client, reg *tmaxv1.Registry) error {
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

func patchRepo(c client.Client, reg *tmaxv1.Registry, repo *tmaxv1.Repository, deletedTags []string) error {
	repoctl := &repoctl.RegistryRepository{}
	patchRepo := repo.DeepCopy()
	patchImageList := []tmaxv1.ImageVersion{}

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
