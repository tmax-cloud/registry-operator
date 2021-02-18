package inter

import (
	regv1 "github.com/tmax-cloud/registry-operator/api/v1"
	"github.com/tmax-cloud/registry-operator/internal/common/certs"
	cmhttp "github.com/tmax-cloud/registry-operator/internal/common/http"
	"github.com/tmax-cloud/registry-operator/internal/schemes"
	"github.com/tmax-cloud/registry-operator/internal/utils"
	"github.com/tmax-cloud/registry-operator/pkg/image"
	"github.com/tmax-cloud/registry-operator/pkg/registry/sync"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var Logger = log.Log.WithName("inter-registry")

type Client struct {
	Name, Namespace string

	kClient     client.Client
	imageClient *image.Image
	scheme      *runtime.Scheme
}

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

// GetClient returns client of internal registry
func GetClient(c client.Client, reg *regv1.Registry, scheme *runtime.Scheme) (*Client, error) {
	imagePullSecret := schemes.SubresourceName(reg, schemes.SubTypeRegistryDCJSecret)
	basic, err := utils.GetBasicAuth(imagePullSecret, reg.Namespace, reg.Status.ServerURL)
	if err != nil {
		Logger.Error(err, "failed to get basic auth")
		return nil, err
	}

	username, password := utils.DecodeBasicAuth(basic)
	caSecret, err := certs.GetRootCert(reg.Namespace)
	if err != nil {
		Logger.Error(err, "failed to get root CA")
		return nil, err
	}
	ca, _ := certs.CAData(caSecret)

	caSecret, err = certs.GetSystemKeycloakCert(c)
	if err == nil {
		kca, _ := certs.CAData(caSecret)
		ca = append(ca, kca...)
	}

	httpClient := cmhttp.NewHTTPClient(
		reg.Status.ServerURL,
		username, password,
		ca,
		len(ca) == 0,
	)

	return NewClient(c, types.NamespacedName{Name: reg.Name, Namespace: reg.Namespace}, scheme, httpClient), nil
}

// ListRepositories get repository list from registry server
func (c *Client) ListRepositories() *regv1.APIRepositories {
	return c.imageClient.Catalog()
}

// ListTags get tag list of repository from registry server
func (c *Client) ListTags(repository string) *regv1.APIRepository {
	if err := c.imageClient.SetImage(repository); err != nil {
		Logger.Error(err, "failed to set image")
		return &regv1.APIRepository{}
	}
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
