package schemes

import (
	"crypto/x509/pkix"
	"fmt"
	"net"

	regv1 "github.com/tmax-cloud/registry-operator/api/v1"
	"github.com/tmax-cloud/registry-operator/internal/utils"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	TLSCert = "tls.crt"
	TLSKey  = "tls.key"
)

// Certificate is to compose x509 certificate's template
type Certificate interface {
	GetSanIP(interface{}) []net.IP
	GetSanDNS(interface{}) []string
	GetSubject() *pkix.Name
}

// CertFactory is to make CertPair
type CertFactory struct {
	client client.Client
}

// CertType is a certificate pair type to create
type CertType int

const (
	certTypeNotaryServer CertType = iota
	certTypeNotarySigner
	certTypeRegistry
)

// NewCertFactory is ...
func NewCertFactory(c client.Client) *CertFactory {
	return &CertFactory{
		client: c,
	}
}

// CreateCertPair is to create new CertPair of the types you want
func (f *CertFactory) CreateCertPair(source interface{}, certType CertType) (*utils.CertPair, error) {
	var out *utils.CertPair
	ct := &CertTemplate{}

	switch certType {
	case certTypeNotaryServer:
		pair, err := ct.CreateCertPair(&NotaryServerCert{}, source)
		if err != nil {
			return nil, err
		}
		out = pair

	case certTypeNotarySigner:
		pair, err := ct.CreateCertPair(&NotarySignerCert{}, source)
		if err != nil {
			return nil, err
		}
		out = pair

	case certTypeRegistry:
		pair, err := ct.CreateCertPair(&RegistryCert{}, source)
		if err != nil {
			return nil, err
		}
		out = pair

	default:
		return nil, fmt.Errorf("not supported cert type")
	}

	f.setParent(out)
	if out.ParentCert == nil || out.ParentKey == nil {
		return nil, fmt.Errorf("failed to get parent cert")
	}
	if err := f.createCertificateData(out); err != nil {
		return nil, err
	}

	return out, nil
}

func (f *CertFactory) setParent(cert *utils.CertPair) {
	parentCert, parentPrivKey := getRootCACertificate(f.client)
	cert.SetParent(parentCert, parentPrivKey)
}

func (f *CertFactory) createCertificateData(cert *utils.CertPair) error {
	if err := cert.CreateCertificateData(); err != nil {
		return err
	}

	return nil
}

// CertTemplate is CertPair's template
type CertTemplate struct{}

func (t *CertTemplate) CreateCertPair(cert Certificate, source interface{}) (*utils.CertPair, error) {
	out, err := utils.NewCertPair(nil, false)
	if err != nil {
		return nil, err
	}

	out.SetSubject(cert.GetSubject())
	out.SetSubjectAltName(cert.GetSanIP(source), cert.GetSanDNS(source))

	return out, nil
}

// NotaryServerCert is Notary Server's Certificate
type NotaryServerCert struct{}

func (n *NotaryServerCert) GetSanIP(notary interface{}) []net.IP {
	nt := notary.(*regv1.Notary)

	ips := []net.IP{}
	if len(nt.Status.ServerClusterIP) > 0 {
		ips = append(ips, net.ParseIP(nt.Status.ServerClusterIP))
	}
	if len(nt.Status.ServerLoadBalancerIP) > 0 {
		ips = append(ips, net.ParseIP(nt.Status.ServerLoadBalancerIP))
	}

	return ips
}

func (n *NotaryServerCert) GetSanDNS(notary interface{}) []string {
	nt := notary.(*regv1.Notary)

	domains := []string{}
	if nt.Spec.ServiceType == "Ingress" {
		domains = append(domains, NotaryDomainName(nt))
	}
	domains = append(domains, utils.BuildServiceHostname(SubresourceName(nt, SubTypeNotaryServerService), nt.Namespace))

	return domains
}

func (n *NotaryServerCert) GetSubject() *pkix.Name {
	subject := &pkix.Name{
		Country:       []string{"KR"},
		Organization:  []string{"tmax"},
		StreetAddress: []string{"Seoul"},
		CommonName:    "notary-server.tmax.co.kr",
	}

	return subject
}

// NotarySignerCert is Notary Signer's Certificate
type NotarySignerCert struct{}

func (n *NotarySignerCert) GetSanIP(notary interface{}) []net.IP {
	nt := notary.(*regv1.Notary)

	ips := []net.IP{}
	if len(nt.Status.SignerClusterIP) > 0 {
		ips = append(ips, net.ParseIP(nt.Status.SignerClusterIP))
	}
	if len(nt.Status.SignerLoadBalancerIP) > 0 {
		ips = append(ips, net.ParseIP(nt.Status.SignerLoadBalancerIP))
	}

	return ips
}

func (n *NotarySignerCert) GetSanDNS(notary interface{}) []string {
	nt := notary.(*regv1.Notary)

	domains := []string{}
	domains = append(domains, utils.BuildServiceHostname(SubresourceName(nt, SubTypeNotarySignerService), nt.Namespace))

	return domains
}

func (n *NotarySignerCert) GetSubject() *pkix.Name {
	subject := &pkix.Name{
		Country:       []string{"KR"},
		Organization:  []string{"tmax"},
		StreetAddress: []string{"Seoul"},
		CommonName:    "notary-signer.tmax.co.kr",
	}

	return subject
}

// RegistryCert is Registry's Certificate
type RegistryCert struct{}

func (c *RegistryCert) GetSanIP(registry interface{}) []net.IP {
	reg := registry.(*regv1.Registry)

	ips := []net.IP{}
	if len(reg.Status.ClusterIP) > 0 {
		ips = append(ips, net.ParseIP(reg.Status.ClusterIP))
	}
	if len(reg.Status.LoadBalancerIP) > 0 {
		ips = append(ips, net.ParseIP(reg.Status.LoadBalancerIP))
	}

	return ips
}

func (c *RegistryCert) GetSanDNS(registry interface{}) []string {
	reg := registry.(*regv1.Registry)

	domains := []string{}
	if reg.Spec.RegistryService.ServiceType == "Ingress" {
		domains = append(domains, RegistryDomainName(reg))
	}
	domains = append(domains, utils.BuildServiceHostname(SubresourceName(reg, SubTypeNotarySignerService), reg.Namespace))

	return domains
}

func (c *RegistryCert) GetSubject() *pkix.Name {
	subject := &pkix.Name{
		Country:       []string{"KR"},
		Organization:  []string{"tmax"},
		StreetAddress: []string{"Seoul"},
		CommonName:    "registry.tmax.co.kr",
	}

	return subject
}
