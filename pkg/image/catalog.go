package image

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

// Catalog gets repository list
func (r *Image) Catalog() *APIRepositories {
	u, err := catalogURL(r.ServerURL)
	if err != nil {
		return nil
	}

	Logger.Info("call", "method", http.MethodGet, "api", u.String())
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

	rawRepos := &APIRepositories{}
	repos := &APIRepositories{}

	if err := json.Unmarshal(body, rawRepos); err != nil {
		Logger.Error(err, "failed to unmarshal registry's repository")
		return nil
	}

	for _, repo := range rawRepos.Repositories {
		if err := r.SetImage(repo); err != nil {
			Logger.Error(err, "failed to set image")
			return nil
		}

		t := r.Tags()
		if t == nil {
			return nil
		}

		tags := t.Tags
		if len(tags) > 0 {
			repos.Repositories = append(repos.Repositories, repo)
		}
	}

	return repos
}

func (r *Image) Tags() *APIRepository {
	repo := &APIRepository{Name: r.Name}

	u, err := tagsURL(r.ServerURL, r.Name)
	if err != nil {
		return nil
	}

	Logger.Info("call", "method", http.MethodGet, "api", u.String())
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
	Logger.Info("contents", "tags", string(body))

	if err := json.Unmarshal(body, repo); err != nil {
		Logger.Error(err, "")
		return nil
	}

	Logger.Info(fmt.Sprintf("APIRepository: %+v", repo))

	return repo
}
