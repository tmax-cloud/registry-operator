package utils

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"strings"
	"time"

	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

type CertPair struct {
	ParentCert *x509.Certificate
	ParentKey  *rsa.PrivateKey

	Template *x509.Certificate
	Key      *rsa.PrivateKey

	X509CertData []byte
}

func NewCertPair(key *rsa.PrivateKey, isCA bool) (*CertPair, error) {
	if key == nil {
		pKey, err := rsa.GenerateKey(rand.Reader, 4096)
		if err != nil {
			return nil, err
		}

		key = pKey
	}

	serialNumber, err := generateSerialNumber()
	if err != nil {
		return nil, err
	}

	return &CertPair{
		Template: &x509.Certificate{
			SerialNumber:          serialNumber,
			NotBefore:             time.Now(),
			NotAfter:              time.Now().Add(time.Hour * 24 * 365 * 10),
			KeyUsage:              x509.KeyUsageCRLSign,
			ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
			IsCA:                  isCA,
			BasicConstraintsValid: true,
		},
		Key: key,
	}, nil
}

func (c *CertPair) SetParent(cert *x509.Certificate, key *rsa.PrivateKey) {
	c.ParentCert = cert
	c.ParentKey = key
}

func (c *CertPair) SetSubject(subject *pkix.Name) {
	c.Template.Subject = *subject
}

func (c *CertPair) SetSubjectAltName(ips []net.IP, domains []string) {
	c.Template.IPAddresses = append(c.Template.IPAddresses, ips...)
	c.Template.DNSNames = append(c.Template.DNSNames, domains...)
}

var utilLogger = logf.Log.WithName("cert")

func (c *CertPair) CreateCertificateData() error {
	certData, err := x509.CreateCertificate(rand.Reader, c.Template, c.ParentCert, &c.Key.PublicKey, c.ParentKey)
	if err != nil {
		return err
	}
	c.X509CertData = certData

	return nil
}

func (c *CertPair) CertDataToPem() ([]byte, error) {
	if c.X509CertData == nil {
		return nil, fmt.Errorf("x509CertData is not set. you must create certificate before endcoding.")
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: c.X509CertData})

	return certPEM, nil
}

func (c *CertPair) KeyToPem() ([]byte, error) {
	KeyData, err := x509.MarshalPKCS8PrivateKey(c.Key)
	if err != nil {
		return nil, err
	}

	return pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: KeyData}), nil
}

func generateSerialNumber() (*big.Int, error) {
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, fmt.Errorf("failed to generate serial number: %s", err.Error())
	}

	return serialNumber, nil
}

func RemovePemBlock(data []byte, blockType string) []byte {
	dataStr := string(data)
	// utilLogger.Info("trim", "before", dataStr)
	dataStr = strings.ReplaceAll(dataStr, "\r", "")
	dataStr = strings.TrimLeft(dataStr, "-----BEGIN "+blockType+"-----\n")
	dataStr = strings.TrimRight(dataStr, "-----END "+blockType+"-----\n")
	dataStr = strings.ReplaceAll(dataStr, "\n", " ")
	// utilLogger.Info("trim", "result", dataStr)
	return []byte(dataStr)
}
