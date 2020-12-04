package trust

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"fmt"
	"github.com/opencontainers/go-digest"
	"github.com/theupdateframework/notary"
	"github.com/theupdateframework/notary/client"
	"github.com/theupdateframework/notary/trustpinning"
	"github.com/theupdateframework/notary/tuf/data"
	tmaxiov1 "github.com/tmax-cloud/registry-operator/api/v1"
	"github.com/tmax-cloud/registry-operator/internal/utils"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	ctrl "sigs.k8s.io/controller-runtime"
	"sort"
	"time"
)

var log = ctrl.Log.WithName("trust")

type notaryRepo struct {
	notaryPath string
	repo       client.Repository
	image      *Image
	passPhrase tmaxiov1.TrustPass
}

func New(image *Image, passPhrase tmaxiov1.TrustPass, path string, ca []byte) (*notaryRepo, error) {
	if image == nil {
		return nil, fmt.Errorf("image cannot be nil")
	}
	n := &notaryRepo{
		notaryPath: path,
		image:      image,
		passPhrase: passPhrase,
	}
	token, err := image.GetToken(TokenTypeNotary)
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
	rt := &RegistryTransport{
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

	// Initialize Notary repository
	repo, err := client.NewFileCachedRepository(n.notaryPath, data.GUN(image.GetImageNameWithHost()), image.NotaryServerUrl, rt, n.passRetriever(), trustpinning.TrustPinConfig{})
	if err != nil {
		return nil, err
	}
	n.repo = repo

	return n, nil
}

func (n notaryRepo) GetPassphrase(id string) (string, error) {
	return n.passPhrase.GetKeyPass(id)
}

func (n notaryRepo) CreateRootKey() error {
	log.Info(fmt.Sprintf("Creating root key"))
	_, err := n.repo.GetCryptoService().Create(data.CanonicalRootRole, "", data.ECDSAKey)
	return err
}

func (n notaryRepo) SignImage() error {
	log.Info(fmt.Sprintf("Signing image %s", n.image.GetImageNameWithHost()))
	imgDigest, size, err := n.image.GetImageManifest()
	if err != nil {
		return err
	}

	// Parse digest
	dgst, err := digest.Parse(imgDigest)
	if err != nil {
		return err
	}
	h, err := hex.DecodeString(dgst.Hex())
	if err != nil {
		return err
	}

	target := &client.Target{
		Name:   n.image.Tag,
		Hashes: data.Hashes{string(dgst.Algorithm()): h},
		Length: size,
	}

	if _, err := n.repo.ListTargets(); err != nil {
		switch err.(type) {
		case client.ErrRepoNotInitialized, client.ErrRepositoryNotExist:
			if err := n.InitNotaryRepoWithSigners(); err != nil {
				return err
			}
		default:
			return err
		}
	}

	err = n.repo.AddTarget(target, data.CanonicalTargetsRole)

	if err := n.repo.Publish(); err != nil {
		return err
	}

	return nil
}

func (n notaryRepo) keyPath(keyId string) string {
	return fmt.Sprintf("%s/private/%s.key", n.notaryPath, keyId)
}

func (n notaryRepo) WriteKey(keyId string, key []byte) error {
	return ioutil.WriteFile(n.keyPath(keyId), key, 0600)
}

func (n notaryRepo) ReadRootKey() (string, []byte, error) {
	key, err := n.getRootKey()
	if err != nil {
		return "", nil, err
	}

	keyVal, err := ioutil.ReadFile(n.keyPath(key))

	return key, keyVal, err
}

func (n notaryRepo) ReadTargetKey() (string, []byte, error) {
	key, err := n.getTargetKey()
	if err != nil {
		return "", nil, err
	}

	keyVal, err := ioutil.ReadFile(fmt.Sprintf("%s/private/%s.key", n.notaryPath, key))

	return key, keyVal, err
}

func (n notaryRepo) ClearDir() error {
	return os.RemoveAll(n.notaryPath)
}

func (n notaryRepo) InitNotaryRepoWithSigners() error {
	rootKey, err := n.getRootKey()
	if err != nil {
		return err
	}

	if err := n.repo.Initialize([]string{rootKey}, data.CanonicalSnapshotRole); err != nil {
		return err
	}

	return nil
}

func (n notaryRepo) getKey(role data.RoleName) (string, error) {
	keys := n.repo.GetCryptoService().ListKeys(role)
	if len(keys) < 1 {
		return "", fmt.Errorf("no root key found")
	}
	sort.Strings(keys)
	return keys[0], nil
}

func (n notaryRepo) getRootKey() (string, error) {
	return n.getKey(data.CanonicalRootRole)
}

func (n notaryRepo) getTargetKey() (string, error) {
	return n.getKey(data.CanonicalTargetsRole)
}

func (n notaryRepo) passRetriever() notary.PassRetriever {
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
