package utils

import (
	"encoding/base64"
	"strings"
)

const (
	SCHEME_HTTP_PREFIX  = "http://"
	SCHEME_HTTPS_PREFIX = "https://"
)

const (
	ContentTypeBinary = "application/octet-stream"
	ContentTypeForm   = "application/x-www-form-urlencoded"
	ContentTypeJSON   = "application/json"
	ContentTypeHTML   = "text/html; charset=utf-8"
	ContentTypeText   = "text/plain; charset=utf-8"
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

// HTTPEncodeBasicAuth encodes basic auth string by base64
func HTTPEncodeBasicAuth(username, password string) string {
	return base64.StdEncoding.EncodeToString([]byte(username + ":" + password))
}

// TrimHTTPScheme trims 'http://' or 'https://' prefix in the url
func TrimHTTPScheme(url string) string {
	url = strings.TrimPrefix(url, SCHEME_HTTP_PREFIX)
	url = strings.TrimPrefix(url, SCHEME_HTTPS_PREFIX)
	return url
}
