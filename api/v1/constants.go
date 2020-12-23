package v1

const (
	K8sPrefix         = "hpcd-"
	OperatorNamespace = "registry-system"
	TLSPrefix         = "tls-"
	K8sRegistryPrefix = "registry-"
	K8sNotaryPrefix   = "notary-"

	CustomObjectGroup = "tmax.io"

	// OpenSSL Cert File Name
	RegistryRootCASecretName = "registry-ca"
	GenCertScriptFile        = "genCert.sh"
	CertKeyFile              = "localhub.key"
	CertCrtFile              = "localhub.crt"
	CertCertFile             = "localhub.cert"
	DockerDir                = "/etc/docker"
	DockerCertDir            = "/etc/docker/certs.d"

	// OpenSSL Certificate Home Directory
	OpenSslHomeDir = "/openssl"

	DockerLoginHomeDir   = "/root/.docker"
	DockerConfigFile     = "config.json"
	DockerConfigJsonFile = ".dockerconfigjson"
)
