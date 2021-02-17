package registry

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	regv1 "github.com/tmax-cloud/registry-operator/api/v1"
	"github.com/tmax-cloud/registry-operator/controllers/keycloakctl"
	cmhttp "github.com/tmax-cloud/registry-operator/internal/common/http"

	"sigs.k8s.io/controller-runtime/pkg/log"
)

type RegistryApi struct {
	*cmhttp.HttpClient
	kcCli *keycloakctl.KeycloakClient
}

var logger = log.Log.WithName("registry-api")

func NewRegistryApi(reg *regv1.Registry) *RegistryApi {
	ra := &RegistryApi{}
	regURL := registryUrl(reg)
	if regURL == "" {
		return nil
	}

	ra.HttpClient = cmhttp.NewHTTPClient(regURL, reg.Spec.LoginID, reg.Spec.LoginPassword, nil, false)
	kcCtl := keycloakctl.NewKeycloakController(reg.Namespace, reg.Name)
	ra.kcCli = keycloakctl.NewKeycloakClient(reg.Spec.LoginID, reg.Spec.LoginPassword, kcCtl.GetRealmName(), kcCtl.GetDockerV2ClientName())

	logger.Info("New Keycloak Client Success")
	return ra
}

func registryUrl(reg *regv1.Registry) string {
	if len(reg.Status.ServerURL) == 0 {
		return ""
	}
	return reg.Status.ServerURL
}

func (r *RegistryApi) Catalog() *regv1.APIRepositories {
	logger.Info("call", "api", r.URL+"/v2/_catalog")
	req, err := http.NewRequest(http.MethodGet, r.URL+"/v2/_catalog", nil)
	if err != nil {
		logger.Error(err, "")
		return nil
	}

	scopes := []string{"registry:catalog:*"}
	token, err := r.kcCli.GetToken(scopes)
	if err != nil {
		logger.Error(err, "")
		return nil
	}
	req.Header.Add("Authorization", "Bearer "+token)

	res, err := r.Client.Do(req)
	if err != nil {
		logger.Error(err, "")
		return nil
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		logger.Error(err, "")
		return nil
	}
	// logger.Info("contents", "repositories", string(body))

	rawRepos := &regv1.APIRepositories{}
	repos := &regv1.APIRepositories{}

	if err := json.Unmarshal(body, rawRepos); err != nil {
		logger.Error(err, "failed to unmarshal registry's repository")
		return nil
	}

	for _, repo := range rawRepos.Repositories {
		tags := r.Tags(repo).Tags
		if len(tags) > 0 {
			repos.Repositories = append(repos.Repositories, repo)
		}
	}

	return repos
}

func (r *RegistryApi) Tags(imageName string) *regv1.APIRepository {
	repo := &regv1.APIRepository{Name: imageName}

	logger.Info("call", "api", r.URL+"/v2/"+imageName+"/tags/list")
	req, err := http.NewRequest(http.MethodGet, r.URL+"/v2/"+imageName+"/tags/list", nil)
	if err != nil {
		logger.Error(err, "")
		return repo
	}

	scopes := []string{strings.Join([]string{"repository", imageName, "pull"}, ":")}
	token, err := r.kcCli.GetToken(scopes)
	if err != nil {
		logger.Error(err, "")
		return repo
	}
	req.Header.Add("Authorization", "Bearer "+token)

	res, err := r.Client.Do(req)
	if err != nil {
		logger.Error(err, "")
		return repo
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		logger.Error(err, "")
		return repo
	}
	// logger.Info("contents", "tags", string(body))
	if err := json.Unmarshal(body, repo); err != nil {
		logger.Error(err, "")
		return repo
	}

	return repo
}

func (r *RegistryApi) DockerContentDigest(imageName, tag string) (string, error) {
	logger.Info("call", "api", r.URL+"/v2/"+imageName+"/manifests/"+tag)
	req, err := http.NewRequest(http.MethodGet, r.URL+"/v2/"+imageName+"/manifests/"+tag, nil)
	if err != nil {
		logger.Error(err, "")
		return "", err
	}

	req.Header.Add("Accept", "application/vnd.docker.distribution.manifest.v2+json")
	scopes := []string{strings.Join([]string{"repository", imageName, "pull"}, ":")}
	token, err := r.kcCli.GetToken(scopes)
	if err != nil {
		logger.Error(err, "")
		return "", err
	}
	req.Header.Add("Authorization", "Bearer "+token)

	res, err := r.Client.Do(req)
	if err != nil {
		logger.Error(err, "")
		return "", err
	}

	for key, val := range res.Header {
		if key == "Docker-Content-Digest" {
			return val[0], nil
		}
	}

	if res.StatusCode >= 400 {
		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			logger.Error(err, "")
			return "", err
		}
		logger.Error(nil, "err", "err", string(body))
		return "", fmt.Errorf("error!! %s", string(body))
	}

	return "", nil
}

func (r *RegistryApi) DeleteManifest(imageName, digest string) error {
	logger.Info("call", "api", r.URL+"/v2/"+imageName+"/manifests/"+digest)
	req, err := http.NewRequest(http.MethodDelete, r.URL+"/v2/"+imageName+"/manifests/"+digest, nil)
	if err != nil {
		logger.Error(err, "")
		return err
	}
	req.Header.Add("Accept", "application/vnd.docker.distribution.manifest.v2+json")
	scopes := []string{strings.Join([]string{"repository", imageName, "*"}, ":")}
	token, err := r.kcCli.GetToken(scopes)
	if err != nil {
		logger.Error(err, "")
		return err
	}
	req.Header.Add("Authorization", "Bearer "+token)

	res, err := r.Client.Do(req)
	if err != nil {
		logger.Error(err, "")
		return err
	}

	if res.StatusCode >= 400 {
		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			logger.Error(err, "")
			return nil
		}
		logger.Error(nil, "err", "err", string(body))
		return fmt.Errorf("error!! %s", string(body))
	}

	return nil
}
