package manager

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"reflect"
	"strings"
	"time"

	"github.com/tmax-cloud/registry-operator/internal/common/certs"
	cmhttp "github.com/tmax-cloud/registry-operator/internal/common/http"
	"github.com/tmax-cloud/registry-operator/internal/schemes"
	"github.com/tmax-cloud/registry-operator/internal/utils"
	"github.com/tmax-cloud/registry-operator/pkg/image"
	"github.com/tmax-cloud/registry-operator/pkg/registry/base"
	"github.com/tmax-cloud/registry-operator/pkg/registry/inter/factory"
	"github.com/tmax-cloud/registry-operator/pkg/trust"

	regv1 "github.com/tmax-cloud/registry-operator/api/v1"
	corev1 "k8s.io/api/core/v1"

	"k8s.io/apimachinery/pkg/types"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var logger = log.Log.WithName("registry-manager")

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
		if err := SyncRegistry(c, &reg, scheme); err != nil {
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

// SyncRegistry synchronizes custom resource repository based on all repositories in registry server
func SyncRegistry(c client.Client, reg *regv1.Registry, scheme *runtime.Scheme) error {
	imagePullSecret := schemes.SubresourceName(reg, schemes.SubTypeRegistryDCJSecret)
	basic, err := utils.GetBasicAuth(imagePullSecret, reg.Namespace, reg.Status.ServerURL)
	if err != nil {
		logger.Error(err, "failed to get basic auth")
		return err
	}

	username, password := utils.DecodeBasicAuth(basic)
	caSecret, err := certs.GetRootCert(reg.Namespace)
	if err != nil {
		logger.Error(err, "failed to get root CA")
		return err
	}
	ca, _ := certs.CAData(caSecret)

	caSecret, err = certs.GetSystemKeycloakCert(c)
	if err == nil {
		kca, _ := certs.CAData(caSecret)
		ca = append(ca, kca...)
	}

	syncFactory := factory.NewRegistryFactory(
		c,
		types.NamespacedName{Name: reg.Name, Namespace: reg.Namespace},
		scheme,
		cmhttp.NewHTTPClient(
			reg.Status.ServerURL,
			username, password,
			ca,
			len(ca) == 0,
		),
	)

	syncClient := syncFactory.Create(regv1.RegistryTypeHpcdRegistry).(base.Synchronizable)
	if err := syncClient.Synchronize(); err != nil {
		logger.Error(err, "failed to synchronize external registry")
		return err
	}

	return nil
}

// SyncAllRepoSigner synchronizes signer in all repositories
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
