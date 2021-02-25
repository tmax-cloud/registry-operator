package image

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"path"
	"strings"

	"github.com/docker/distribution/reference"
	"github.com/docker/distribution/registry/client"
	"github.com/docker/distribution/registry/client/auth/challenge"
	"github.com/tmax-cloud/registry-operator/internal/common/auth"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var Logger = log.Log.WithName("image-client")

const (
	// DefaultServer is the default registry server
	DefaultServer = "https://registry-1.docker.io"
	// DefaultHostname is the default built-in hostname
	DefaultHostname = "docker.io"
)

type Image struct {
	ServerURL string

	Host         string
	Name         string
	FamiliarName string
	Tag          string
	Digest       string

	// username:password string encrypted by base64
	BasicAuth string
	Token     auth.Token

	HttpClient http.Client
}

// NewImage creates new image client
func NewImage(uri, registryServer, basicAuth string, ca []byte) (*Image, error) {
	r := &Image{}

	// Server url
	if registryServer == "" || strings.HasPrefix(uri, DefaultHostname) {
		r.ServerURL = DefaultServer
	} else {
		// set protocol scheme
		if !strings.HasPrefix(registryServer, "http://") && !strings.HasPrefix(registryServer, "https://") {
			registryServer = "https://" + registryServer
		}
		r.ServerURL = registryServer
	}

	// Set image
	if uri != "" {
		if err := r.SetImage(uri); err != nil {
			Logger.Error(err, "failed to set image", "uri", uri)
			return nil, err
		}
	}

	// Auth
	r.BasicAuth = basicAuth
	r.Token = auth.Token{}

	// Generate HTTPS client
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
	r.HttpClient = http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}

	return r, nil
}

// SetImage sets image from "[<server>/]<imageName>[:<tag>|@<digest>]" form argument
func (r *Image) SetImage(image string) error {
	// Parse image
	var img reference.Named
	var err error
	if r.ServerURL == "" || strings.HasPrefix(r.ServerURL, DefaultHostname) {
		r.ServerURL = DefaultServer
	}
	if r.ServerURL == DefaultServer {
		img, err = reference.ParseNormalizedNamed(image)
		if err != nil {
			Logger.Error(err, "failed to parse image", "image", image)
			return err
		}
		r.FamiliarName = reference.FamiliarName(img)
	} else {
		img, err = reference.ParseNamed(image)
		if err != nil {
			uri := r.ServerURL
			uri = strings.TrimPrefix(uri, "http://")
			uri = strings.TrimPrefix(uri, "https://")
			uri = path.Join(uri, image)
			img, err = reference.ParseNamed(uri)
			if err != nil {
				Logger.Error(err, "failed to parse uri", "uri", uri)
				return err
			}
		}
		r.FamiliarName = reference.Path(img)
	}

	r.Host, r.Name = reference.SplitHostname(img)
	refered := false
	r.Digest = ""
	r.Tag = ""
	if canonical, isCanonical := img.(reference.Canonical); isCanonical {
		r.Digest = canonical.Digest().String()
		refered = true
	}

	img = reference.TagNameOnly(img)
	if tagged, isTagged := img.(reference.NamedTagged); isTagged {
		r.Tag = tagged.Tag()
		refered = true
	}

	if !refered {
		return fmt.Errorf("no tag and digest given")
	}

	return nil
}

func (r *Image) GetImageNameWithHost() string {
	return path.Join(r.Host, r.Name)
}

func (r *Image) GetToken(scope string) (auth.Token, error) {
	if scope == "" {
		if r.Name == "" {
			scope = catalogScope()
		} else {
			scope = repositoryScope(r.Name)
		}
	}

	if err := r.fetchToken(scope); err != nil {
		Logger.Error(err, "")
		return auth.Token{}, err
	}

	return r.Token, nil
}

func (r *Image) fetchToken(scope string) error {
	Logger.Info("Fetching token...")
	// Ping
	u, err := pingURL(r.ServerURL)
	if err != nil {
		return err
	}

	pingReq, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return err
	}
	pingReq.Header.Set("Authorization", fmt.Sprintf("Basic %s", r.BasicAuth))
	pingResp, err := r.HttpClient.Do(pingReq)
	if err != nil {
		return err
	}
	defer pingResp.Body.Close()

	// If 200, use basic auth
	if pingResp.StatusCode >= 200 && pingResp.StatusCode < 300 {
		r.Token = auth.Token{
			Type:  "Basic",
			Value: base64.StdEncoding.EncodeToString([]byte(r.BasicAuth)),
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
	param := map[string]string{
		"service": service,
		"scope":   scope,
	}
	tokenReq, err := http.NewRequest(http.MethodGet, realm, nil)
	if err != nil {
		return err
	}
	tokenReq.Header.Set("Authorization", fmt.Sprintf("Basic %s", r.BasicAuth))
	tokenQ := tokenReq.URL.Query()
	for k, v := range param {
		tokenQ.Add(k, v)
	}
	tokenReq.URL.RawQuery = tokenQ.Encode()

	tokenResp, err := r.HttpClient.Do(tokenReq)
	if err != nil {
		return err
	}
	defer tokenResp.Body.Close()
	if !client.SuccessStatus(tokenResp.StatusCode) {
		err := client.HandleErrorResponse(tokenResp)
		return err
	}

	decoder := json.NewDecoder(tokenResp.Body)
	token := &auth.TokenResponse{}
	if err := decoder.Decode(token); err != nil {
		return err
	}

	r.Token = auth.Token{
		Type:  "Bearer",
		Value: token.Token,
	}

	return nil
}
