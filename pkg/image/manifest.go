package image

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/docker/distribution"
	"github.com/docker/distribution/manifest/schema2"
	"github.com/docker/distribution/registry/client"
)

type ImageManifest struct {
	Digest        string
	ContentLength int64
	Manifest      distribution.Manifest
}

func (r *Image) manifest(schemaVersion int) (*ImageManifest, error) {
	ref := r.Tag
	if ref == "" {
		ref = r.Digest
	}
	u, err := manifestURL(r.ServerURL, r.Name, ref)
	if err != nil {
		return nil, err
	}

	Logger.Info("call", "method", http.MethodGet, "api", u.String())
	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		Logger.Error(err, "")
		return nil, err
	}

	token, err := r.GetToken(repositoryScope(r.Name))
	if err != nil {
		Logger.Error(err, "")
		return nil, err
	}

	if schemaVersion == 2 {
		req.Header.Set("Accept", schema2.MediaTypeManifest)
	}
	req.Header.Set("Authorization", fmt.Sprintf("%s %s", token.Type, token.Value))

	res, err := r.HttpClient.Do(req)
	if err != nil {
		Logger.Error(err, "")
		return nil, err
	}
	defer res.Body.Close()
	bodyData, err := ioutil.ReadAll(res.Body)
	if err != nil {
		Logger.Error(err, "")
		return nil, err
	}
	if !client.SuccessStatus(res.StatusCode) {
		err := client.HandleErrorResponse(res)
		Logger.Error(err, "")
		return nil, err
	}

	mediaType := res.Header.Get("Content-Type")
	digest := res.Header.Get("Docker-Content-Digest")
	lengthStr := res.Header.Get("Content-Length")
	manifest, _, err := distribution.UnmarshalManifest(mediaType, bodyData)
	if err != nil {
		return nil, err
	}

	if digest == "" || lengthStr == "" {
		return nil, fmt.Errorf("expected headers not exist")
	}

	length, err := strconv.Atoi(lengthStr)
	if err != nil {
		Logger.Error(err, "")
		return nil, err
	}

	return &ImageManifest{
		Digest:        digest,
		ContentLength: int64(length),
		Manifest:      manifest,
	}, nil
}

// GetManifest gets manifests of image in the registry
func (r *Image) GetManifest() (*ImageManifest, error) {
	mf2, err := r.manifest(2)
	ok := true
	if err != nil {
		Logger.Error(err, "failed to get v2 manifest")
		ok = false
	}

	if ok {
		return mf2, nil
	}

	Logger.Info("unable to get v2 manifest, fall back to v1 manifest")
	mf1, err := r.manifest(1)
	if err != nil {
		Logger.Error(err, "failed to get v1 manifest")
		return nil, err
	}

	return mf1, nil
}

// DeleteManifest deletes manifest in the registry
func (r *Image) DeleteManifest(manifest *ImageManifest) error {
	ref := r.Tag
	if ref == "" {
		ref = r.Digest
	}
	u, err := manifestURL(r.ServerURL, r.Name, ref)
	if err != nil {
		return err
	}

	Logger.Info("call", "method", http.MethodDelete, "api", u.String())
	req, err := http.NewRequest(http.MethodDelete, u.String(), nil)
	if err != nil {
		Logger.Error(err, "")
		return err
	}

	token, err := r.GetToken(repositoryScope(r.Name))
	if err != nil {
		Logger.Error(err, "failed to get token")
		return err
	}
	req.Header.Set("Authorization", fmt.Sprintf("%s %s", token.Type, token.Value))

	res, err := r.HttpClient.Do(req)
	if err != nil {
		Logger.Error(err, "")
		return err
	}
	defer res.Body.Close()

	if !client.SuccessStatus(res.StatusCode) {
		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			Logger.Error(err, "")
			return nil
		}
		Logger.Error(nil, "err", "err", string(body))
		return fmt.Errorf("error!! %s", string(body))
	}

	return nil
}

func (r *Image) PutManifest(manifest *ImageManifest) error {
	mediaType, payload, err := manifest.Manifest.Payload()
	if err != nil {
		Logger.Error(err, "failed to get payload")
		return err
	}
	ref := r.Tag
	if ref == "" {
		ref = r.Digest
	}
	u, err := manifestURL(r.ServerURL, r.Name, ref)
	if err != nil {
		return err
	}

	Logger.Info("call", "method", http.MethodPut, "api", u.String())
	req, err := http.NewRequest(http.MethodPut, u.String(), bytes.NewReader(payload))
	if err != nil {
		Logger.Error(err, "")
		return err
	}

	req.Header.Add("Content-Type", mediaType)

	token, err := r.GetToken(repositoryScope(r.Name))
	if err != nil {
		Logger.Error(err, "failed to get token")
		return err
	}
	req.Header.Set("Authorization", fmt.Sprintf("%s %s", token.Type, token.Value))

	res, err := r.HttpClient.Do(req)
	if err != nil {
		Logger.Error(err, "")
		return err
	}
	defer res.Body.Close()

	if !client.SuccessStatus(res.StatusCode) {
		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			Logger.Error(err, "")
			return nil
		}
		Logger.Error(nil, "err", "err", string(body))
		return fmt.Errorf("error!! %s", string(body))
	}

	return nil
}
