package v1

import "github.com/operator-framework/operator-lib/status"

const (
	/* Registry conditions */

	// ConditionTypeDeployment is a condition that deployment exists
	ConditionTypeDeployment = status.ConditionType("DeploymentExist")
	// ConditionTypePod is a condition that pod is running
	ConditionTypePod = status.ConditionType("PodRunning")
	// ConditionTypeContainer is a condition that container is ready
	ConditionTypeContainer = status.ConditionType("ContainerReady")
	// ConditionTypeService is a condition that service exists
	ConditionTypeService = status.ConditionType("ServiceExist")
	// ConditionTypeSecretOpaque is a condition that opaque secret exists
	ConditionTypeSecretOpaque = status.ConditionType("SecretOpaqueExist")
	// ConditionTypeSecretDockerConfigJSON is a condition that docker config json secret exists
	ConditionTypeSecretDockerConfigJSON = status.ConditionType("SecretDockerConfigJsonExist")
	// ConditionTypeSecretTLS is a condition that tls secret exists
	ConditionTypeSecretTLS = status.ConditionType("SecretTlsExist")
	// ConditionTypeIngress is a condition that ingress exists
	ConditionTypeIngress = status.ConditionType("IngressExist")
	// ConditionTypePvc is a condition that PVC exists
	ConditionTypePvc = status.ConditionType("PvcExist")
	// ConditionTypeConfigMap is a condition that confimap exists
	ConditionTypeConfigMap = status.ConditionType("ConfigMapExist")
	// ConditionTypeKeycloakRealm is a condition that keycloak realm exists
	ConditionTypeKeycloakRealm = status.ConditionType("KeycloakRealmExist")
	// ConditionTypeNotary is a condition that notary exists
	ConditionTypeNotary = status.ConditionType("NotaryExist")

	/* Notary conditions */

	// ConditionTypeNotaryDBPod is a condition that notary DB pod exists
	ConditionTypeNotaryDBPod = status.ConditionType("NotaryDBPodExist")
	// ConditionTypeNotaryDBPVC is a condition that notary DB PVC exists
	ConditionTypeNotaryDBPVC = status.ConditionType("NotaryDBPVCExist")
	// ConditionTypeNotaryDBService is a condition that notary DB service exists
	ConditionTypeNotaryDBService = status.ConditionType("NotaryDBServiceExist")
	// ConditionTypeNotaryServerIngress is a condition that notary server ingress exists
	ConditionTypeNotaryServerIngress = status.ConditionType("NotaryServerIngressExist")
	// ConditionTypeNotaryServerPod is a condition that notary server pod exists
	ConditionTypeNotaryServerPod = status.ConditionType("NotaryServerPodExist")
	// ConditionTypeNotaryServerSecret is a condition that notary server secret exists
	ConditionTypeNotaryServerSecret = status.ConditionType("NotaryServerSecretExist")
	// ConditionTypeNotaryServerService is a condition that notary server service exists
	ConditionTypeNotaryServerService = status.ConditionType("NotaryServerServiceExist")
	// ConditionTypeNotarySignerPod is a condition that notary signer pod exists
	ConditionTypeNotarySignerPod = status.ConditionType("NotarySignerPodExist")
	// ConditionTypeNotarySignerSecret is a condition that notary signer secret exists
	ConditionTypeNotarySignerSecret = status.ConditionType("NotarySignerSecretExist")
	// ConditionTypeNotarySignerService is a condition that notary signer service exists
	ConditionTypeNotarySignerService = status.ConditionType("NotarySignerServiceExist")

	/* ExternalRegistry conditions */

	// ConditionTypeExRegistryInitialized is a condition that external registry is initialized
	ConditionTypeExRegistryInitialized = status.ConditionType("Initialized")
	// ConditionTypeExRegistryCronJobExist is a condition that cronjob exists
	ConditionTypeExRegistryCronJobExist = status.ConditionType("CronJobExist")

	/* ImageReplicate conditions */

	// ConditionTypeImageReplicateRegistryJobExist is a condition that registry job to replicate image exists
	ConditionTypeImageReplicateRegistryJobExist = status.ConditionType("ImageReplicate")
	// ConditionTypeImageReplicateRegistryJobSuccess is a condition that registry job to replicate image is succeeded
	ConditionTypeImageReplicateRegistryJobProcessing = status.ConditionType("ImageReplicateProcessing")
	// ConditionTypeImageReplicateRegistryJobSuccess is a condition that registry job to replicate image is succeeded
	ConditionTypeImageReplicateRegistryJobSuccess = status.ConditionType("ImageReplicateSuccess")
)
