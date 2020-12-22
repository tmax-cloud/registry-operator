package utils

import "strings"

const (
	SCHEME_HTTP_PREFIX  = "http://"
	SCHEME_HTTPS_PREFIX = "https://"
)

func AddQueryParams(url string, params map[string][]string) string {
	isFirst := false
	if !strings.Contains(url, "?") {
		url += "?"
		isFirst = true
	}
	for key, values := range params {
		for _, v := range values {
			if !isFirst {
				url += "&"
			}
			url += strings.Join([]string{key, v}, "=")
			isFirst = false
		}
	}

	return url
}
