package image

type Manifest struct {
	Digest        string
	ContentLength int64

	// *schema1.Manifest or *schema2.Manifest
	Schema interface{}
}

// API
type APIRepositories struct {
	Repositories []string `json:"repositories"`
}

type APIRepository struct {
	Name string   `json:"name"`
	Tags []string `json:"tags"`
}

type APIRepositoryList []APIRepository

func (l APIRepositoryList) GetRepository(name string) *APIRepository {
	for _, repo := range l {
		if repo.Name == name {
			return &repo
		}
	}

	return nil
}

func (l *APIRepositoryList) AddRepository(repo APIRepository) {
	*l = append(*l, repo)
}
