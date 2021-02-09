package v2

import (
	"fmt"

	"net/url"
)

func listProjectsURL(baseURL string) string {
	return fmt.Sprintf("%s/api/v2.0/projects", baseURL)
}

func listRepositoriessURL(baseURL, project string) string {
	return fmt.Sprintf("%s/api/v2.0/projects/%s/repositories", baseURL, project)
}

func listTagsURL(baseURL, project, repository string) string {
	return fmt.Sprintf("%s/api/v2.0/projects/%s/repositories/%s/artifacts", baseURL, project, url.PathEscape(repository))
}
