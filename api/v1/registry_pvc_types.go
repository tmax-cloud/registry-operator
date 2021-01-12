package v1

const AccessModeDefault = "ReadWriteMany"

type ExistPvc struct {
	// PVC's name you have created
	PvcName string `json:"pvcName"`
}

type CreatePvc struct {
	// AccessModes is . Recommend value is "ReadWriteMany"
	// You can choose ReadWriteOnce, ReadOnlyMany and ReadWriteMany
	AccessModes []AccessMode `json:"accessModes"`

	// Enter the desired storage size like "10Gi"
	StorageSize string `json:"storageSize"`

	// Enter StorageClassName like "csi-cephfs-sc"
	StorageClassName string `json:"storageClassName"`

	// Delete the pvc as well when this registry is deleted (default: true)
	DeleteWithPvc bool `json:"deleteWithPvc,omitempty"`
}

// +kubebuilder:validation:Enum=ReadWriteOnce;ReadWriteMany
type AccessMode string
