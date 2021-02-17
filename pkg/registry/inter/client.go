package inter

import (
	"net/http"

	"github.com/docker/docker/client"
	regv1 "github.com/tmax-cloud/registry-operator/api/v1"
	cmhttp "github.com/tmax-cloud/registry-operator/internal/common/http"
	"github.com/tmax-cloud/registry-operator/internal/utils"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var Logger = log.Log.WithName("inter-registry")

// NewClient is api client of internal registry
func NewClient(c client.Client, registry types.NamespacedName, scheme *runtime.Scheme, httpClient *cmhttp.HttpClient) *Client {
	return &Client{
		Name:       registry.Name,
		Namespace:  registry.Namespace,
		HttpClient: httpClient,
		scheme:     scheme,
	}
}

type Client struct {
	Name, Namespace string

	kClient client.Client
	*cmhttp.HttpClient
	scheme *runtime.Scheme
}

// SetAuth sets Authorization header
func (c *Client) SetAuth(req *http.Request) {
	req.Header.Add("Authorization", "Basic "+utils.HTTPEncodeBasicAuth(c.Login.Username, c.Login.Password))
}

// ListRepositories get repository list from registry server
func (c *Client) ListRepositories() *regv1.APIRepositories {
	return nil
}

// ListTags get tag list of repository from registry server
func (c *Client) ListTags(repository string) *regv1.APIRepository {
	return nil

}

// Synchronize synchronizes repository list between tmax.io.Repository resource and Registry server
func (c *Client) Synchronize() error {
	return nil
}
