package v1

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"path"
	"strings"
	"time"

	gerr "errors"

	"github.com/genuinetools/reg/clair"
	"github.com/genuinetools/reg/registry"
	"github.com/genuinetools/reg/repoutils"
	"github.com/gorilla/mux"
	v1 "github.com/tmax-cloud/registry-operator/api/v1"
	"github.com/tmax-cloud/registry-operator/internal/common/certs"
	config "github.com/tmax-cloud/registry-operator/internal/common/config"
	"github.com/tmax-cloud/registry-operator/internal/schemes"
	"github.com/tmax-cloud/registry-operator/internal/utils"
	"github.com/tmax-cloud/registry-operator/internal/wrapper"
	"github.com/tmax-cloud/registry-operator/pkg/scan"
	clairReg "github.com/tmax-cloud/registry-operator/pkg/scan/clair"
	"github.com/tmax-cloud/registry-operator/pkg/trust"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
)

const (
	RepositoryKind    = "repositories"
	ExtRepositoryKind = "ext-repositories"
	ScanResultKind    = "imagescanresults"

	RepositoryParamKey = "repositoryName"
	TagParamKey        = "tagName"
)

func AddScanResult(parent *wrapper.RouterWrapper) error {
	listScanSummaryWrapper := wrapper.New(fmt.Sprintf("/%s/{%s}/%s", RepositoryKind, RepositoryParamKey, ScanResultKind), []string{http.MethodGet}, listScanSummaryHandler)
	if err := parent.Add(listScanSummaryWrapper); err != nil {
		return err
	}

	scanResultWrapper := wrapper.New(fmt.Sprintf("/%s/{%s}/%s/{%s}", RepositoryKind, RepositoryParamKey, ScanResultKind, TagParamKey), []string{http.MethodGet}, scanResultHandler)
	if err := parent.Add(scanResultWrapper); err != nil {
		return err
	}

	listExtScanSummaryWrapper := wrapper.New(fmt.Sprintf("/%s/{%s}/%s", ExtRepositoryKind, RepositoryParamKey, ScanResultKind), []string{http.MethodGet}, listExtScanSummaryHandler)
	if err := parent.Add(listExtScanSummaryWrapper); err != nil {
		return err
	}

	scanExtResultWrapper := wrapper.New(fmt.Sprintf("/%s/{%s}/%s/{%s}", ExtRepositoryKind, RepositoryParamKey, ScanResultKind, TagParamKey), []string{http.MethodGet}, extScanResultHandler)
	if err := parent.Add(scanExtResultWrapper); err != nil {
		return err
	}

	return nil
}

// Return summary of vulnerabilities
func listScanSummaryHandler(w http.ResponseWriter, req *http.Request) {
	results, err := getScanResultFromInternal(req)
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
func listExtScanSummaryHandler(w http.ResponseWriter, req *http.Request) {
	results, err := getScanResultFromExternal(req)
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
	results, err := getScanResultFromInternal(req)
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
func extScanResultHandler(w http.ResponseWriter, req *http.Request) {
	results, err := getScanResultFromExternal(req)
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

func getScanResultFromInternal(req *http.Request) (map[string]scan.ResultResponse, error) {
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
	if err := k8sClient.Get(context.Background(), types.NamespacedName{Name: v1.K8sPrefix + v1.K8sRegistryPrefix + strings.ToLower(reg.Name), Namespace: ns}, secret); err != nil {
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

	img, err := trust.NewImage(path.Join(regBaseUrl, repo.Spec.Name), "https://"+regBaseUrl, "", basicAuthObj.Auth, nil)
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
			continue
		}

		results[version.Version] = res
	}

	return results, nil
}

func getScanResultFromExternal(req *http.Request) (map[string]scan.ResultResponse, error) {
	reqId := utils.RandomString(10)
	log := logger.WithValues("request", reqId)

	// Get path parameters
	vars := mux.Vars(req)

	namespace, isNamespaceExist := vars[NamespaceParamKey]
	repository, isRepositoryExist := vars[RepositoryParamKey]
	tag, isTagExist := vars[TagParamKey]

	if !isNamespaceExist || !isRepositoryExist {
		return nil, errors.NewBadRequest("url is malformed")
	}

	log.Info(fmt.Sprintf("*** namespace: %s/ repository: %s / tag: %s", namespace, repository, tag))

	ctx := context.Background()
	repo := &v1.Repository{}
	if err := k8sClient.Get(ctx, types.NamespacedName{Name: repository, Namespace: namespace}, repo); err != nil {
		log.Info(err.Error())
		return nil, errors.NewInternalError(err)
	}

	reg := &v1.ExternalRegistry{}
	if err := k8sClient.Get(ctx, types.NamespacedName{Name: repo.Spec.Registry, Namespace: namespace}, reg); err != nil {
		log.Info(err.Error())
		return nil, errors.NewInternalError(err)
	}

	imagePullSecret := &corev1.Secret{}
	if err := k8sClient.Get(ctx, types.NamespacedName{Name: reg.Spec.ImagePullSecret, Namespace: namespace}, imagePullSecret); err != nil {
		log.Info(err.Error())
		return nil, errors.NewInternalError(gerr.New(fmt.Sprintf("Failed to get ImagePullSecret(%s)", reg.Spec.ImagePullSecret)))
	}

	imagePullSecretData, ok := imagePullSecret.Data[schemes.DockerConfigJson]
	if !ok {
		return nil, errors.NewInternalError(gerr.New(fmt.Sprintf("Failed to get dockerconfig from ImagePullSecret")))
	}

	dockerConfig := &schemes.DockerConfig{}
	if err := json.Unmarshal(imagePullSecretData, dockerConfig); err != nil {
		return nil, errors.NewInternalError(gerr.New(fmt.Sprintf("Failed to unmarshal ImagePullSecret(%s)'s dockerconfig", reg.Spec.ImagePullSecret)))
	}

	registryHostname := strings.TrimPrefix(reg.Spec.RegistryURL, "https://")
	dockerConfigAuths, ok := dockerConfig.Auths[registryHostname]
	if !ok {
		return nil, errors.NewInternalError(gerr.New(fmt.Sprintf("Failed to get dockerconfig[%s].auths", registryHostname)))
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
	if err := k8sClient.Get(ctx, types.NamespacedName{Name: reg.Spec.CertificateSecret, Namespace: namespace}, certSecret); err != nil {
		return nil, errors.NewInternalError(gerr.New(fmt.Sprintf("Failed to get dockerconfig[%s].auths", reg.Spec.RegistryURL)))
	}

	var tlsCertData []byte
	if certSecret.Type == corev1.SecretTypeTLS {
		tlsCertData, ok = certSecret.Data[corev1.TLSCertKey]
		if !ok {
			return nil, errors.NewInternalError(gerr.New(fmt.Sprintf("Failed to get TLS Certificate from CertificateSecret")))
		}
	} else if certSecret.Type == corev1.SecretTypeOpaque {
		// FIXME: if cert's key name is random.
		tlsCertData, ok = certSecret.Data["ca.crt"]
		if !ok {
			return nil, errors.NewInternalError(gerr.New(fmt.Sprintf("Failed to get TLS Certificate from CertificateSecret")))
		}
	} else {
		return nil, errors.NewInternalError(gerr.New(fmt.Sprintf("Failed to get TLS Certificate from CertificateSecret")))
	}

	imageURL := strings.Join([]string{registryHostname, repo.Spec.Name}, "/")
	if isTagExist {
		imageURL = strings.Join([]string{imageURL, tag}, ":")
	}

	image, err := registry.ParseImage(imageURL)
	if err != nil {
		log.Error(err, "failed to parse image")
		return nil, err
	}

	// Create the registry client.
	r, err := newRegistryClient(reg.Spec.RegistryURL, username, password, tlsCertData)
	if err != nil {
		log.Error(err, "Failed to create registry client")
		return nil, err
	}

	// Initialize clair client.
	cr, err := clair.New(config.Config.GetString(config.ConfigClairURL), clair.Opt{
		Debug:    false,
		Timeout:  time.Second * 3,
		Insecure: false,
	})
	if err != nil {
		log.Error(err, "Failed to new clair client")
		return nil, err
	}

	log.Info(fmt.Sprintf("*** registry: %s/ clair: %s", r.URL, cr.URL))

	report := clair.VulnerabilityReport{}
	// Get the vulnerability report.
	if report, err = cr.Vulnerabilities(ctx, r, image.Path, image.Reference()); err != nil {
		log.Error(err, "failed to get image vulnerabilities")
		return nil, err
	}

	var versions []v1.ImageVersion
	if isTagExist {
		versions = []v1.ImageVersion{{Version: tag}}
	} else {
		versions = repo.Spec.Versions
	}

	results := map[string]scan.ResultResponse{}
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
