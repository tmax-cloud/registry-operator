package v1

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	v1 "github.com/tmax-cloud/registry-operator/api/v1"
	"github.com/tmax-cloud/registry-operator/internal/utils"
	"github.com/tmax-cloud/registry-operator/internal/wrapper"
	"github.com/tmax-cloud/registry-operator/pkg/scan"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"net/http"
	"path"
	"strings"
)

const (
	ScanKind = "scans"
)

func AddScanRequest(parent *wrapper.RouterWrapper) error {
	scanRequestWrapper := wrapper.New(fmt.Sprintf("/%s", ScanKind), []string{http.MethodPost}, scanRequestHandler)
	scanRequestWrapper.Router.Use(authenticate)
	// TODO : Authorize
	if err := parent.Add(scanRequestWrapper); err != nil {
		return err
	}

	return nil
}

func scanRequestHandler(w http.ResponseWriter, req *http.Request) {
	reqId := utils.RandomString(10)
	log := logger.WithValues("request", reqId)

	// Get path parameters
	vars := mux.Vars(req)

	ns, nsExist := vars[NamespaceParamKey]
	if !nsExist {
		_ = utils.RespondError(w, http.StatusBadRequest, "url is malformed")
		return
	}

	// Decode request body
	reqBody := &scan.Request{}
	decoder := json.NewDecoder(req.Body)
	if err := decoder.Decode(reqBody); err != nil {
		log.Info(err.Error())
		_ = utils.RespondError(w, http.StatusBadRequest, fmt.Sprintf("req: %s, body is not in json form or is malformed, err : %s", reqId, err.Error()))
		return
	}

	// Create ImageScanRequest
	scanRequest, err := newImageScanReq(ns, reqBody)
	if err != nil {
		log.Info(err.Error())
		_ = utils.RespondError(w, http.StatusInternalServerError, fmt.Sprintf("req: %s, cannot create ImageScanRequest", reqId))
		return
	}

	if err := k8sClient.Create(context.Background(), scanRequest); err != nil {
		log.Info(err.Error())
		_ = utils.RespondError(w, http.StatusInternalServerError, fmt.Sprintf("req: %s, cannot create ImageScanRequest", reqId))
		return
	}

	w.WriteHeader(http.StatusCreated)
	_ = utils.RespondJSON(w, &scan.RequestResponse{ImageScanRequestName: scanRequest.Name})
}

func newImageScanReq(ns string, reqBody *scan.Request) (*v1.ImageScanRequest, error) {
	if reqBody == nil {
		return nil, fmt.Errorf("reqBody is nil")
	}

	randId := utils.RandomString(5)

	var targets []v1.ScanTarget

	// Parse registry url
	for _, reg := range reqBody.Registries {
		regName := reg.Name
		regObj := &v1.Registry{}
		if err := k8sClient.Get(context.Background(), types.NamespacedName{Name: regName, Namespace: ns}, regObj); err != nil {
			return nil, err
		}

		regCred := v1.K8sPrefix + strings.ToLower(regName)

		var repoUrls []string
		for _, repo := range reg.Repositories {
			repoName := repo.Name
			repoObj := &v1.Repository{}
			if err := k8sClient.Get(context.Background(), types.NamespacedName{Name: repoName, Namespace: ns}, repoObj); err != nil {
				return nil, err
			}

			// Repo wild card
			if repoName == "*" {
				repoUrls = []string{"*"}
				break
			}

			repoBaseUrl := repoObj.Spec.Name

			var tagUrls []string
			for _, tag := range repo.Versions {
				// Tag wild card
				if tag == "*" {
					tagUrls = []string{path.Join(repoBaseUrl, "*")}
					break
				}

				tagUrls = append(tagUrls, path.Join(repoBaseUrl, tag))
			}

			repoUrls = append(repoUrls, tagUrls...)
		}

		targets = append(targets, v1.ScanTarget{
			Images:          repoUrls,
			ImagePullSecret: regCred,
			RegistryUrl: regObj.Status.ServerURL,
			ElasticSearch: true,
			Insecure: true,
		})
	}

	return &v1.ImageScanRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "image-scan-" + randId,
			Namespace: ns,
		},
		Spec: v1.ImageScanRequestSpec{
			ScanTargets: targets,
		},
	}, nil
}
