package image

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/docker/distribution/registry/client"
	"github.com/tmax-cloud/registry-operator/internal/utils"
)

func (r *Image) PullBlob() (io.ReadCloser, int64, error) {
	u, err := blobURL(r.ServerURL, r.Name, r.Digest)
	if err != nil {
		return nil, 0, err
	}

	Logger.Info("call", "method", http.MethodGet, "api", u.String())
	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		Logger.Error(err, "")
		return nil, 0, err
	}

	token, err := r.GetToken(repositoryScope(r.Name))
	if err != nil {
		Logger.Error(err, "")
		return nil, 0, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("%s %s", token.Type, token.Value))
	req.Header.Set("Accept-Encoding", "identity")

	res, err := r.HttpClient.Do(req)
	if err != nil {
		Logger.Error(err, "")
		return nil, 0, err
	}

	if !client.SuccessStatus(res.StatusCode) {
		defer res.Body.Close()
		Logger.Error(err, "failed to pull blob")
		err := client.HandleErrorResponse(res)
		return nil, 0, err
	}

	l := res.Header.Get("Content-Length")
	if l == "" {
		return res.Body, 0, nil
	}

	size, err := strconv.ParseInt(l, 10, 64)
	if err != nil {
		defer res.Body.Close()
		return nil, 0, err
	}

	return res.Body, size, nil
}

// ExistBlob checks if blob exists. If exist, return true
func (r *Image) ExistBlob() (bool, error) {
	u, err := blobURL(r.ServerURL, r.Name, r.Digest)
	if err != nil {
		return false, err
	}

	Logger.Info("call", "method", http.MethodHead, "api", u.String())
	req, err := http.NewRequest(http.MethodHead, u.String(), nil)
	if err != nil {
		Logger.Error(err, "")
		return false, err
	}

	token, err := r.GetToken(repositoryScope(r.Name))
	if err != nil {
		Logger.Error(err, "")
		return false, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("%s %s", token.Type, token.Value))

	res, err := r.HttpClient.Do(req)
	if err != nil {
		Logger.Error(err, "")
		return false, err
	}
	defer res.Body.Close()

	if !client.SuccessStatus(res.StatusCode) {
		if res.StatusCode == 404 {
			return false, nil
		}

		Logger.Error(err, "failed to check if blob exists")
		err := client.HandleErrorResponse(res)
		return false, err
	}

	return true, nil
}

// PushBlob pushes blob
// returns location, uuid, error
func (r *Image) PushBlob(blob []byte, size int64) (string, string, error) {
	location, uuid, err := r.initUpdateBlob()
	if err != nil {
		Logger.Error(err, "")
		return "", "", err
	}
	Logger.Info("debug", "location", location, "uuid", uuid)

	u, err := completelyUploadBlobURL(location, r.Digest)
	if err != nil {
		Logger.Error(err, "")
		return "", "", err
	}

	Logger.Info("call", "method", http.MethodPut, "api", u.String())
	req, err := http.NewRequest(http.MethodPut, u.String(), bytes.NewReader(blob))
	if err != nil {
		Logger.Error(err, "")
		return "", "", err
	}

	token, err := r.GetToken(repositoryScope(r.Name))
	if err != nil {
		Logger.Error(err, "")
		return "", "", err
	}
	req.ContentLength = size
	req.Header.Set("Authorization", fmt.Sprintf("%s %s", token.Type, token.Value))
	req.Header.Set("Content-Type", utils.ContentTypeBinary)

	res, err := r.HttpClient.Do(req)
	if err != nil {
		Logger.Error(err, "")
		return "", "", err
	}

	if !client.SuccessStatus(res.StatusCode) {
		defer res.Body.Close()
		err := client.HandleErrorResponse(res)
		Logger.Error(err, "failed to push blob")
		return "", "", err
	}

	return res.Header.Get("Location"), res.Header.Get("Docker-Upload-UUID"), nil
}

func (r *Image) initUpdateBlob() (string, string, error) {
	u, err := uploadBlobURL(r.ServerURL, r.Name)
	if err != nil {
		Logger.Error(err, "")
		return "", "", err
	}

	Logger.Info("call", "method", http.MethodPost, "api", u.String()+"/")
	req, err := http.NewRequest(http.MethodPost, u.String()+"/", nil)
	if err != nil {
		Logger.Error(err, "")
		return "", "", err
	}

	token, err := r.GetToken(repositoryScope(r.Name))
	if err != nil {
		Logger.Error(err, "")
		return "", "", err
	}

	req.Header.Set("Content-Type", utils.ContentTypeBinary)
	req.Header.Set("Authorization", fmt.Sprintf("%s %s", token.Type, token.Value))
	req.Header.Set("Content-Length", "0")

	res, err := r.HttpClient.Do(req)
	if err != nil {
		Logger.Error(err, "")
		return "", "", err
	}

	if !client.SuccessStatus(res.StatusCode) {
		defer res.Body.Close()
		err := client.HandleErrorResponse(res)
		Logger.Error(err, "failed to init push blob")
		return "", "", err
	}

	return res.Header.Get("Location"), res.Header.Get("Docker-Upload-UUID"), nil
}
