package v1

const AccessModeDefault = "ReadWriteMany"

type ExistPvc struct {
	PvcName string `json:"pvcName"`
}

type CreatePvc struct {
	// +kubebuilder:validation:Enum=ReadWriteOnce;ReadWriteMany
	AccessModes []string `json:"accessModes"`

	// enter the desired storage size (ex: 10Gi)
	StorageSize string `json:"storageSize"`

	StorageClassName string `json:"storageClassName"`

	// +kubebuilder:validation:Enum=Filesystem;Block
	VolumeMode string `json:"volumeMode,omitempty"`

	// Delete the pvc as well when this registry is deleted
	DeleteWithPvc bool `json:"deleteWithPvc,omitempty"`
}
