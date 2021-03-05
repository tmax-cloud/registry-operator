package trust

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"time"

	regclient "github.com/docker/distribution/registry/client"
	"github.com/docker/distribution/registry/client/auth/challenge"
	"github.com/opencontainers/go-digest"
	"github.com/theupdateframework/notary"
	"github.com/theupdateframework/notary/client"
	"github.com/theupdateframework/notary/client/changelist"
	"github.com/theupdateframework/notary/cryptoservice"
	store "github.com/theupdateframework/notary/storage"
	"github.com/theupdateframework/notary/trustmanager"
	"github.com/theupdateframework/notary/trustpinning"
	"github.com/theupdateframework/notary/tuf/data"
	apiv1 "github.com/tmax-cloud/registry-operator/api/v1"
	tmaxiov1 "github.com/tmax-cloud/registry-operator/api/v1"
	"github.com/tmax-cloud/registry-operator/internal/common/auth"
	"github.com/tmax-cloud/registry-operator/internal/utils"
	"github.com/tmax-cloud/registry-operator/pkg/image"
	ctrl "sigs.k8s.io/controller-runtime"
)

var log = ctrl.Log.WithName("trust")

const (
	DefaultNotaryServer = "https://notary.docker.io"
)

func NewReadOnly(image *image.Image, notaryURL, path string) (ReadOnly, error) {
	n := &notaryRepo{
		notaryPath: path,
		image:      image,
	}

	// Notary Server url
	if notaryURL == "" {
		n.notaryServerURL = DefaultNotaryServer
	} else {
		n.notaryServerURL = notaryURL
	}

	token, err := n.GetToken()
	if err != nil {
		return nil, err
	}

	// Generate Transport
	rt := &auth.RegistryTransport{
		Base: &http.Transport{ // Base is DefaultTransport, added TLSClientConfig
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
		},
		Token: token,
	}

	// Initialize Notary repository
	repo, err := client.NewFileCachedRepository(n.notaryPath, data.GUN(image.GetImageNameWithHost()), n.notaryServerURL, rt, n.passRetriever(), trustpinning.TrustPinConfig{})
	if err != nil {
		return nil, err
	}
	n.repo = repo

	return n, nil
}

func New(img *image.Image, notaryURL string, passPhrase tmaxiov1.TrustPass, path string, ca []byte, rootKey apiv1.TrustKey, targetKey *apiv1.TrustKey) (NotaryRepository, error) {
	if img == nil {
		return nil, fmt.Errorf("image cannot be nil")
	}
	n := &notaryRepo{
		notaryPath: path,
		image:      img,
		passPhrase: passPhrase,
	}

	// Notary Server url
	if notaryURL == "" {
		n.notaryServerURL = DefaultNotaryServer
	} else {
		n.notaryServerURL = notaryURL
	}

	token, err := n.GetToken()
	if err != nil {
		return nil, err
	}

	// Add CA certificate
	var tlsConfig *tls.Config
	if len(ca) == 0 {
		tlsConfig = &tls.Config{InsecureSkipVerify: true}
	} else {
		caPool := x509.NewCertPool()
		caPool.AppendCertsFromPEM(ca)
		tlsConfig = &tls.Config{
			RootCAs: caPool,
		}
	}

	// Generate Transport
	rt := &auth.RegistryTransport{
		Base: &http.Transport{ // Base is DefaultTransport, added TLSClientConfig
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			TLSClientConfig:       tlsConfig,
		},
		Token: token,
	}

	// get trust key &  write it as a file
	log.Info("Writing root key")
	decRootKey, err := base64.StdEncoding.DecodeString(rootKey.Key)
	if err != nil {
		log.Error(err, "unable to decode base64 string")
		return nil, err
	}
	if err := n.WriteKey(rootKey.ID, decRootKey); err != nil {
		log.Error(err, "")
		return nil, err
	}

	if targetKey != nil {
		log.Info("Writing target key")
		decTargetKey, err := base64.StdEncoding.DecodeString(targetKey.Key)
		if err != nil {
			log.Error(err, "unable to decode base64 string")
			return nil, err
		}
		if err := n.WriteKey(targetKey.ID, decTargetKey); err != nil {
			log.Error(err, "")
			return nil, err
		}
	}

	// Initialize Notary repository
	repo, err := client.NewFileCachedRepository(n.notaryPath, data.GUN(img.GetImageNameWithHost()), n.notaryServerURL, rt, n.passRetriever(), trustpinning.TrustPinConfig{})
	if err != nil {
		return nil, err
	}
	n.repo = repo

	return n, nil
}

func NewDummy(path string) (Writable, error) {
	n := &notaryRepo{
		notaryPath: path,
		passPhrase: apiv1.TrustPass{},
	}

	gun := data.GUN("dummy/dummy:dummy")
	basicPath := filepath.Join(path, "tuf", filepath.FromSlash(gun.String()))
	cache, err := store.NewFileStore(filepath.Join(basicPath, "metadata"), "json")
	if err != nil {
		return nil, err
	}

	keyStores, err := getKeyStores(path, n.passRetriever())
	if err != nil {
		return nil, err
	}

	cryptoService := cryptoservice.NewCryptoService(keyStores...)

	cl, err := changelist.NewFileChangelist(filepath.Join(basicPath, "changelist"))
	if err != nil {
		return nil, err
	}

	n.repo, err = client.NewRepository(gun, "", nil, cache, trustpinning.TrustPinConfig{}, cryptoService, cl)
	if err != nil {
		return nil, err
	}

	return n, nil
}

func getKeyStores(baseDir string, retriever notary.PassRetriever) ([]trustmanager.KeyStore, error) {
	fileKeyStore, err := trustmanager.NewKeyFileStore(baseDir, retriever)
	if err != nil {
		return nil, fmt.Errorf("failed to create private key store in directory: %s", baseDir)
	}
	return []trustmanager.KeyStore{fileKeyStore}, nil
}

type notaryRepo struct {
	notaryPath      string
	notaryServerURL string
	repo            client.Repository
	token           auth.Token
	image           *image.Image
	passPhrase      tmaxiov1.TrustPass
}

func (n *notaryRepo) GetToken() (auth.Token, error) {
	if n.token.Type == "" || n.token.Value == "" {
		if err := n.fetchToken(); err != nil {
			log.Error(err, "")
			return auth.Token{}, err
		}
	}

	return n.token, nil
}

func (n *notaryRepo) fetchToken() error {
	log.Info("Fetching token...")
	// Ping
	u, err := url.Parse(n.notaryServerURL)
	if err != nil {
		return err
	}
	u.Path = path.Join(u.Path, "v2")
	pingReq, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return err
	}
	pingReq.Header.Set("Authorization", fmt.Sprintf("Basic %s", n.image.BasicAuth))
	pingResp, err := n.image.HttpClient.Do(pingReq)
	if err != nil {
		return err
	}
	defer pingResp.Body.Close()

	// If 200, use basic auth
	if pingResp.StatusCode >= 200 && pingResp.StatusCode < 300 {
		n.token = auth.Token{
			Type:  "Basic",
			Value: n.image.BasicAuth,
		}
		return nil
	}

	challenges := challenge.ResponseChallenges(pingResp)
	if len(challenges) < 1 {
		return fmt.Errorf("header does not contain WWW-Authenticate")
	}
	realm, realmExist := challenges[0].Parameters["realm"]
	service, serviceExist := challenges[0].Parameters["service"]
	if !realmExist || !serviceExist {
		return fmt.Errorf("there is no realm or service in parameters")
	}

	// Get Token
	img := n.image.GetImageNameWithHost()

	param := map[string]string{
		"service": service,
		"scope":   fmt.Sprintf("repository:%s:pull,push", img),
	}
	tokenReq, err := http.NewRequest(http.MethodGet, realm, nil)
	if err != nil {
		return err
	}
	tokenReq.Header.Set("Authorization", fmt.Sprintf("Basic %s", n.image.BasicAuth))
	tokenQ := tokenReq.URL.Query()
	for k, v := range param {
		tokenQ.Add(k, v)
	}
	tokenReq.URL.RawQuery = tokenQ.Encode()

	tokenResp, err := n.image.HttpClient.Do(tokenReq)
	if err != nil {
		return err
	}
	defer tokenResp.Body.Close()
	if !regclient.SuccessStatus(tokenResp.StatusCode) {
		err := regclient.HandleErrorResponse(tokenResp)
		return err
	}

	decoder := json.NewDecoder(tokenResp.Body)
	token := &auth.TokenResponse{}
	if err := decoder.Decode(token); err != nil {
		return err
	}

	n.token = auth.Token{
		Type:  "Bearer",
		Value: token.Token,
	}

	return nil
}

func (n *notaryRepo) GetPassphrase(id string) (string, error) {
	return n.passPhrase.GetKeyPass(id)
}

func (n *notaryRepo) CreateRootKey() error {
	log.Info("Creating root key")
	_, err := n.repo.GetCryptoService().Create(data.CanonicalRootRole, "", data.ECDSAKey)
	return err
}

func (n *notaryRepo) SignImage() error {
	log.Info(fmt.Sprintf("Signing image %s", n.image.GetImageNameWithHost()))
	manifest, err := n.image.GetManifest()
	if err != nil {
		log.Error(err, "")
		return err
	}

	// Parse digest
	dgst, err := digest.Parse(manifest.Digest)
	if err != nil {
		log.Error(err, "")
		return err
	}
	h, err := hex.DecodeString(dgst.Hex())
	if err != nil {
		log.Error(err, "")
		return err
	}

	target := &client.Target{
		Name:   n.image.Tag,
		Hashes: data.Hashes{string(dgst.Algorithm()): h},
		Length: manifest.ContentLength,
	}

	if _, err := n.repo.ListTargets(); err != nil {
		switch err.(type) {
		case client.ErrRepoNotInitialized, client.ErrRepositoryNotExist:
			if err := n.InitNotaryRepoWithSigners(); err != nil {
				log.Error(err, "")
				return err
			}
		default:
			log.Error(err, "")
			return err
		}
	}

	if err := n.repo.AddTarget(target, data.CanonicalTargetsRole); err != nil {
		log.Error(err, "")
		return err
	}

	if err := n.repo.Publish(); err != nil {
		log.Error(err, "")
		return err
	}

	return nil
}

func (n *notaryRepo) keyPath(keyId string) string {
	return fmt.Sprintf("%s/private/%s.key", n.notaryPath, keyId)
}

func (n *notaryRepo) WriteKey(keyId string, key []byte) error {
	path := n.keyPath(keyId)
	parent := filepath.Dir(path)
	if err := os.MkdirAll(parent, os.ModePerm); err != nil {
		return err
	}
	return ioutil.WriteFile(path, key, 0600)
}

func (n *notaryRepo) ReadRootKey() (string, []byte, error) {
	key, err := n.getRootKey()
	if err != nil {
		return "", nil, err
	}

	keyVal, err := ioutil.ReadFile(n.keyPath(key))

	return key, keyVal, err
}

func (n *notaryRepo) ReadTargetKey() (string, []byte, error) {
	key, err := n.getTargetKey()
	if err != nil {
		return "", nil, err
	}

	keyVal, err := ioutil.ReadFile(fmt.Sprintf("%s/private/%s.key", n.notaryPath, key))

	return key, keyVal, err
}

func (n *notaryRepo) ClearDir() error {
	return os.RemoveAll(n.notaryPath)
}

func (n *notaryRepo) InitNotaryRepoWithSigners() error {
	rootKey, err := n.getRootKey()
	if err != nil {
		return err
	}

	if err := n.repo.Initialize([]string{rootKey}, data.CanonicalSnapshotRole); err != nil {
		return err
	}

	return nil
}

func (n *notaryRepo) getKey(role data.RoleName) (string, error) {
	keys := n.repo.GetCryptoService().ListKeys(role)
	if len(keys) < 1 {
		return "", fmt.Errorf("no key found with role %s", role.String())
	}
	sort.Strings(keys)
	return keys[0], nil
}

func (n *notaryRepo) getRootKey() (string, error) {
	return n.getKey(data.CanonicalRootRole)
}

func (n *notaryRepo) getTargetKey() (string, error) {
	return n.getKey(data.CanonicalTargetsRole)
}

func (n *notaryRepo) passRetriever() notary.PassRetriever {
	return func(id, _ string, createNew bool, attempts int) (string, bool, error) {
		if createNew {
			n.passPhrase.AddKeyPass(id, utils.RandomString(10))
		}
		phrase, ok := n.passPhrase[id]
		if !ok {
			return "", attempts > 1, fmt.Errorf("no pass phrase is found")
		}
		return phrase, attempts > 1, nil
	}
}

func (n *notaryRepo) GetSignedMetadata(tag string) (*trustRepo, error) {
	allSignedTargets, err := n.repo.GetAllTargetMetadataByName(tag)
	if err != nil {
		log.Error(err, "failed to get all target metadata")
		return &trustRepo{}, err
	}

	signatureRows := matchReleasedSignatures(allSignedTargets)

	// get the administrative roles
	adminRolesWithSigs, err := n.repo.ListRoles()
	if err != nil {
		return &trustRepo{}, fmt.Errorf("No signers for %s", n.notaryServerURL)
	}

	// get delegation roles with the canonical key IDs
	delegationRoles, err := n.repo.GetDelegationRoles()
	if err != nil {
		log.Error(err, "no delegation roles found, or error fetching them for %s", n.notaryServerURL)
	}

	// process the signatures to include repo admin if signed by the base targets role
	for idx, sig := range signatureRows {
		if len(sig.Signers) == 0 {
			signatureRows[idx].Signers = append(sig.Signers, releasedRoleName)
		}
	}

	signerList, adminList := []trustSigner{}, []trustSigner{}

	signerRoleToKeyIDs := getDelegationRoleToKeyMap(delegationRoles)

	for signerName, signerKeys := range signerRoleToKeyIDs {
		signerKeyList := []trustKey{}
		for _, keyID := range signerKeys {
			signerKeyList = append(signerKeyList, trustKey{ID: keyID})
		}
		signerList = append(signerList, trustSigner{signerName, signerKeyList})
	}
	sort.Slice(signerList, func(i, j int) bool { return signerList[i].Name > signerList[j].Name })

	for _, adminRole := range adminRolesWithSigs {
		switch adminRole.Name {
		case data.CanonicalRootRole:
			rootKeys := []trustKey{}
			for _, keyID := range adminRole.KeyIDs {
				rootKeys = append(rootKeys, trustKey{ID: keyID})
			}
			adminList = append(adminList, trustSigner{"Root", rootKeys})
		case data.CanonicalTargetsRole:
			targetKeys := []trustKey{}
			for _, keyID := range adminRole.KeyIDs {
				targetKeys = append(targetKeys, trustKey{ID: keyID})
			}
			adminList = append(adminList, trustSigner{"Repository", targetKeys})
		}
	}
	sort.Slice(adminList, func(i, j int) bool { return adminList[i].Name > adminList[j].Name })

	return &trustRepo{
		Name:               n.repo.GetGUN().String(),
		SignedTags:         signatureRows,
		Signers:            signerList,
		AdministrativeKeys: adminList,
	}, nil
}

func (n *notaryRepo) DeleteSign() {
	// TODO
}
