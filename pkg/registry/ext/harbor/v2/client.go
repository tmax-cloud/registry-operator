package v2

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"strings"

	cmhttp "github.com/tmax-cloud/registry-operator/internal/common/http"
	"github.com/tmax-cloud/registry-operator/internal/utils"
	"github.com/tmax-cloud/registry-operator/pkg/image"
	"github.com/tmax-cloud/registry-operator/pkg/registry/base"
	"github.com/tmax-cloud/registry-operator/pkg/registry/ext"
	"github.com/tmax-cloud/registry-operator/pkg/registry/sync"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NewClient is api client of harbor v2 registry
func NewClient(c client.Client, namespacedName types.NamespacedName, scheme *runtime.Scheme, httpClient *cmhttp.HttpClient) *Client {
	img, err := image.NewImage("", httpClient.URL, utils.EncryptBasicAuth(httpClient.Login.Username, httpClient.Login.Password), httpClient.CA)
	if err != nil {
		ext.Logger.Error(err, "failed to create image client")
		return nil
	}
	return &Client{
		Name:        namespacedName.Name,
		Namespace:   namespacedName.Namespace,
		HttpClient:  httpClient,
		kClient:     c,
		imageClient: img,
		scheme:      scheme,
	}
}

type Client struct {
	Name, Namespace string

	*cmhttp.HttpClient
	imageClient *image.Image
	kClient     client.Client
	scheme      *runtime.Scheme
}

// SetAuth sets Authorization header
func (c *Client) SetAuth(req *http.Request) {
	req.Header.Add("Authorization", "Basic "+utils.HTTPEncodeBasicAuth(c.Login.Username, c.Login.Password))
}

// ListRepositories get repository list from registry server
func (c *Client) ListRepositories() *image.APIRepositories {
	ext.Logger.Info("call", "method", http.MethodGet, "api", listProjectsURL(c.URL))
	req, err := http.NewRequest(http.MethodGet, listProjectsURL(c.URL), nil)
	if err != nil {
		ext.Logger.Error(err, "")
		return &image.APIRepositories{}
	}

	if c.Login.Username != "" && c.Login.Password != "" {
		c.SetAuth(req)
	}

	res, err := c.Client.Do(req)
	if err != nil {
		ext.Logger.Error(err, "")
		return &image.APIRepositories{}
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		ext.Logger.Error(err, "")
		return &image.APIRepositories{}
	}

	// ext.Logger.Info("contents", "projects", string(body))
	projects := []Project{}
	if err := json.Unmarshal(body, &projects); err != nil {
		ext.Logger.Error(err, "failed to unmarshal project", "body", string(body))
		return &image.APIRepositories{}
	}

	extRepos := &image.APIRepositories{}

	for _, proj := range projects {
		req, err := http.NewRequest(http.MethodGet, listRepositoriessURL(c.URL, proj.Name), nil)
		if err != nil {
			ext.Logger.Error(err, "")
			return &image.APIRepositories{}
		}

		c.SetAuth(req)

		res, err := c.Client.Do(req)
		if err != nil {
			ext.Logger.Error(err, "")
			return &image.APIRepositories{}
		}

		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			ext.Logger.Error(err, "")
			return &image.APIRepositories{}
		}

		// ext.Logger.Info("contents", "repositories", string(body))

		repos := []Repository{}

		if err := json.Unmarshal(body, &repos); err != nil {
			ext.Logger.Error(err, "failed to unmarshal registry's repository")
			return &image.APIRepositories{}
		}

		for _, repo := range repos {
			extRepos.Repositories = append(extRepos.Repositories, repo.Name)
		}
	}

	ext.Logger.Info("list", "repositories", extRepos.Repositories)

	return extRepos
}

func projectAndRepositoryName(repositoryFullName string) (project, repository string) {
	slashIdx := strings.Index(repositoryFullName, "/")
	if slashIdx < 0 {
		return
	}

	project = repositoryFullName[:slashIdx]
	repository = repositoryFullName[slashIdx+1:]
	return
}

// ListTags get tag list of repository from registry server
func (c *Client) ListTags(repository string) *image.APIRepository {
	project, repoName := projectAndRepositoryName(repository)

	ext.Logger.Info("call", "method", http.MethodGet, "api", listTagsURL(c.URL, project, repoName))
	req, err := http.NewRequest(http.MethodGet, listTagsURL(c.URL, project, repoName), nil)
	if err != nil {
		ext.Logger.Error(err, "")
		return nil
	}

	c.SetAuth(req)

	res, err := c.Client.Do(req)
	if err != nil {
		ext.Logger.Error(err, "")
		return nil
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		ext.Logger.Error(err, "")
		return nil
	}

	// ext.Logger.Info("contents", "artifact", string(body))
	artifacts := []Artifact{}
	if err := json.Unmarshal(body, &artifacts); err != nil {
		ext.Logger.Error(err, "failed to unmarshal artifact")
		return nil
	}

	regRepo := &image.APIRepository{Name: repository}
	for _, artifact := range artifacts {
		if strings.ToUpper(artifact.Type) == "IMAGE" {
			for _, tag := range artifact.Tags {
				regRepo.Tags = append(regRepo.Tags, tag.Name)
			}

			ext.Logger.Info("list", "tags", regRepo.Tags)
			break
		}
	}

	return regRepo
}

// Synchronize synchronizes repository list between tmax.io.Repository resource and Registry server
func (c *Client) Synchronize() error {
	repos := c.ListRepositories()
	repoList := &image.APIRepositoryList{}

	for _, repo := range repos.Repositories {
		tags := c.ListTags(repo)
		repoList.AddRepository(*tags)
	}

	if err := sync.ExternalRegistry(c.kClient, c.Name, c.Namespace, c.scheme, repoList); err != nil {
		ext.Logger.Error(err, "failed to synchronize external registry")
		return err
	}

	return nil
}

// GetManifest gets manifests of image in the registry
func (c *Client) GetManifest(image string) (*image.ImageManifest, error) {
	if err := c.imageClient.SetImage(image); err != nil {
		ext.Logger.Error(err, "failed to set image")
		return nil, err
	}
	return c.imageClient.GetManifest()
}

// DeleteManifest deletes manifest in the registry
func (c *Client) DeleteManifest(image string, manifest *image.ImageManifest) error {
	if err := c.imageClient.SetImage(image); err != nil {
		ext.Logger.Error(err, "failed to set image")
		return err
	}
	return c.imageClient.DeleteManifest(manifest)
}

// PutManifest updates manifest in the registry
func (c *Client) PutManifest(image string, manifest *image.ImageManifest) error {
	if err := c.imageClient.SetImage(image); err != nil {
		ext.Logger.Error(err, "failed to set image")
		return err
	}
	return c.imageClient.PutManifest(manifest)
}

// ExistBlob checks if blob exists
func (c *Client) ExistBlob(repository, digest string) (bool, error) {
	image := fmt.Sprintf("%s@%s", repository, digest)
	if err := c.imageClient.SetImage(image); err != nil {
		ext.Logger.Error(err, "failed to set image")
		return false, err
	}
	return c.imageClient.ExistBlob()
}

// PullBlob pulls and stores blob
func (c *Client) PullBlob(repository, digest string) (string, int64, error) {
	image := fmt.Sprintf("%s@%s", repository, digest)
	if err := c.imageClient.SetImage(image); err != nil {
		ext.Logger.Error(err, "failed to set image")
		return "", 0, err
	}
	blob, size, err := c.imageClient.PullBlob()
	if err != nil {
		ext.Logger.Error(err, "failed to pull blob")
		return "", 0, err
	}

	defer blob.Close()
	data, err := ioutil.ReadAll(blob)
	if err != nil {
		ext.Logger.Error(err, "")
		return "", 0, err
	}

	file := path.Join(base.TempBlobsDir, repository, digest)
	if err := os.MkdirAll(path.Dir(file), os.ModePerm); err != nil {
		ext.Logger.Error(err, "failed to make directory", "dir", path.Dir(file))
		return "", 0, err
	}
	ext.Logger.Info("debug", "mkdir path", file)

	if err := ioutil.WriteFile(file, data, 0644); err != nil {
		ext.Logger.Error(err, "failed to write file", "file", file)
		return "", 0, err
	}

	return file, size, nil
}

// PushBlob pushes and stores blob
func (c *Client) PushBlob(repository, digest, blobPath string, size int64) error {
	data, err := ioutil.ReadFile(blobPath)
	if err != nil {
		ext.Logger.Error(err, "failed to read file", "file", blobPath)
		return err
	}

	image := fmt.Sprintf("%s@%s", repository, digest)
	if err := c.imageClient.SetImage(image); err != nil {
		ext.Logger.Error(err, "failed to set image")
		return err
	}

	_, _, err = c.imageClient.PushBlob(data, size)
	if err != nil {
		ext.Logger.Error(err, "failed to push blob")
		return err
	}

	return nil
}
