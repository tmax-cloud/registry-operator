package secret

import (
	"fmt"
	"k8s.io/api/core/v1"
)

type CertSecret struct {
	Secret *corev1.Secret
	tls *Tls
}

type Tls struct {
	TlsCert []byte
	TlsKey []byte
}

func NewCertAuth(secret *v1.Secret) *CertProvider {
	instance := CertSecret{secret}

	err := instance.init()
	if err != nil {
		fmt.Println("Failed to initialize secret provider")
	}

	return &instance
}

func (p *CertSecret) init() error {
	
	switch p.secret.Type {
	case v1.SecretTypeTls:
		p.tls.TlsKey, p.tls.TlsCert, err = p.parseTls()
		if err != nil {
			return err
		}
	// case v1.SecretTypeOpaque:
	// 	p.user, p.password, err := p.parseOpaque()
	default:
		return errors.New("Unsupported Secret type.")
	} 

	return nil
}

func (p *CertSecret) getKey() (string, error) {
	if len(p.tls.TlsKey) < 1 {
		return nil, errors.New("User ID is empty")
	}
	return p.tls.TlsKey, nil
}

func (p *CertSecret) getCert() (string, error) {
	if len(p.tls.TlsCert) < 1 {
		return nil, errors.New("TlsCert is empty")
	}
	return p.tls.TlsCert, nil
}

func (p *CertSecret) parseTls() (key, cert []byte, error) {
	
	key, ok := p.Secret.Data[v1.TLSPrivateKeyKey]
	if !ok {
		return nil, nil, errors.New("Invalid private key")
	}

	cert, ok := p.Secret.Data[v1.TLSCertKey]
	if !ok {
		return nil, nil, errors.New("Invalid TLS certificate")
	}

	return key, cert, nil
}

// func (p *CertSecret) parseOpaqueSecret() (id, password string, error) {

// }