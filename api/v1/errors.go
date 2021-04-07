package v1

import (
	"errors"
	"fmt"
)

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

// IsPodError returns true if the specified error was created by PodNotFound, ContainerStatusIsNil, or PodNotRunning.
func IsPodError(err error) bool {
	if err.Error() == PodNotFound || err.Error() == ContainerStatusIsNil || err.Error() == PodNotRunning {
		return true
	}

	return false
}

func AppendError(err error, appendMessage string) error {
	if err == nil || err.Error() == "" {
		return errors.New(appendMessage)
	}

	return fmt.Errorf("%s, %s", err.Error(), appendMessage)
}
