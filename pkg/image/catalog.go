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
		return &regv1.APIRepositories{}
	}

	Logger.Info("call", "api", u.String())
	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		Logger.Error(err, "")
		return &regv1.APIRepositories{}
	}

	token, err := r.GetToken(catalogScope())
	if err != nil {
		Logger.Error(err, "")
		return &regv1.APIRepositories{}
	}

	req.Header.Set("Authorization", fmt.Sprintf("%s %s", token.Type, token.Value))

	res, err := r.HttpClient.Do(req)
	if err != nil {
		Logger.Error(err, "")
		return &regv1.APIRepositories{}
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		Logger.Error(err, "")
		return &regv1.APIRepositories{}
	}
	// Logger.Info("contents", "repositories", string(body))

	rawRepos := &regv1.APIRepositories{}
	repos := &regv1.APIRepositories{}

	if err := json.Unmarshal(body, rawRepos); err != nil {
		Logger.Error(err, "failed to unmarshal registry's repository")
		return &regv1.APIRepositories{}
	}

	for _, repo := range rawRepos.Repositories {
		if err := r.SetImage(repo); err != nil {
			Logger.Error(err, "failed to set image")
			return &regv1.APIRepositories{}
		}

		tags := r.Tags().Tags
		if len(tags) > 0 {
			repos.Repositories = append(repos.Repositories, repo)
		}
	}

	return repos
}

func (r *Image) Tags() *regv1.APIRepository {
	repo := &regv1.APIRepository{Name: r.Name}

	u, err := tagsURL(r.ServerURL, r.Name)
	if err != nil {
		return &regv1.APIRepository{Name: r.Name}
	}

	Logger.Info("call", "api", u.String())
	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		Logger.Error(err, "")
		return &regv1.APIRepository{Name: r.Name}
	}

	token, err := r.GetToken(repositoryScope(r.Name))
	if err != nil {
		Logger.Error(err, "")
		return &regv1.APIRepository{Name: r.Name}
	}
	req.Header.Set("Authorization", fmt.Sprintf("%s %s", token.Type, token.Value))

	res, err := r.HttpClient.Do(req)
	if err != nil {
		Logger.Error(err, "")
		return &regv1.APIRepository{Name: r.Name}
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		Logger.Error(err, "")
		return &regv1.APIRepository{Name: r.Name}
	}
	Logger.Info("contents", "tags", string(body))

	if err := json.Unmarshal(body, repo); err != nil {
		Logger.Error(err, "")
		return &regv1.APIRepository{Name: r.Name}
	}

	Logger.Info(fmt.Sprintf("APIRepository: %+v", repo))

	return repo
}
