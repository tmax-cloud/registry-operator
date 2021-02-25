/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/docker/distribution/manifest/manifestlist"
	"github.com/docker/distribution/manifest/schema1"
	"github.com/docker/distribution/manifest/schema2"
	"github.com/docker/distribution/reference"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	contv1 "github.com/opencontainers/image-spec/specs-go/v1"
	tmaxiov1 "github.com/tmax-cloud/registry-operator/api/v1"
	"github.com/tmax-cloud/registry-operator/pkg/registry"
	"github.com/tmax-cloud/registry-operator/pkg/registry/base"
)

// ImageReplicateReconciler reconciles a ImageReplicate object
type ImageReplicateReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=tmax.io,resources=imagereplicates,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=tmax.io,resources=imagereplicates/status,verbs=get;update;patch

func (r *ImageReplicateReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()
	_ = r.Log.WithValues("imagereplicate", req.NamespacedName)

	// get image replicate
	r.Log.Info("get image replicate")
	replImage := &tmaxiov1.ImageReplicate{}
	if err := r.Get(context.TODO(), req.NamespacedName, replImage); err != nil {
		r.Log.Error(err, "")
		return ctrl.Result{}, nil
	}

	// TODO: Below copy logic will be moved. it is here for testing temporarilly.
	from := replImage.Spec.FromImage
	to := replImage.Spec.ToImage
	fromReplicate, fromURL, err := r.GetReplicate(&from)
	if err != nil {
		r.Log.Error(err, "failed to get replicate client")
		return ctrl.Result{}, err
	}
	toReplicate, toURL, err := r.GetReplicate(&to)
	if err != nil {
		r.Log.Error(err, "failed to get replicate client")
		return ctrl.Result{}, err
	}

	fromURL = strings.TrimPrefix(fromURL, "http://")
	fromURL = strings.TrimPrefix(fromURL, "https://")
	toURL = strings.TrimPrefix(toURL, "http://")
	toURL = strings.TrimPrefix(toURL, "https://")
	if err := r.Copy(fromReplicate, toReplicate, fmt.Sprintf("%s/%s", fromURL, from.Image), fmt.Sprintf("%s/%s", toURL, to.Image)); err != nil {
		r.Log.Error(err, "failed to get copy image")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// TODO: Below copy logic will be moved. it is here for testing temporarilly.
func (r *ImageReplicateReconciler) GetReplicate(image *tmaxiov1.ImageInfo) (base.Replicatable, string, error) {
	regNames := types.NamespacedName{Name: image.RegistryName, Namespace: image.RegistryNamespace}
	url, err := registry.GetURL(r.Client, regNames, image.RegistryType)
	if err != nil {
		r.Log.Error(err, "failed to get url")
		return nil, "", err
	}

	httpClient := registry.GetHTTPClient(url, image.RegistryNamespace, image.ImagePullSecret, image.CertificateSecret)
	baseFactory := &base.Factory{
		K8sClient:      r.Client,
		NamespacedName: regNames,
		Scheme:         r.Scheme,
		HttpClient:     httpClient,
	}
	factory := registry.GetFactory(image.RegistryType, baseFactory)
	replicate, ok := factory.Create(image.RegistryType).(base.Replicatable)
	if !ok {
		return nil, "", errors.New("failed to create replicatable registry client")
	}

	return replicate, url, nil
}

func (r *ImageReplicateReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&tmaxiov1.ImageReplicate{}).
		Complete(r)
}

// TODO: Below copy logic will be moved. it is here for testing temporarilly.
func (r *ImageReplicateReconciler) Copy(fromReplicate, toReplicate base.Replicatable, fromImage, toImage string) error {
	fromNamed, err := reference.ParseNamed(fromImage)
	if err != nil {
		r.Log.Error(err, "failed to parse image", "image", fromImage)
		return err
	}
	toNamed, err := reference.ParseNamed(toImage)
	if err != nil {
		r.Log.Error(err, "failed to parse image", "image", toImage)
		return err
	}

	r.Log.Info("debug", "fromNamed.RemoteName()", fromImage, "toNamed.RemoteName()", toImage)

	manifest, err := fromReplicate.GetManifest(fromImage)
	if err != nil {
		r.Log.Error(err, "failed to get manifest", "fromImage", fromImage)
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
				r.Log.Error(err, "failed to parse digest", "digest", digest)
				return err
			}
			toDigest, err := reference.WithDigest(toNamed, digest)
			if err != nil {
				r.Log.Error(err, "failed to parse digest", "digest", digest)
				return err
			}

			r.Log.Info("debug", "fromDigest", fromDigest.String(), "fromDigest.name", fromDigest.Name())
			if err = r.Copy(fromReplicate, toReplicate, fromDigest.String(), toDigest.String()); err != nil {
				r.Log.Error(err, "failed to copy", "from", fromDigest.String(), "to", toDigest.String())
				return err
			}
		case contv1.MediaTypeImageConfig, contv1.MediaTypeImageLayer, contv1.MediaTypeImageLayerGzip,
			schema1.MediaTypeManifestLayer, schema2.MediaTypeLayer, schema2.MediaTypeUncompressedLayer,
			schema2.MediaTypeImageConfig:
			exist, err := fromReplicate.ExistBlob(fromNamed.Name(), digest.String())
			if err != nil {
				r.Log.Error(err, "failed to check blob exists")
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
				r.Log.Info("remove temporalilly stored blob", "file", filePath)
				if err := os.Remove(filePath); err != nil {
					r.Log.Error(err, "failed to remove file", "file", filePath)
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
				r.Log.Error(err, "failed to check blob exists")
				return err
			}

			if exist {
				return fmt.Errorf("%s blob already exist", toNamed.Name()+"@"+digest.String())
			}

			if err := toReplicate.PushBlob(toNamed.Name(), digest.String(), contentMap[digest.String()].file, contentMap[digest.String()].size); err != nil {
				r.Log.Error(err, "failed to push blob", "blob", toNamed.Name()+"@"+digest.String())
				return err
			}
		}
	}

	// push manifest
	if err := toReplicate.PutManifest(toImage, manifest); err != nil {
		r.Log.Error(err, "failed to upload manifest", "image", toImage)
		return err
	}

	return nil
}
