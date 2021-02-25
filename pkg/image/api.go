package image

import (
	"fmt"
	"net/url"
	"path"
)

func manifestURL(baseURL, imageName, ref string) (*url.URL, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		Logger.Error(err, "failed to parse url")
		return nil, err
	}
	u.Path = path.Join(u.Path, fmt.Sprintf("v2/%s/manifests/%s", imageName, ref))
	return u, nil
}

func blobURL(baseURL, imageName, digest string) (*url.URL, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		Logger.Error(err, "failed to parse url")
		return nil, err
	}
	u.Path = path.Join(u.Path, fmt.Sprintf("v2/%s/blobs/%s", imageName, digest))
	return u, nil
}

func uploadBlobURL(baseURL, imageName string) (*url.URL, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		Logger.Error(err, "failed to parse url")
		return nil, err
	}
	u.Path = path.Join(u.Path, fmt.Sprintf("v2/%s/blobs/uploads/", imageName))
	return u, nil
}

func completelyUploadBlobURL(location, digest string) (*url.URL, error) {
	u, err := url.Parse(location)
	if err != nil {
		Logger.Error(err, "failed to parse url")
		return nil, err
	}
	q := u.Query()
	q.Set("digest", digest)
	u.RawQuery = q.Encode()
	return u, nil
}

func pingURL(baseURL string) (*url.URL, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		Logger.Error(err, "failed to parse url")
		return nil, err
	}
	u.Path = path.Join(u.Path, "v2")
	return u, nil
}

func catalogURL(baseURL string) (*url.URL, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		Logger.Error(err, "failed to parse url")
		return nil, err
	}
	u.Path = path.Join(u.Path, "v2/_catalog")
	return u, nil
}

func tagsURL(baseURL, imageName string) (*url.URL, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		Logger.Error(err, "failed to parse url")
		return nil, err
	}
	u.Path = path.Join(u.Path, "/v2/"+imageName+"/tags/list")
	return u, nil
}

func repositoryScope(imageName string) string {
	return fmt.Sprintf("repository:%s:pull,push", imageName)
}

func catalogScope() string {
	return "registry:catalog:*"
}
