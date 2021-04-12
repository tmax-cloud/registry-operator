package docker

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"

	cmhttp "github.com/tmax-cloud/registry-operator/internal/common/http"
	"github.com/tmax-cloud/registry-operator/internal/utils"
	"github.com/tmax-cloud/registry-operator/pkg/image"
	"github.com/tmax-cloud/registry-operator/pkg/registry/base"
	"github.com/tmax-cloud/registry-operator/pkg/registry/sync"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var Logger = log.Log.WithName("docker-registry")

type Client struct {
	Name, Namespace string

	kClient     client.Client
	imageClient *image.Image
	scheme      *runtime.Scheme
}

// NewClient is api client of docker registry
func NewClient(c client.Client, registry types.NamespacedName, scheme *runtime.Scheme, httpClient *cmhttp.HttpClient) *Client {
	img, err := image.NewImage("", httpClient.URL, utils.EncryptBasicAuth(httpClient.Login.Username, httpClient.Login.Password), httpClient.CA)
	if err != nil {
		Logger.Error(err, "failed to create image client")
		return nil
	}
	return &Client{
		Name:        registry.Name,
		Namespace:   registry.Namespace,
		kClient:     c,
		imageClient: img,
		scheme:      scheme,
	}
}

// ListRepositories get repository list from registry server
func (c *Client) ListRepositories() *image.APIRepositories {
	return c.imageClient.Catalog()
}

// ListTags get tag list of repository from registry server
func (c *Client) ListTags(repository string) *image.APIRepository {
	if err := c.imageClient.SetImage(repository); err != nil {
		Logger.Error(err, "failed to set image")
		return nil
	}
	return c.imageClient.Tags()

}

// Synchronize synchronizes repository list between tmax.io.Repository resource and Registry server
func (c *Client) Synchronize() error {
	repos := c.ListRepositories()
	if repos == nil {
		return errors.New("failed to get repository list")
	}

	repoList := &image.APIRepositoryList{}

	for _, repo := range repos.Repositories {
		tags := c.ListTags(repo)
		if tags == nil {
			return errors.New("failed to get tag list")
		}
		repoList.AddRepository(*tags)
	}

	if err := sync.ExternalRegistry(c.kClient, c.Name, c.Namespace, c.scheme, repoList); err != nil {
		Logger.Error(err, "failed to synchronize external registry")
		return err
	}

	return nil
}

// GetManifest gets manifests of image in the registry
func (c *Client) GetManifest(image string) (*image.ImageManifest, error) {
	if err := c.imageClient.SetImage(image); err != nil {
		Logger.Error(err, "failed to set image")
		return nil, err
	}
	return c.imageClient.GetManifest()
}

// DeleteManifest deletes manifest in the registry
func (c *Client) DeleteManifest(image string, manifest *image.ImageManifest) error {
	if err := c.imageClient.SetImage(image); err != nil {
		Logger.Error(err, "failed to set image")
		return err
	}
	return c.imageClient.DeleteManifest(manifest)
}

// PutManifest updates manifest in the registry
func (c *Client) PutManifest(image string, manifest *image.ImageManifest) error {
	if err := c.imageClient.SetImage(image); err != nil {
		Logger.Error(err, "failed to set image")
		return err
	}
	return c.imageClient.PutManifest(manifest)
}

// ExistBlob returns true, if blob exists
func (c *Client) ExistBlob(repository, digest string) (bool, error) {
	image := fmt.Sprintf("%s@%s", repository, digest)
	if err := c.imageClient.SetImage(image); err != nil {
		Logger.Error(err, "failed to set image")
		return false, err
	}
	return c.imageClient.ExistBlob()
}

// PullBlob pulls and stores blob
func (c *Client) PullBlob(repository, digest string) (string, int64, error) {
	image := fmt.Sprintf("%s@%s", repository, digest)
	if err := c.imageClient.SetImage(image); err != nil {
		Logger.Error(err, "failed to set image")
		return "", 0, err
	}
	blob, size, err := c.imageClient.PullBlob()
	if err != nil {
		Logger.Error(err, "failed to pull blob")
		return "", 0, err
	}

	defer blob.Close()
	data, err := ioutil.ReadAll(blob)
	if err != nil {
		Logger.Error(err, "")
		return "", 0, err
	}

	file := path.Join(base.TempBlobsDir, repository, digest)
	if err := os.MkdirAll(path.Dir(file), os.ModePerm); err != nil {
		Logger.Error(err, "failed to make directory", "dir", path.Dir(file))
		return "", 0, err
	}
	Logger.Info("debug", "mkdir path", file)

	if err := ioutil.WriteFile(file, data, 0644); err != nil {
		Logger.Error(err, "failed to write file", "file", file)
		return "", 0, err
	}

	return file, size, nil
}

// PushBlob pushes and stores blob
func (c *Client) PushBlob(repository, digest, blobPath string, size int64) error {
	data, err := ioutil.ReadFile(blobPath)
	if err != nil {
		Logger.Error(err, "failed to read file", "file", blobPath)
		return err
	}

	image := fmt.Sprintf("%s@%s", repository, digest)
	if err := c.imageClient.SetImage(image); err != nil {
		Logger.Error(err, "failed to set image")
		return err
	}

	_, _, err = c.imageClient.PushBlob(data, size)
	if err != nil {
		Logger.Error(err, "failed to push blob")
		return err
	}

	return nil
}
