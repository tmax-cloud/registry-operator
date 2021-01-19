package v1

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	v1 "github.com/tmax-cloud/registry-operator/api/v1"
	"github.com/tmax-cloud/registry-operator/internal/schemes"
	"github.com/tmax-cloud/registry-operator/internal/utils"
	"github.com/tmax-cloud/registry-operator/internal/wrapper"
	"github.com/tmax-cloud/registry-operator/pkg/scan"
	"github.com/tmax-cloud/registry-operator/pkg/trust"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"net/http"
	"path"
	"strings"
)

const (
	RepositoryKind = "repositories"
	ScanResultKind = "imagescanresults"

	RepositoryParamKey = "repositoryName"
	TagParamKey        = "tagName"
)

func AddScanResult(parent *wrapper.RouterWrapper) error {
	listScanSummaryWrapper := wrapper.New(fmt.Sprintf("/%s/{%s}/%s", RepositoryKind, RepositoryParamKey, ScanResultKind), []string{http.MethodGet}, listScanSummaryHandler)
	if err := parent.Add(listScanSummaryWrapper); err != nil {
		return err
	}
	listScanSummaryWrapper.Router.Use(authenticate)
	// TODO : Authorize

	scanResultWrapper := wrapper.New(fmt.Sprintf("/%s/{%s}/%s/{%s}", RepositoryKind, RepositoryParamKey, ScanResultKind, TagParamKey), []string{http.MethodGet}, scanResultHandler)
	if err := parent.Add(scanResultWrapper); err != nil {
		return err
	}
	scanResultWrapper.Router.Use(authenticate)
	// TODO : Authorize

	return nil
}

// Return summary of vulnerabilities
func listScanSummaryHandler(w http.ResponseWriter, req *http.Request) {
	results, err := getScanResult(req)
	if err != nil {
		code := http.StatusInternalServerError
		msg := err.Error()
		statErr, ok := err.(*errors.StatusError)
		if ok {
			code = int(statErr.ErrStatus.Code)
			msg = statErr.Error()
		}
		_ = utils.RespondError(w, code, msg)
	}

	summary := map[string]map[string]int{}
	for tag, vuls := range results {
		summary[tag] = map[string]int{}
		for severity, v := range vuls {
			summary[tag][severity] = len(v)
		}
	}

	_ = utils.RespondJSON(w, summary)
}

// Return actual list of vulnerabilities
func scanResultHandler(w http.ResponseWriter, req *http.Request) {
	results, err := getScanResult(req)
	if err != nil {
		code := http.StatusInternalServerError
		msg := err.Error()
		statErr, ok := err.(*errors.StatusError)
		if ok {
			code = int(statErr.ErrStatus.Code)
			msg = statErr.Error()
		}
		_ = utils.RespondError(w, code, msg)
	}
	_ = utils.RespondJSON(w, results)
}

func getScanResult(req *http.Request) (map[string]scan.ResultResponse, error) {
	reqId := utils.RandomString(10)
	log := logger.WithValues("request", reqId)

	// Get path parameters
	vars := mux.Vars(req)

	ns, nsExist := vars[NamespaceParamKey]
	repoName, repoNameExist := vars[RepositoryParamKey]
	if !nsExist || !repoNameExist {
		return nil, errors.NewBadRequest("url is malformed")
	}

	// Get tag
	tag, tagExist := vars[TagParamKey]

	repo := &v1.Repository{}
	if err := k8sClient.Get(context.Background(), types.NamespacedName{Name: repoName, Namespace: ns}, repo); err != nil {
		log.Info(err.Error())
		return nil, errors.NewInternalError(err)
	}

	reg := &v1.Registry{}
	if err := k8sClient.Get(context.Background(), types.NamespacedName{Name: repo.Spec.Registry, Namespace: ns}, reg); err != nil {
		log.Info(err.Error())
		return nil, errors.NewInternalError(err)
	}

	regBaseUrl := strings.TrimPrefix(reg.Status.ServerURL, "https://")

	// TODO - functionize
	secret := &corev1.Secret{}
	if err := k8sClient.Get(context.Background(), types.NamespacedName{Name: v1.K8sPrefix + strings.ToLower(reg.Name), Namespace: ns}, secret); err != nil {
		log.Info(err.Error())
		return nil, errors.NewInternalError(err)
	}

	authStr, ok := secret.Data[schemes.DockerConfigJson]
	if !ok {
		msg := "cannot find .dockerconfigjson from the secret"
		log.Info(msg)
		return nil, errors.NewInternalError(fmt.Errorf(msg))
	}

	basicAuth := &schemes.DockerConfig{}
	if err := json.Unmarshal(authStr, basicAuth); err != nil {
		log.Info(err.Error())
		return nil, errors.NewInternalError(err)
	}

	basicAuthObj, ok := basicAuth.Auths[regBaseUrl]
	if !ok {
		msg := "cannot find cred for " + regBaseUrl + " from the secret"
		log.Info(msg)
		return nil, errors.NewInternalError(fmt.Errorf(msg))
	}

	img, err := trust.NewImage(path.Join(regBaseUrl, repo.Spec.Name), regBaseUrl, "", basicAuthObj.Auth, nil)
	if err != nil {
		log.Info(err.Error())
		return nil, errors.NewInternalError(err)
	}

	var versions []v1.ImageVersion
	if tagExist {
		versions = []v1.ImageVersion{{Version: tag}}
	} else {
		versions = repo.Spec.Versions
	}

	results := map[string]scan.ResultResponse{}
	for _, version := range versions {
		img.Tag = version.Version
		res, err := scan.GetScanResult(img)
		if err != nil {
			log.Info(err.Error())
			return nil, errors.NewInternalError(err)
		}

		results[version.Version] = res
	}

	return results, nil
}
