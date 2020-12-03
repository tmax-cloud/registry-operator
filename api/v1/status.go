package v1

import "github.com/operator-framework/operator-lib/status"

type Status string

const (
	StatusSucceeded = Status("Succeeded")
	StatusFailed    = Status("Failed")
	StatusReady     = Status("Ready")
	StatusNotReady  = Status("NotReady")
	StatusRunning   = Status("Running")
	StatusPending   = Status("Pending")
	StatusSkipped   = Status("Skipped")
	StatusCreating  = Status("Creating")

	// Registry conditions
	ConditionTypeDeployment             = status.ConditionType("DeploymentExist")
	ConditionTypePod                    = status.ConditionType("PodRunning")
	ConditionTypeContainer              = status.ConditionType("ContainerReady")
	ConditionTypeService                = status.ConditionType("ServiceExist")
	ConditionTypeSecretOpaque           = status.ConditionType("SecretOpaqueExist")
	ConditionTypeSecretDockerConfigJson = status.ConditionType("SecretDockerConfigJsonExist")
	ConditionTypeSecretTls              = status.ConditionType("SecretTlsExist")
	ConditionTypeIngress                = status.ConditionType("IngressExist")
	ConditionTypePvc                    = status.ConditionType("PvcExist")
	ConditionTypeConfigMap              = status.ConditionType("ConfigMapExist")
	ConditionKeycloakRealm              = status.ConditionType("KeycloakRealm")

	// Notary conditions
	ConditionTypeNotaryDBPod         = status.ConditionType("NotaryDBPodExist")
	ConditionTypeNotaryDBPVC         = status.ConditionType("NotaryDBPVCExist")
	ConditionTypeNotaryDBService     = status.ConditionType("NotaryDBServiceExist")
	ConditionTypeNotaryServerIngress = status.ConditionType("NotaryServerIngressExist")
	ConditionTypeNotaryServerPod     = status.ConditionType("NotaryServerPodExist")
	ConditionTypeNotaryServerSecret  = status.ConditionType("NotaryServerSecretExist")
	ConditionTypeNotaryServerService = status.ConditionType("NotaryServerServiceExist")
	ConditionTypeNotarySignerPod     = status.ConditionType("NotarySignerPodExist")
	ConditionTypeNotarySignerSecret  = status.ConditionType("NotarySignerSecretExist")
	ConditionTypeNotarySignerService = status.ConditionType("NotarySignerServiceExist")
)
