package dockerhub

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"

	"github.com/tmax-cloud/registry-operator/internal/common/auth"
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

var Logger = log.Log.WithName("dockerhub-registry")

type Client struct {
	Name, Namespace string

	kClient      client.Client
	dockerClient *cmhttp.HttpClient
	imageClient  *image.Image
	scheme       *runtime.Scheme
}

// NewClient is api client of internal registry
func NewClient(c client.Client, registry types.NamespacedName, scheme *runtime.Scheme, httpClient *cmhttp.HttpClient) *Client {
	client := &Client{
		Name:      registry.Name,
		Namespace: registry.Namespace,
		kClient:   c,
		scheme:    scheme,
	}

	client.dockerClient = cmhttp.NewHTTPClient(dockerHubURL, httpClient.Login.Username, httpClient.Login.Password, nil, true)
	if err := client.LoginDockerHub(); err != nil {
		Logger.Error(err, "failed to login dockerhub")
		return nil
	}

	var err error
	client.imageClient, err = image.NewImage("", "", utils.EncryptBasicAuth(httpClient.Login.Username, httpClient.Login.Password), httpClient.CA)
	if err != nil {
		Logger.Error(err, "failed to create image client")
		return nil
	}

	return client
}

func (c *Client) LoginDockerHub() error {
	author := &Authorizer{
		Username: c.dockerClient.Login.Username,
		Password: c.dockerClient.Login.Password,
	}

	authData, err := json.Marshal(author)
	if err != nil {
		Logger.Error(err, "failed to marshal")
		return err
	}

	Logger.Info("call", "method", http.MethodPost, "api", loginURL())
	req, err := http.NewRequest(http.MethodPost, loginURL(), bytes.NewReader(authData))
	if err != nil {
		Logger.Error(err, "")
		return err
	}
	req.Header.Set("Content-Type", utils.ContentTypeJSON)

	res, err := c.dockerClient.Do(req)
	if err != nil {
		Logger.Error(err, "failed to request", "url", req.URL.String())
		return err
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		Logger.Error(err, "")
		return err
	}

	token := &auth.TokenResponse{}
	if err := json.Unmarshal(body, token); err != nil {
		Logger.Error(err, "failed to unmarshal")
		return err
	}

	c.dockerClient.Token = auth.Token{Type: auth.TokenTypeBearer, Value: token.Token}

	return nil
}

func (c *Client) ListNamespaces() []string {
	Logger.Info("call", "method", http.MethodGet, "api", listNamespacesURL())
	req, err := http.NewRequest(http.MethodGet, listNamespacesURL(), nil)
	if err != nil {
		fmt.Println(err.Error())
		Logger.Error(err, "")
		return []string{}
	}

	if c.dockerClient.Token.Type == "" || c.dockerClient.Token.Value == "" {
		if err := c.LoginDockerHub(); err != nil {
			fmt.Println(err.Error())
			Logger.Error(err, "")
			return []string{}
		}
	}

	req.Header.Set("Authorization", fmt.Sprintf("%s %s", c.dockerClient.Token.Type, c.dockerClient.Token.Value))

	res, err := c.dockerClient.Do(req)
	if err != nil {
		fmt.Println(err.Error())
		Logger.Error(err, "")
		return []string{}
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err.Error())
		Logger.Error(err, "")
		return []string{}
	}

	namespaces := &NamespacesResponse{}
	if err := json.Unmarshal(body, namespaces); err != nil {
		fmt.Println(err.Error())
		Logger.Error(err, "failed to unmarshal")
		return []string{}
	}

	return namespaces.Namespaces
}

func (c *Client) listRepositories(namespace string, page, page_size int) ([]string, string) {
	Logger.Info("call", "method", http.MethodGet, "api", listRepositoriesURL(namespace, page, page_size))
	req, err := http.NewRequest(http.MethodGet, listRepositoriesURL(namespace, page, page_size), nil)
	if err != nil {
		Logger.Error(err, "")
		return []string{}, ""
	}

	if c.dockerClient.Token.Type == "" || c.dockerClient.Token.Value == "" {
		if err := c.LoginDockerHub(); err != nil {
			Logger.Error(err, "")
			return []string{}, ""
		}
	}

	req.Header.Set("Authorization", fmt.Sprintf("%s %s", c.dockerClient.Token.Type, c.dockerClient.Token.Value))

	res, err := c.dockerClient.Do(req)
	if err != nil {
		Logger.Error(err, "")
		return []string{}, ""
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		Logger.Error(err, "")
		return []string{}, ""
	}

	reposRes := &RepositoriesResponse{}
	if err := json.Unmarshal(body, reposRes); err != nil {
		Logger.Error(err, "failed to unmarshal")
		return []string{}, ""
	}

	repos := []string{}
	for _, repo := range reposRes.Repositories {
		repos = append(repos, path.Join(namespace, repo.Name))
	}

	return repos, reposRes.Next
}

// ListRepositories get repository list from registry server
func (c *Client) ListRepositories() *image.APIRepositories {
	namespaces := c.ListNamespaces()

	repos := &image.APIRepositories{}
	for _, namespace := range namespaces {
		page := 1
		for {
			list, next := c.listRepositories(namespace, page, 100)
			repos.Repositories = append(repos.Repositories, list...)
			if next == "" {
				break
			}
			page++
		}
	}

	return repos
}

func (c *Client) listTags(namespace, repo string, page, page_size int) ([]string, string) {
	Logger.Info("call", "method", http.MethodGet, "api", listTagsURL(namespace, repo, page, page_size))
	req, err := http.NewRequest(http.MethodGet, listTagsURL(namespace, repo, page, page_size), nil)
	if err != nil {
		Logger.Error(err, "")
		return []string{}, ""
	}

	if c.dockerClient.Token.Type == "" || c.dockerClient.Token.Value == "" {
		if err := c.LoginDockerHub(); err != nil {
			Logger.Error(err, "")
			return []string{}, ""
		}
	}

	req.Header.Set("Authorization", fmt.Sprintf("%s %s", c.dockerClient.Token.Type, c.dockerClient.Token.Value))

	res, err := c.dockerClient.Do(req)
	if err != nil {
		Logger.Error(err, "")
		return []string{}, ""
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		Logger.Error(err, "")
		return []string{}, ""
	}

	tagsRes := &TagsResponse{}
	if err := json.Unmarshal(body, tagsRes); err != nil {
		Logger.Error(err, "failed to unmarshal")
		return []string{}, ""
	}

	tags := []string{}
	for _, tag := range tagsRes.Tags {
		tags = append(tags, tag.Name)
	}

	return tags, tagsRes.Next
}

// ListTags get tag list of repository from registry server
func (c *Client) ListTags(repository string) *image.APIRepository {
	namespace, repo, err := ParseName(repository)
	if err != nil {
		Logger.Error(err, "failed to parse repository name", "repository", repository)
		return &image.APIRepository{}
	}

	tags := &image.APIRepository{Name: repository}
	page := 1
	for {
		list, next := c.listTags(namespace, repo, page, 100)
		tags.Tags = append(tags.Tags, list...)
		if next == "" {
			break
		}
	}

	return tags
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

// PullBlob pulls and stores blob
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
