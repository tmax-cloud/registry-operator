package scanctl

import (
	"fmt"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sync"
	"net/url"

	"k8s.io/apimachinery/pkg/types"
	corev1 "k8s.io/api/core/v1"
	scanv1 "github.com/tmax-cloud/registry-operator/api/v1"
	"github.com/tmax-cloud/registry-operator/controllers/scanctl"
	"github.com/tmax-cloud/registry-operator/internal/common/config"
	"github.com/tmax-cloud/registry-operator/internal/utils/k8s/secrethelper"
	clairReg "github.com/tmax-cloud/registry-operator/pkg/scan/clair"
)

type ScanWorker struct {
	client	*client.Client
	workqueue chan scanv1.ImageScanRequest
	queueSize int
	nWorkers  int
	wait      *sync.WaitGroup
	stopCh    chan bool
}

func NewScanWorker(queueSize, nWorkers int) *ScanWorker {
	c, err := client.New(config.GetConfigOrDie(), client.Options{})
	if err != nil {
		panic(err)
	}

	return &ScanWorker{
		client: c
		workqueue: make(chan scanv1.ImageScanRequest, queueSize),
		queueSize: queueSize,
		nWorkers:  nWorkers,
		wait:      &sync.WaitGroup{},
		stopCh:    make(chan bool, 1),
	}
}

func (s *ScanWorker) GetRequest(o *scanv1.ImageScanRequest) {
	s.workqueue <- o
}

func (s *ScanWorker) Start() {

	for i := 0; i < s.nWorkers; i++ {
		s.wait.Add(1)

		go func() {
			defer s.wait.Done()
			for {
				select {
				case request, isOpened := <-s.workqueue:
					if !isOpened {
						fmt.Println("terminate")
						return
					}
					fmt.Printf("*** Start scanning(%s)\n", request.Name)
					if err := doScan(&request); err != nil {
						fmt.Printf("*** Scan(%s) failed", request.Name)
					}
					onComplete(&request)
					fmt.Printf("*** Done scanning(%s)\n", request.Name)
				}
			}
		}()
	}

	go func() {
		select {
		case <-s.stopCh:
			close(s.workqueue)
		}
	}()
}

func (s *ScanWorker) Stop() {
	s.stopCh <- true
}

func (s *ScanWorker) doScan(instance *scanv1.ImageScanRequest) error {
	
	for _, e := range instance.Spec.ScanTargets {
		ctx := context.TODO()

		secret := &corev1.Secret{}
		if err := s.client.Get(ctx, types.NamespacedName{Name: e.ImagePullSecret, Namespace: namespace}, secret); err != nil {
			return fmt.Sprintf("ImagePullSecret not found: %s\n", e.ImagePullSecret)
		}

		imagePullSecret, err := secrethelper.NewImagePullSecret(secret)
		if err != nil {
			return err
		}

		registryHost := url.Parse(e.RegistryURL).Hostname()
		login, err := imagePullSecret.GetHostCredential(registryHost)
		if err != nil {
			return err
		}

		// FIXME: insecure 허용 시 예외처리
		tlsSecret := &corev1.Secret{}
		if err := s.client.Get(ctx, types.NamespacedName{Name: e.CertificateSecret, Namespace: namespace}, tlsSecret); err != nil {
			return fmt.Sprintf("TLS Secret not found: %s\n", e.CertificateSecret)
		}

		tlsCertData, err := secrethelper.GetCert(tlsSecret, "")
		if err != nil {
			return err
		}

		r, err := newRegistryClient(e.RegistryURL, login.Username, login.Password, tlsCertData)
		if err != nil {
			return err
		}

		c, err := clair.New(config.Config.GetString(config.ConfigClairURL), clair.Opt{
			Debug:    e.Debug,
			Timeout:  e.TimeOut,
			Insecure: e.Insecure,
		})
		if err != nil {
			return err
		}

		
		imageURL := strings.Join([]string{registryHost, e.Images}, "/")
		if isTagExist {
			imageURL = strings.Join([]string{imageURL, tag}, ":")
		}

		image, err := registry.ParseImage(imageURL)
		if err != nil {
			log.Error(err, "failed to parse image")
			return nil, err
		}

	}

	return nil
}

func fetchRepos(image string) ([]string, error) {
	var entries = []string{}

	if strings.Contains(image, "*") || strings.Contains(image, "?") {
		
		entries = append(entries, GetRegistryImages(c, registryURL, basicAuth, image, certificateSecret, namespace)...)
	} else {
		entries = append(entries, image)
	}

	return entries, nil
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