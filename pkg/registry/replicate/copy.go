package replicate

import (
	"fmt"
	"os"

	"github.com/docker/distribution/manifest/manifestlist"
	"github.com/docker/distribution/manifest/schema1"
	"github.com/docker/distribution/manifest/schema2"
	"github.com/docker/distribution/reference"
	contv1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/tmax-cloud/registry-operator/pkg/registry/base"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var logger = log.Log.WithName("replicate-copy")

// TODO: Below copy logic will be moved. it is here for testing temporarilly.
func Copy(fromReplicate, toReplicate base.Replicatable, fromImage, toImage string) error {
	fromNamed, err := reference.ParseNamed(fromImage)
	if err != nil {
		logger.Error(err, "failed to parse image", "image", fromImage)
		return err
	}
	toNamed, err := reference.ParseNamed(toImage)
	if err != nil {
		logger.Error(err, "failed to parse image", "image", toImage)
		return err
	}

	logger.Info("debug", "fromNamed.RemoteName()", fromImage, "toNamed.RemoteName()", toImage)

	manifest, err := fromReplicate.GetManifest(fromImage)
	if err != nil {
		logger.Error(err, "failed to get manifest", "fromImage", fromImage)
		return err
	}

	type content struct {
		file string
		size int64
	}

	contentMap := map[string]content{}
	for _, descriptor := range manifest.Manifest.References() {
		digest := descriptor.Digest

		switch descriptor.MediaType {
		case contv1.MediaTypeImageIndex, manifestlist.MediaTypeManifestList, contv1.MediaTypeImageManifest, schema2.MediaTypeManifest,
			schema1.MediaTypeSignedManifest, schema1.MediaTypeManifest:
			fromDigest, err := reference.WithDigest(fromNamed, digest)
			if err != nil {
				logger.Error(err, "failed to parse digest", "digest", digest)
				return err
			}
			toDigest, err := reference.WithDigest(toNamed, digest)
			if err != nil {
				logger.Error(err, "failed to parse digest", "digest", digest)
				return err
			}

			logger.Info("debug", "fromDigest", fromDigest.String(), "fromDigest.name", fromDigest.Name())
			if err = Copy(fromReplicate, toReplicate, fromDigest.String(), toDigest.String()); err != nil {
				logger.Error(err, "failed to copy", "from", fromDigest.String(), "to", toDigest.String())
				return err
			}
		case contv1.MediaTypeImageConfig, contv1.MediaTypeImageLayer, contv1.MediaTypeImageLayerGzip,
			schema1.MediaTypeManifestLayer, schema2.MediaTypeLayer, schema2.MediaTypeUncompressedLayer,
			schema2.MediaTypeImageConfig:
			exist, err := fromReplicate.ExistBlob(fromNamed.Name(), digest.String())
			if err != nil {
				logger.Error(err, "failed to check blob exists")
				return err
			}

			if !exist {
				return fmt.Errorf("%s blob not found", fromNamed.Name()+"@"+digest.String())
			}

			filePath, size, err := fromReplicate.PullBlob(fromNamed.Name(), digest.String())
			if err != nil {
				return err
			}

			contentMap[digest.String()] = content{filePath, size}
			// clean file
			defer func() {
				logger.Info("remove temporalilly stored blob", "file", filePath)
				if err := os.Remove(filePath); err != nil {
					logger.Error(err, "failed to remove file", "file", filePath)
				}
			}()
		}
	}

	// push blob
	for _, descriptor := range manifest.Manifest.References() {
		digest := descriptor.Digest

		switch descriptor.MediaType {
		case contv1.MediaTypeImageConfig, contv1.MediaTypeImageLayer, contv1.MediaTypeImageLayerGzip,
			schema1.MediaTypeManifestLayer, schema2.MediaTypeLayer, schema2.MediaTypeUncompressedLayer,
			schema2.MediaTypeImageConfig:
			exist, err := toReplicate.ExistBlob(toNamed.Name(), digest.String())
			if err != nil {
				logger.Error(err, "failed to check blob exists")
				return err
			}

			if exist {
				return fmt.Errorf("%s blob already exist", toNamed.Name()+"@"+digest.String())
			}

			if err := toReplicate.PushBlob(toNamed.Name(), digest.String(), contentMap[digest.String()].file, contentMap[digest.String()].size); err != nil {
				logger.Error(err, "failed to push blob", "blob", toNamed.Name()+"@"+digest.String())
				return err
			}
		}
	}

	// push manifest
	if err := toReplicate.PutManifest(toImage, manifest); err != nil {
		logger.Error(err, "failed to upload manifest", "image", toImage)
		return err
	}

	return nil
}
