package v1

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/tmax-cloud/registry-operator/pkg/apiserver/models"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/genuinetools/reg/clair"
	"github.com/genuinetools/reg/registry"
	"github.com/genuinetools/reg/repoutils"
	"github.com/gorilla/mux"
	v1 "github.com/tmax-cloud/registry-operator/api/v1"
	"github.com/tmax-cloud/registry-operator/internal/common/certs"
	config "github.com/tmax-cloud/registry-operator/internal/common/config"
	"github.com/tmax-cloud/registry-operator/internal/schemes"
	"github.com/tmax-cloud/registry-operator/internal/utils"
	"github.com/tmax-cloud/registry-operator/pkg/image"
	clairReg "github.com/tmax-cloud/registry-operator/pkg/scan/clair"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
)

const (
	RepositoryParamKey = "repositoryName"
	TagParamKey        = "tagName"
)

// Return summary of vulnerabilities
func (h *RegistryAPI) ScanResultSummaryList(w http.ResponseWriter, req *http.Request) {
	results, err := h.getScanResultFromInternal(req)
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

// Return summary of vulnerabilities
func (h *RegistryAPI) ExternalScanResultSummaryList(w http.ResponseWriter, req *http.Request) {
	results, err := h.getScanResultFromExternal(req)
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
func (h *RegistryAPI) ScanResultHandler(w http.ResponseWriter, req *http.Request) {
	results, err := h.getScanResultFromInternal(req)
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

// Return actual list of vulnerabilities
func (h *RegistryAPI) ExtScanResultHandler(w http.ResponseWriter, req *http.Request) {
	results, err := h.getScanResultFromExternal(req)
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

func (h *RegistryAPI) getScanResultFromInternal(r *http.Request) (map[string]models.ResultResponse, error) {
	vars := mux.Vars(r)
	namespace, namespaceOk := vars[NamespaceParamKey]
	name, nameOk := vars[RepositoryParamKey]
	if !namespaceOk || !nameOk {
		return nil, errors.NewBadRequest("url is malformed")
	}
	tag, tagExist := vars[TagParamKey]

	ctx := r.Context()
	log := h.logger.WithValues("namespace", namespace, "name", name)

	repo := &v1.Repository{}
	if err := h.c.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, repo); err != nil {
		log.Error(err, "failed to get XXXX")
		return nil, errors.NewInternalError(err)
	}

	reg := &v1.Registry{}
	if err := h.c.Get(ctx, types.NamespacedName{Name: repo.Spec.Registry, Namespace: namespace}, reg); err != nil {
		log.Error(err, "failed to get XXXX")
		return nil, errors.NewInternalError(err)
	}

	regBaseUrl := strings.TrimPrefix(reg.Status.ServerURL, "https://")

	// TODO - functionize
	secret := &corev1.Secret{}
	if err := h.c.Get(ctx, types.NamespacedName{Name: v1.K8sPrefix + v1.K8sRegistryPrefix + strings.ToLower(reg.Name), Namespace: namespace}, secret); err != nil {
		log.Error(err, "failed to get XXXX")
		return nil, errors.NewInternalError(err)
	}

	authStr, ok := secret.Data[corev1.DockerConfigJsonKey]
	if !ok {
		msg := "cannot find .dockerconfigjson from the secret"
		log.Info(msg)
		return nil, errors.NewInternalError(fmt.Errorf(msg))
	}

	basicAuth := &schemes.DockerConfig{}
	if err := json.Unmarshal(authStr, basicAuth); err != nil {
		log.Error(err, "failed to get XXXX")
		return nil, errors.NewInternalError(err)
	}
	basicAuthObj, ok := basicAuth.Auths[regBaseUrl]
	if !ok {
		msg := "cannot find cred for " + regBaseUrl + " from the secret"
		log.Info(msg)
		return nil, errors.NewInternalError(fmt.Errorf(msg))
	}

	img, err := image.NewImage(path.Join(regBaseUrl, repo.Spec.Name), "https://"+regBaseUrl, basicAuthObj.Auth, nil)
	if err != nil {
		log.Error(err, "failed to get XXXX")
		return nil, errors.NewInternalError(err)
	}

	var versions []v1.ImageVersion
	if tagExist {
		versions = []v1.ImageVersion{{Version: tag}}
	} else {
		versions = repo.Spec.Versions
	}

	results := map[string]models.ResultResponse{}
	for _, version := range versions {
		img.Tag = version.Version
		res, err := models.GetScanResult(img)
		if err != nil {
			log.Error(err, "failed to get XXXX")
			continue
		}
		results[version.Version] = res
	}

	return results, nil
}

func (h *RegistryAPI) getScanResultFromExternal(r *http.Request) (map[string]models.ResultResponse, error) {
	vars := mux.Vars(r)
	namespace, namespaceOk := vars[NamespaceParamKey]
	repository, repositoryOk := vars[RepositoryParamKey]
	if !namespaceOk || !repositoryOk {
		return nil, errors.NewBadRequest("url is malformed")
	}
	tag, isTagExist := vars[TagParamKey]

	ctx := r.Context()
	log := h.logger.WithValues("namespace", namespace, "name", repository)

	repo := &v1.Repository{}
	if err := h.c.Get(ctx, types.NamespacedName{Name: repository, Namespace: namespace}, repo); err != nil {
		log.Error(err, "failed to get XXX")
		return nil, errors.NewInternalError(err)
	}

	reg := &v1.ExternalRegistry{}
	if err := h.c.Get(ctx, types.NamespacedName{Name: repo.Spec.Registry, Namespace: namespace}, reg); err != nil {
		log.Error(err, "failed to get XXX")
		return nil, errors.NewInternalError(err)
	}

	imagePullSecret := &corev1.Secret{}
	if err := h.c.Get(ctx, types.NamespacedName{Name: reg.Status.LoginSecret, Namespace: namespace}, imagePullSecret); err != nil {
		log.Error(err, "failed to get XXX")
		return nil, errors.NewInternalError(fmt.Errorf("Failed to get ImagePullSecret(%s)", reg.Status.LoginSecret))
	}

	imagePullSecretData, ok := imagePullSecret.Data[corev1.DockerConfigJsonKey]
	if !ok {
		return nil, errors.NewInternalError(fmt.Errorf("Failed to get dockerconfig from ImagePullSecret"))
	}

	dockerConfig := &schemes.DockerConfig{}
	if err := json.Unmarshal(imagePullSecretData, dockerConfig); err != nil {
		return nil, errors.NewInternalError(fmt.Errorf("Failed to unmarshal ImagePullSecret(%s)'s dockerconfig", reg.Status.LoginSecret))
	}

	registryHostname := strings.TrimPrefix(reg.Spec.RegistryURL, "https://")
	dockerConfigAuths, ok := dockerConfig.Auths[registryHostname]
	if !ok {
		return nil, errors.NewInternalError(fmt.Errorf("Failed to get dockerconfig[%s].auths", registryHostname))
	}
	basicAuthCredential := dockerConfigAuths.Auth
	decoded, err := base64.StdEncoding.DecodeString(basicAuthCredential)
	if err != nil {
		logger.Error(err, "failed to decode string by base64")
		return nil, err
	}

	decodedCredential := string(decoded)
	sepIdx := strings.Index(decodedCredential, ":")
	username := decodedCredential[:sepIdx]
	password := decodedCredential[sepIdx+1:]

	certSecret := &corev1.Secret{}
	if err := h.c.Get(ctx, types.NamespacedName{Name: reg.Spec.CertificateSecret, Namespace: namespace}, certSecret); err != nil {
		return nil, errors.NewInternalError(fmt.Errorf("Failed to get dockerconfig[%s].auths", reg.Spec.RegistryURL))
	}

	var tlsCertData []byte
	if certSecret.Type == corev1.SecretTypeTLS {
		tlsCertData, ok = certSecret.Data[corev1.TLSCertKey]
		if !ok {
			return nil, errors.NewInternalError(fmt.Errorf("Failed to get TLS Certificate from CertificateSecret"))
		}
	} else if certSecret.Type == corev1.SecretTypeOpaque {
		// FIXME: if cert's key name is random.
		tlsCertData, ok = certSecret.Data["ca.crt"]
		if !ok {
			return nil, errors.NewInternalError(fmt.Errorf("Failed to get TLS Certificate from CertificateSecret"))
		}
	} else {
		return nil, errors.NewInternalError(fmt.Errorf("Failed to get TLS Certificate from CertificateSecret"))
	}

	imageURL := strings.Join([]string{registryHostname, repo.Spec.Name}, "/")
	if isTagExist {
		imageURL = strings.Join([]string{imageURL, tag}, ":")
	}

	img, err := registry.ParseImage(imageURL)
	if err != nil {
		log.Error(err, "failed to parse image")
		return nil, err
	}

	// Create the registry client.
	regCli, err := newRegistryClient(reg.Spec.RegistryURL, username, password, tlsCertData)
	if err != nil {
		log.Error(err, "Failed to create registry client")
		return nil, err
	}

	// Initialize clair client.
	cr, err := clair.New(config.Config.GetString(config.ConfigImageScanSvr), clair.Opt{
		Debug:    false,
		Timeout:  time.Second * 3,
		Insecure: false,
	})
	if err != nil {
		log.Error(err, "Failed to new clair client")
		return nil, err
	}

	// Get the vulnerability report.
	report, err := cr.Vulnerabilities(ctx, regCli, img.Path, img.Reference())
	if err != nil {
		log.Error(err, "failed to get image vulnerabilities")
		return nil, err
	}
	var versions []v1.ImageVersion
	if isTagExist {
		versions = []v1.ImageVersion{{Version: tag}}
	} else {
		versions = repo.Spec.Versions
	}

	results := map[string]models.ResultResponse{}
	for _, version := range versions {
		results[version.Version] = report.VulnsBySeverity
	}

	return results, nil
}

func newRegistryClient(url, username, password string, ca []byte) (*registry.Registry, error) {
	// get keycloak ca if exists
	keycloakCA, err := certs.GetSystemKeycloakCert(nil)
	if err != nil && !errors.IsNotFound(err) {
		logger.Error(err, "failed to get system keycloak cert")
		return nil, err
	}
	if keycloakCA != nil {
		caCert, _ := certs.CAData(keycloakCA)
		ca = append(ca, caCert...)
	}

	// get auth config
	authCfg, err := repoutils.GetAuthConfig(username, password, url)
	if err != nil {
		logger.Error(err, "failed to get auth config")
		return nil, err
	}

	// Create the registry client.
	return clairReg.New(context.TODO(), authCfg, registry.Opt{
		Insecure: false,
		Debug:    false,
		SkipPing: true,
		NonSSL:   false,
		Timeout:  time.Second * 5,
	}, ca)
}
