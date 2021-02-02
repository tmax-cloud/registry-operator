package v1

const AccessModeDefault = "ReadWriteMany"

type ExistPvc struct {
	// PVC's name you have created
	PvcName string `json:"pvcName"`
}

type CreatePvc struct {
	// Each PV's access modes are set to the specific modes supported by that particular volume.
	// Ref: https://kubernetes.io/docs/concepts/storage/persistent-volumes/#access-modes
	// You can choose ReadWriteOnce, ReadOnlyMany and ReadWriteMany
	AccessModes []AccessMode `json:"accessModes"`

	// Desired storage size like "10Gi"
	StorageSize string `json:"storageSize"`

	// StorageClassName like "csi-cephfs-sc"
	StorageClassName string `json:"storageClassName"`

	// Delete the pvc as well when this registry is deleted (default: false)
	DeleteWithPvc bool `json:"deleteWithPvc,omitempty"`
}

// +kubebuilder:validation:Enum=ReadWriteOnce;ReadWriteMany
type AccessMode string
