package config

const (
	// ConfigImageRegistry is the key to get image.registry config
	ConfigImageRegistry = "image.registry"
	// ConfigImageRegistryPullRequest is the key to get image.registry_pull_request config
	ConfigImageRegistryPullRequest = "image.registry_pull_request"
	ConfigTokenServiceAddr         = "token.url"
	ConfigTokenServiceInsecure     = "token.insecure"
	ConfigTokenServiceDebug        = "token.debug"
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

	// ConfigRegistryCPU is the key to get registry.cpu config
	ConfigRegistryCPU = "registry.cpu"
	// ConfigRegistryMemory is the key to get registry.memory config
	ConfigRegistryMemory = "registry.memory"
	// ConfigNotaryServerCPU is the key to get notary.server.cpu config
	ConfigNotaryServerCPU = "notary.server.cpu"
	// ConfigNotaryServerMemory is the key to get notary.server.memory config
	ConfigNotaryServerMemory = "notary.server.memory"
	// ConfigNotarySignerCPU is the key to get notary.signer.cpu config
	ConfigNotarySignerCPU = "notary.signer.cpu"
	// ConfigNotarySignerMemory is the key to get notary.signer.memory config
	ConfigNotarySignerMemory = "notary.signer.memory"
	// ConfigNotaryDBCPU is the key to get notary.db.cpu config
	ConfigNotaryDBCPU = "notary.db.cpu"
	// ConfigNotaryDBMemory is the key to get notary.db.memory config
	ConfigNotaryDBMemory = "notary.db.memory"
)
