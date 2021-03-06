package handler

import (
	"context"
	"errors"
	"fmt"
	"strings"

	v1 "github.com/tmax-cloud/registry-operator/api/v1"
	"github.com/tmax-cloud/registry-operator/internal/schemes"
	"github.com/tmax-cloud/registry-operator/pkg/registry"
	"github.com/tmax-cloud/registry-operator/pkg/registry/base"
	"github.com/tmax-cloud/registry-operator/pkg/registry/replicate"
	"github.com/tmax-cloud/registry-operator/pkg/scheduler"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var logger = log.Log.WithName("replicate-handler")

func RegisterHandler(mgr ctrl.Manager, s *scheduler.Scheduler) error {
	h := NewReplicateHandler(mgr.GetClient(), mgr.GetScheme())
	if err := s.RegisterHandler(v1.JobTypeImageReplicate, h); err != nil {
		logger.Error(err, "unable to register handler", "type", v1.JobTypeImageReplicate)
		return err
	}
	return nil
}

// NewReplicateHandler returns a new handler to replicate image
func NewReplicateHandler(k8sClient client.Client, scheme *runtime.Scheme) *ReplicateHandler {
	return &ReplicateHandler{
		k8sClient: k8sClient,
		scheme:    scheme,
	}
}

// ReplicateHandler contains objects to use in handle function
type ReplicateHandler struct {
	k8sClient client.Client
	scheme    *runtime.Scheme
}

// Handle synchronizes external registry repository list
func (h *ReplicateHandler) Handle(object types.NamespacedName) error {
	// get image replicate
	logger.Info("get image replicate")
	object.Name = strings.TrimPrefix(object.Name, v1.K8sPrefix+schemes.ImageReplicatePrefix)
	replImage := &v1.ImageReplicate{}
	if err := h.k8sClient.Get(context.TODO(), object, replImage); err != nil {
		logger.Error(err, "")
		return err
	}

	from := replImage.Spec.FromImage
	to := replImage.Spec.ToImage
	fromReplicate, fromURL, err := h.GetReplicate(&from)
	if err != nil {
		logger.Error(err, "failed to get replicate client")
		return err
	}
	toReplicate, toURL, err := h.GetReplicate(&to)
	if err != nil {
		logger.Error(err, "failed to get replicate client")
		return err
	}

	fromURL = strings.TrimPrefix(fromURL, "http://")
	fromURL = strings.TrimPrefix(fromURL, "https://")
	toURL = strings.TrimPrefix(toURL, "http://")
	toURL = strings.TrimPrefix(toURL, "https://")
	if err := replicate.Copy(fromReplicate, toReplicate, fmt.Sprintf("%s/%s", fromURL, from.Image), fmt.Sprintf("%s/%s", toURL, to.Image)); err != nil {
		logger.Error(err, "failed to get copy image")
		return err
	}

	return nil
}

// GetReplicate returns replicable registry client
func (h *ReplicateHandler) GetReplicate(image *v1.ImageInfo) (base.Replicatable, string, error) {
	regNames := types.NamespacedName{Name: image.RegistryName, Namespace: image.RegistryNamespace}
	httpClient, err := registry.GetHTTPClient(h.k8sClient, image)
	if err != nil {
		logger.Error(err, "failed to get http client", "image", fmt.Sprintf("%+v", image))
		return nil, "", err
	}
	baseFactory := &base.Factory{
		K8sClient:      h.k8sClient,
		NamespacedName: regNames,
		Scheme:         h.scheme,
		HttpClient:     httpClient,
	}
	factory := registry.GetFactory(image.RegistryType, baseFactory)
	replicate, ok := factory.Create(image.RegistryType).(base.Replicatable)
	if !ok {
		return nil, "", errors.New("failed to create replicatable registry client")
	}

	return replicate, httpClient.URL, nil
}
