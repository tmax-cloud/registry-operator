package dockerhub

import "time"

type Authorizer struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type NamespacesResponse struct {
	Namespaces []string `json:"namespaces"`
}

type Repository struct {
	User              string    `json:"user"`
	Name              string    `json:"name"`
	Namespace         string    `json:"namespace"`
	RepositoryType    string    `json:"repository_type"`
	Status            int       `json:"status"`
	Description       string    `json:"description"`
	IsPrivate         bool      `json:"is_private"`
	IsAutomated       bool      `json:"is_automated"`
	CanEdit           bool      `json:"can_edit"`
	StarCount         int       `json:"star_count"`
	PullCount         int       `json:"pull_count"`
	LastUpdated       time.Time `json:"last_updated"`
	IsMigrated        bool      `json:"is_migrated"`
	CollaboratorCount int       `json:"collaborator_count"`
	Affiliation       string    `json:"affiliation"`
	HubUser           string    `json:"hub_user"`
}

type RepositoriesResponse struct {
	Count        int          `json:"count"`
	Next         string       `json:"next"`
	Previous     string       `json:"previous"`
	Repositories []Repository `json:"results"`
}

type Tag struct {
	Name string `json:"name"`
}

type TagsResponse struct {
	Count    int    `json:"count"`
	Next     string `json:"next"`
	Previous string `json:"previous"`
	Tags     []Tag  `json:"results"`
}
