package image

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"strconv"

	"github.com/docker/distribution/manifest/schema1"
	"github.com/docker/distribution/manifest/schema2"
	"github.com/docker/distribution/registry/client"
)

type ImageManifest struct {
	Digest        string
	ContentLength int64

	// *schema1.Manifest or *schema2.Manifest
	Schema interface{}
}

func (r *Image) manifest(schemaVersion int) (*ImageManifest, error) {
	u, err := manifestURL(r.ServerURL, r.Name, r.Tag)
	if err != nil {
		return nil, err
	}

	Logger.Info("call", "api", u.String())
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
	bodyStr, err := ioutil.ReadAll(res.Body)
	if err != nil {
		Logger.Error(err, "")
		return nil, err
	}
	if !client.SuccessStatus(res.StatusCode) {
		Logger.Error(err, "")
		err := client.HandleErrorResponse(res)
		return nil, err
	}

	digest := res.Header.Get("Docker-Content-Digest")
	lengthStr := res.Header.Get("Content-Length")

	if digest == "" || lengthStr == "" {
		return nil, fmt.Errorf("expected headers not exist")
	}

	length, err := strconv.Atoi(lengthStr)
	if err != nil {
		Logger.Error(err, "")
		return nil, err
	}

	if schemaVersion == 2 {
		body := &schema2.Manifest{}
		if err := json.Unmarshal(bodyStr, body); err != nil {
			Logger.Error(err, "")
			return nil, err
		}

		if body.Versioned.SchemaVersion != 2 {
			return nil, errors.New("expected schema version is v2 but v1")
		}

		return &ImageManifest{
			Digest:        digest,
			ContentLength: int64(length),
			Schema:        body,
		}, nil
	}

	body := &schema1.Manifest{}
	if err := json.Unmarshal(bodyStr, body); err != nil {
		Logger.Error(err, "")
		return nil, err
	}

	if body.Versioned.SchemaVersion != 1 {
		return nil, errors.New("expected schema version is v1 but v2")
	}

	return &ImageManifest{
		Digest:        digest,
		ContentLength: int64(length),
		Schema:        body,
	}, nil
}

func (r *Image) GetImageManifest() (*ImageManifest, error) {
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

func (r *Image) ManifestVersion(manifest *ImageManifest) int {
	switch manifest.Schema.(type) {
	case *schema2.Manifest:
		return 2
	case *schema1.Manifest:
		return 1
	}

	return -1
}

func (r *Image) DeleteManifest(manifest *ImageManifest) error {
	u, err := url.Parse(r.ServerURL)
	if err != nil {
		Logger.Error(err, "")
		return err
	}

	u.Path = path.Join(u.Path, fmt.Sprintf("v2/%s/manifests/%s", r.Name, manifest.Digest))
	Logger.Info("call", "api", u.String())
	req, err := http.NewRequest(http.MethodDelete, u.String(), nil)
	if err != nil {
		Logger.Error(err, "")
		return err
	}
	if r.ManifestVersion(manifest) == 2 {
		req.Header.Add("Accept", schema2.MediaTypeManifest)
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

	if res.StatusCode >= 400 {
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
