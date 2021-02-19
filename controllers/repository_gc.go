package controllers

import (
	"fmt"

	"github.com/tmax-cloud/registry-operator/internal/utils"
	"github.com/tmax-cloud/registry-operator/pkg/registry/inter"
	"k8s.io/apimachinery/pkg/runtime"

	regv1 "github.com/tmax-cloud/registry-operator/api/v1"
	"github.com/tmax-cloud/registry-operator/controllers/regctl"
	"github.com/tmax-cloud/registry-operator/controllers/repoctl"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Delete versions with delete value true in repository
func sweepImages(c client.Client, reg *regv1.Registry, scheme *runtime.Scheme, repo *regv1.Repository) error {
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

	deletedTags, err := deleteImagesInRepo(c, reg, scheme, repoName, delTags)
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

// Delete all tags and gc
func sweepRegistryRepo(c client.Client, reg *regv1.Registry, scheme *runtime.Scheme, repoName string) error {
	log = log.WithValues("namespace", reg.Namespace, "name", reg.Name, "repository", repoName)
	regClient, err := inter.GetClient(c, reg, scheme)
	if err != nil {
		log.Error(err, "failed to get reg client")
		return err
	}

	tags := regClient.ListTags(repoName)

	log.Info("delete_images")
	deletedTags, err := deleteImagesInRepo(c, reg, scheme, repoName, tags.Tags)
	if err != nil {
		log.Error(err, "failed to delete tags", "tags", tags.Tags)
		return err
	}

	for _, tag := range deletedTags {
		log.Info("delete", "tag", tag)
	}

	log.Info("garbage_collect")
	if err := garbageCollect(c, reg); err != nil {
		return err
	}

	log.Info("sweep_repo_success")

	return nil
}

// Delete all tags
func deleteImagesInRepo(c client.Client, reg *regv1.Registry, scheme *runtime.Scheme, repoName string, tags []string) ([]string, error) {
	log = log.WithValues("namespace", reg.Namespace, "name", reg.Name, "repository", repoName)
	regClient, err := inter.GetClient(c, reg, scheme)
	if err != nil {
		log.Error(err, "failed to get reg client")
		return nil, err
	}

	deletedTags := []string{}

	for _, tag := range tags {
		log.Info("repository", "tag", tag)
		image := fmt.Sprintf("%s:%s", repoName, tag)
		mf, err := regClient.GetManifest(image)
		if err != nil {
			log.Error(err, "failed to get manifest")
			return deletedTags, err
		}
		log.Info("get", "digest", mf.Digest)

		if err := regClient.DeleteManifest(image, mf); err != nil {
			log.Error(err, "failed to delete manifest")
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

	cmder := inter.NewCommander(podName, reg.Namespace)
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
