package registry

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"reflect"
	"strings"
	"time"

	"github.com/tmax-cloud/registry-operator/controllers/repoctl"
	"github.com/tmax-cloud/registry-operator/internal/schemes"
	"github.com/tmax-cloud/registry-operator/internal/utils"
	"github.com/tmax-cloud/registry-operator/pkg/image"
	"github.com/tmax-cloud/registry-operator/pkg/trust"

	regv1 "github.com/tmax-cloud/registry-operator/api/v1"
	corev1 "k8s.io/api/core/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"

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
		if syncedRegistry[registryNamespacedName(reg.Namespace, reg.Name)] {
			continue
		}

		logger.Info("synchronize registry", "name", reg.Name, "namespace", reg.Namespace)
		ra := NewRegistryApi(&reg)
		if ra == nil {
			logger.Error(fmt.Errorf("couldn't get registry api caller"), "failed to registry api caller")
			continue
		}

		if err := SyncRegistry(ra, c, &reg, scheme); err != nil {
			logger.Error(err, "failed to sync registry")
			continue
		}

		syncedRegistry[registryNamespacedName(reg.Namespace, reg.Name)] = true
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

func registryNamespacedName(namespace, name string) string {
	return fmt.Sprintf("%s/%s", namespace, name)
}

// SyncAllRegistry synchronizes all registries
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
		syncedRegistry[registryNamespacedName(reg.Namespace, reg.Name)] = false
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
		return errors.New("failed to synchronize all registries")
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

// SyncRegistry synchronizes custom resource repository based on all repositories in registry server
func SyncRegistry(r *RegistryApi, c client.Client, reg *regv1.Registry, scheme *runtime.Scheme) error {
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

func SyncAllRepoSigner(c client.Client, scheme *runtime.Scheme) error {
	skList := &regv1.SignerKeyList{}

	if err := c.List(context.TODO(), skList); err != nil {
		logger.Error(err, "failed to get signer key list")
		return err
	}

	for _, sk := range skList.Items {
		for gun, target := range sk.Spec.Targets {
			logger.Info("debug", "GUN", gun)
			host, repoName, err := parseGUN(gun)
			logger.Info("debug", "host", host, "repoName", repoName)
			if err != nil {
				logger.Error(err, fmt.Sprintf("failed to parse url: %s", gun))
				continue
			}

			reg, err := findRegistryByHost(c, host)
			if err != nil {
				logger.Error(err, fmt.Sprintf("failed to find registry by %s host", host))
				continue
			}

			// Get secret
			regSecret := &corev1.Secret{}
			if err := c.Get(context.TODO(), types.NamespacedName{Name: schemes.SubresourceName(reg, schemes.SubTypeRegistryDCJSecret), Namespace: reg.Namespace}, regSecret); err != nil {
				logger.Error(err, "")
				continue
			}

			repo := &regv1.Repository{}
			if err := c.Get(context.TODO(), types.NamespacedName{Name: schemes.RepositoryName(repoName, reg.Name), Namespace: reg.Namespace}, repo); err != nil {
				logger.Error(err, fmt.Sprintf("failed to get repository: name(%s), ns(%s)", schemes.RepositoryName(repoName, reg.Name), reg.Namespace))
				continue
			}
			originRepo := repo.DeepCopy()

			for i, tag := range repo.Spec.Versions {
				imageName := fmt.Sprintf("%s:%s", gun, tag.Version)
				img, err := image.NewImage(imageName, reg.Status.ServerURL, "", nil)
				if err != nil {
					logger.Error(err, "")
					continue
				}

				basicAuth, err := utils.ParseBasicAuth(regSecret, img.Host)
				if err != nil {
					logger.Error(err, "")
					continue
				}
				img.BasicAuth = basicAuth

				not, err := trust.NewReadOnly(img, reg.Status.NotaryURL, fmt.Sprintf("/tmp/notary/%s", utils.RandomString(10)))
				if err != nil {
					logger.Error(err, "failed to create notary client")
					continue
				}

				signedRepo, err := not.GetSignedMetadata(tag.Version)
				if err != nil {
					logger.Error(err, "failed to get target metadata")
					continue
				}
				logger.Info("debug", "signedRepo", fmt.Sprintf("%+v", signedRepo))

			AdminKeyLoop:
				for _, adminKey := range signedRepo.AdministrativeKeys {
					if adminKey.Name == "Repository" {
						for _, k := range adminKey.Keys {
							if k.ID == target.ID {
								if repo.Spec.Versions[i].Signer == sk.Name {
									logger.Info("repository's signer is latest", "repository", repo.Namespace+"/"+repo.Name, "signer", sk.Name)
									break AdminKeyLoop
								}
								logger.Info("update required", "repository", repo.Namespace+"/"+repo.Name, "signer", sk.Name)
								repo.Spec.Versions[i].Signer = sk.Name
								break AdminKeyLoop
							}
						}
					}
				}
			}

			if !reflect.DeepEqual(originRepo.Spec, repo.Spec) {
				logger.Info("update repository spec", "repository", repo.Namespace+"/"+repo.Name)
				if err := c.Update(context.TODO(), repo); err != nil {
					logger.Error(err, "failed to update repository", "namespace", repo.Namespace, "name", repo.Name)
					continue
				}
			}
		}
	}

	return nil
}

func parseGUN(gun string) (host, repoName string, err error) {
	if gun == "" {
		return
	}

	if !strings.HasPrefix(gun, "http://") && !strings.HasPrefix(gun, "https://") {
		gun = "http://" + gun
	}

	var u *url.URL
	u, err = url.Parse(gun)
	if err != nil {
		logger.Error(err, fmt.Sprintf("failed to parse url: %s", gun))
		return
	}

	host = u.Host
	repoName = u.Path
	repoName = strings.TrimPrefix(repoName, "/")
	return
}

func findRegistryByHost(c client.Client, hostname string) (*regv1.Registry, error) {
	regList := &regv1.RegistryList{}
	if err := c.List(context.TODO(), regList); err != nil {
		return nil, err
	}

	var targetReg regv1.Registry
	targetFound := false
	for _, r := range regList.Items {
		logger.Info(r.Name)
		serverUrl := strings.TrimPrefix(r.Status.ServerURL, "https://")
		serverUrl = strings.TrimPrefix(serverUrl, "http://")
		serverUrl = strings.TrimSuffix(serverUrl, "/")

		if serverUrl == hostname {
			targetReg = r
			targetFound = true
		}
	}

	if !targetFound {
		return nil, fmt.Errorf("target registry is not an internal registry")
	}

	return &targetReg, nil
}
