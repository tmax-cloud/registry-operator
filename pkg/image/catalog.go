package image

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	regv1 "github.com/tmax-cloud/registry-operator/api/v1"
)

// Catalog gets repository list
func (r *Image) Catalog() *regv1.APIRepositories {
	u, err := catalogURL(r.ServerURL)
	if err != nil {
		return nil
	}

	Logger.Info("call", "api", u.String())
	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		Logger.Error(err, "")
		return nil
	}

	token, err := r.GetToken(catalogScope())
	if err != nil {
		Logger.Error(err, "")
		return nil
	}

	req.Header.Set("Authorization", fmt.Sprintf("%s %s", token.Type, token.Value))

	res, err := r.HttpClient.Do(req)
	if err != nil {
		Logger.Error(err, "")
		return nil
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		Logger.Error(err, "")
		return nil
	}
	// Logger.Info("contents", "repositories", string(body))

	rawRepos := &regv1.APIRepositories{}
	repos := &regv1.APIRepositories{}

	if err := json.Unmarshal(body, rawRepos); err != nil {
		Logger.Error(err, "failed to unmarshal registry's repository")
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

func (r *Image) Tags(imageName string) *regv1.APIRepository {
	u, err := manifestURL(r.ServerURL, r.Name, r.Tag)
	if err != nil {
		return nil
	}

	Logger.Info("call", "api", u.String())
	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		Logger.Error(err, "")
		return nil
	}

	token, err := r.GetToken(repositoryScope(r.Name))
	if err != nil {
		Logger.Error(err, "")
		return nil
	}
	req.Header.Set("Authorization", fmt.Sprintf("%s %s", token.Type, token.Value))

	res, err := r.HttpClient.Do(req)
	if err != nil {
		Logger.Error(err, "")
		return nil
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		Logger.Error(err, "")
		return nil
	}
	// Logger.Info("contents", "tags", string(body))

	repo := &regv1.APIRepository{}
	if err := json.Unmarshal(body, repo); err != nil {
		Logger.Error(err, "")
		return nil
	}

	return repo
}
