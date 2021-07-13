package v1

const (
	// PodNotFound is an error that pod is not found
	PodNotFound = "PodNotFound"
	// ContainerNotFound is an error that container is not found
	ContainerNotFound = "ContainerNotFound"
	// ContainerStatusIsNil is an error that container status is nil
	ContainerStatusIsNil = "ContainerStatusIsNil"
	// PodNotRunning is an error that pod is not running
	PodNotRunning = "PodNotRunning"
	// PvcVolumeMountNotFound is an error that PVC volume mount is not found in pod
	PvcVolumeMountNotFound = "PvcVolumeMountNotFound"
	// PvcVolumeNotFound is an error that volume is not found in pod
	PvcVolumeNotFound = "PvcVolumeNotFound"
)

// RegistryErrors represents error of registry subresource
type RegistryErrors struct {
	errorType    *string
	errorMessage *string
}

func (r RegistryErrors) Error() string {
	if r.errorType != nil {
		return *r.errorType
	}

	return *r.errorMessage
}

// MakeRegistryError sets error of registry subresource
func MakeRegistryError(e string) error {
	RegistryError := RegistryErrors{}
	if e == PodNotFound || e == ContainerNotFound || e == ContainerStatusIsNil || e == PodNotRunning || e == PvcVolumeMountNotFound {
		RegistryError.errorType = &e
	} else {
		RegistryError.errorMessage = &e
	}
	return RegistryError
}
