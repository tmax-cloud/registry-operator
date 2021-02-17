package inter

import (
	regv1 "github.com/tmax-cloud/registry-operator/api/v1"
	cmhttp "github.com/tmax-cloud/registry-operator/internal/common/http"
	"github.com/tmax-cloud/registry-operator/internal/utils"
	"github.com/tmax-cloud/registry-operator/pkg/image"
	"github.com/tmax-cloud/registry-operator/pkg/registry/sync"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var Logger = log.Log.WithName("inter-registry")

// NewClient is api client of internal registry
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

type Client struct {
	Name, Namespace string

	kClient     client.Client
	imageClient *image.Image
	scheme      *runtime.Scheme
}

// ListRepositories get repository list from registry server
func (c *Client) ListRepositories() *regv1.APIRepositories {
	return c.imageClient.Catalog()
}

// ListTags get tag list of repository from registry server
func (c *Client) ListTags(repository string) *regv1.APIRepository {
	c.imageClient.SetImage(repository)
	return c.imageClient.Tags()

}

// Synchronize synchronizes repository list between tmax.io.Repository resource and Registry server
func (c *Client) Synchronize() error {
	repos := c.ListRepositories()
	repoList := &regv1.APIRepositoryList{}

	for _, repo := range repos.Repositories {
		tags := c.ListTags(repo)
		repoList.AddRepository(*tags)
	}

	if err := sync.Registry(c.kClient, c.Name, c.Namespace, c.scheme, repoList); err != nil {
		Logger.Error(err, "failed to synchronize external registry")
		return err
	}

	return nil
}
