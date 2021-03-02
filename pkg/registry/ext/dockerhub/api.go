package dockerhub

import (
	"fmt"
)

const (
	dockerHubURL = "https://hub.docker.com"
)

func loginURL() string {
	return fmt.Sprintf("%s/v2/users/login", dockerHubURL)
}

func listNamespacesURL() string {
	return fmt.Sprintf("%s/v2/repositories/namespaces", dockerHubURL)
}

func listRepositoriesURL(namespace string, page, pageSize int) string {
	return fmt.Sprintf("%s/v2/repositories/%s?page=%d&page_size=%d", dockerHubURL, namespace, page, pageSize)
}

func listTagsURL(namespace, repository string, page, pageSize int) string {
	return fmt.Sprintf("%s/v2/repositories/%s/%s/tags?page=%d&page_size=%d", dockerHubURL, namespace, repository, page, pageSize)
}
