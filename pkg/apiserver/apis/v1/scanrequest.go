package v1

import (
	"encoding/json"
	"fmt"
	"github.com/tmax-cloud/registry-operator/pkg/apiserver/models"
	"k8s.io/apimachinery/pkg/util/rand"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	v1 "github.com/tmax-cloud/registry-operator/api/v1"
	"github.com/tmax-cloud/registry-operator/internal/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

const (
	ScanRequestNameParamKey         = "scanReqName"
	ExternalScanRequestNameParamKey = "ext-scanReqName"
)

func (h *RegistryAPI) CreateImageScanRequest(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	namespace, namespaceOk := vars[NamespaceParamKey]
	name, nameOk := vars[ScanRequestNameParamKey]
	if !namespaceOk || !nameOk {
		_ = utils.RespondError(w, http.StatusBadRequest, "url is malformed")
		return
	}

	ctx := r.Context()
	log := h.logger.WithName("ScanAPI").WithValues("name", name, "namespace", namespace)

	// Decode request body
	reqBody := &models.ScanApiRequest{}
	if err := json.NewDecoder(r.Body).Decode(reqBody); err != nil {
		log.Error(err, "failed to decode request body")
		_ = utils.RespondError(w, http.StatusInternalServerError, "failed to decode request body")
		return
	}

	var targets []v1.ScanTarget
	for _, reg := range reqBody.Registries {
		o := &v1.Registry{}
		if err := h.c.Get(ctx, types.NamespacedName{Name: reg.Name, Namespace: namespace}, o); err != nil {
			log.Error(err, "failed to get registry")
			_ = utils.RespondError(w, http.StatusNotFound, "registry not found")
			return
		}

		var images []string
		for _, repository := range reg.Repositories {
			if repository.Name == "*" {
				images = []string{"*"}
				break
			}
			repo := &v1.Repository{}
			if err := h.c.Get(ctx, types.NamespacedName{Name: repository.Name, Namespace: namespace}, repo); err != nil {
				log.Error(err, "failed to get repository", "repository", repository.Name)
				_ = utils.RespondError(w, http.StatusNotFound, "repository not found")
				return
			}

			var taggedImageNames []string
			for _, tag := range repository.Versions {
				// FIXME: Change if ImageScanRequest Controller support for tag pattern
				if tag == "*" {
					taggedImageNames = append(taggedImageNames, repo.Spec.Name)
					break
				}
				taggedImageNames = append(taggedImageNames, fmt.Sprintf("%s:%s", repo.Spec.Name, tag))
			}
			images = append(images, taggedImageNames...)
		}
		targets = append(targets, v1.ScanTarget{
			RegistryURL:     strings.TrimPrefix(o.Status.ServerURL, "https://"),
			Images:          images,
			ImagePullSecret: v1.K8sPrefix + v1.K8sRegistryPrefix + strings.ToLower(reg.Name),
		})
	}

	imageScanRequest := &v1.ImageScanRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name + "-" + rand.String(10),
			Namespace: namespace,
		},
		Spec: v1.ImageScanRequestSpec{
			ScanTargets: targets,
			SendReport:  true,
			Insecure:    true,
		},
	}
	if err := h.c.Create(ctx, imageScanRequest); err != nil {
		log.Error(err, "failed to create resource ImageScanRequest")
		_ = utils.RespondError(w, http.StatusInternalServerError, "failed to create resource ImageScanRequest")
		return
	}
	w.WriteHeader(http.StatusCreated)
	_ = utils.RespondJSON(w, &models.ScanApiResponse{Name: imageScanRequest.Name})
}

func (h *RegistryAPI) CreateImageScanRequestFromExternalReg(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	namespace, namespaceOk := vars[NamespaceParamKey]
	name, nameOk := vars[ExternalScanRequestNameParamKey]
	if !namespaceOk || !nameOk {
		_ = utils.RespondError(w, http.StatusBadRequest, "url is malformed")
		return
	}

	ctx := r.Context()
	log := h.logger.WithName("ExternalScanAPI").WithValues("name", name, "namespace", namespace)

	reqBody := &models.ScanApiRequest{}
	if err := json.NewDecoder(r.Body).Decode(reqBody); err != nil {
		log.Error(err, "failed to decode request body")
		_ = utils.RespondError(w, http.StatusBadRequest, "failed to decode request body")
		return
	}

	var targets []v1.ScanTarget
	for _, reg := range reqBody.Registries {
		o := &v1.ExternalRegistry{}
		if err := h.c.Get(ctx, types.NamespacedName{Name: reg.Name, Namespace: namespace}, o); err != nil {
			log.Error(err, "failed to get external registry")
			_ = utils.RespondError(w, http.StatusNotFound, "external registry not found")
			return
		}

		var images []string
		for _, repository := range reg.Repositories {
			if repository.Name == "*" {
				images = []string{"*"}
				break
			}

			repo := &v1.Repository{}
			if err := h.c.Get(ctx, types.NamespacedName{Name: repository.Name, Namespace: namespace}, repo); err != nil {
				log.Error(err, "failed to get repository", "repository", repository.Name)
				_ = utils.RespondError(w, http.StatusNotFound, "repository not found")
				return
			}

			var taggedImageNames []string
			for _, tag := range repository.Versions {
				if tag == "*" {
					images = append(images, repo.Spec.Name)
					break
				}
				taggedImageNames = append(taggedImageNames, fmt.Sprintf("%s:%s", repo.Spec.Name, tag))
			}
			images = append(images, taggedImageNames...)
		}

		targets = append(targets, v1.ScanTarget{
			RegistryURL:       strings.TrimPrefix(o.Spec.RegistryURL, "https://"),
			Images:            images,
			ImagePullSecret:   o.Status.LoginSecret,
			CertificateSecret: o.Spec.CertificateSecret,
		})
	}

	imageScanRequest := &v1.ImageScanRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name + "-" + rand.String(10),
			Namespace: namespace,
		},
		Spec: v1.ImageScanRequestSpec{
			ScanTargets: targets,
			SendReport:  true,
			Insecure:    true,
		},
	}

	if err := h.c.Create(ctx, imageScanRequest); err != nil {
		log.Error(err, "failed to create resource ImageScanRequest")
		_ = utils.RespondError(w, http.StatusInternalServerError, "failed to create resource ImageScanRequest")
		return
	}

	w.WriteHeader(http.StatusCreated)
	_ = utils.RespondJSON(w, &models.ScanApiResponse{Name: imageScanRequest.Name})
}
