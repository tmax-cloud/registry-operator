package config

const (
	// ConfigImageRegistry is the key to get image.registry config
	ConfigImageRegistry = "image.registry"
	// ConfigImageRegistryPullRequest is the key to get image.registry_pull_request config
	ConfigImageRegistryPullRequest = "image.registry_pull_request"
	// ConfigKeycloakService is the key to get keycloak.service config
	ConfigKeycloakService = "keycloak.service"
	// ConfigClusterName is the key to get cluster.name config
	ConfigClusterName = "cluster.name"
	// ConfigImageScanSvr is the key to get clair.url config
	ConfigImageScanSvr = "scanning.scanner.url"
	// ConfigImageReportSvr is the key to get elastic_search.url config
	ConfigImageReportSvr = "scanning.report.url"
	// ConfigHarborNamespace is the key to get harbor.namespace config
	ConfigHarborNamespace = "harbor.namespace"
	// ConfigHarborCoreIngress is the key to get harbor.core.ingress config
	ConfigHarborCoreIngress = "harbor.core.ingress"
	// ConfigHarborNotaryIngress is the key to get harbor.notary.ingress config
	ConfigHarborNotaryIngress = "harbor.notary.ingress"
	// ConfigRegistryImage is the key to get registry.image config
	ConfigRegistryImage = "registry.image"
	// ConfigNotaryServerImage is the key to get notary.server.image config
	ConfigNotaryServerImage = "notary.server.image"
	// ConfigNotarySignerImage is the key to get notary.signer.image config
	ConfigNotarySignerImage = "notary.signer.image"
	// ConfigNotaryDBImage is the key to get notary.db.image config
	ConfigNotaryDBImage = "notary.db.image"
	// ConfigRegistryImagePullSecret is the key to get registry.image_pull_secret config
	ConfigRegistryImagePullSecret = "registry.image_pull_secret"
	// ConfigNotaryServerImagePullSecret is the key to get notary.server.image_pull_secret config
	ConfigNotaryServerImagePullSecret = "notary.server.image_pull_secret"
	// ConfigNotarySignerImagePullSecret is the key to get notary.signer.image_pull_secret config
	ConfigNotarySignerImagePullSecret = "notary.signer.image_pull_secret"
	// ConfigNotaryDBImagePullSecret is the key to get notary.db.image_pull_secret config
	ConfigNotaryDBImagePullSecret = "notary.db.image_pull_secret"
	// ConfigExternalRegistrySyncPeriod is the key to get external_registry.sync_period config
	ConfigExternalRegistrySyncPeriod = "external_registry.sync_period"
)
