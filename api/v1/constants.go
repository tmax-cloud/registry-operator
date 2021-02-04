package v1

const (
	// K8sPrefix is hypercloud prefix
	K8sPrefix = "hpcd-"
	// OperatorNamespace is default operator namespace
	OperatorNamespace = "registry-system"
	// TLSPrefix is TLS secret prefix
	TLSPrefix = "tls-"
	// K8sRegistryPrefix is registry's image pull secret resource prefix
	K8sRegistryPrefix = "registry-"
	// K8sNotaryPrefix is notary resource prefix
	K8sNotaryPrefix = "notary-"
	// K8sKeycloakPrefix is keycloak resource prefix
	K8sKeycloakPrefix = "keycloak-"
	// CustomObjectGroup is custom resource group
	CustomObjectGroup = "tmax.io"

	// RegistryRootCASecretName is OpenSSL Cert File Name
	RegistryRootCASecretName = "registry-ca"
	// KeycloakCASecretName is keycloak cert secret name
	KeycloakCASecretName = "keycloak-cert"
)
