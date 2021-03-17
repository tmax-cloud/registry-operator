/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1

import (
	"github.com/operator-framework/operator-lib/status"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ImageReplicateStatusType is status type of external registry
type ImageReplicateStatusType string

const (
	// ImageReplicateSuccess is a status that replicating image is finished successfully
	ImageReplicateSuccess ImageReplicateStatusType = "Success"
	// ImageReplicateFail is a failed status while copying image
	ImageReplicateFail ImageReplicateStatusType = "Fail"
	// ImageReplicatePending is an initial status
	ImageReplicatePending ImageReplicateStatusType = "Pending"
	// ImageReplicateProcessing is status that replicating is started
	ImageReplicateProcessing ImageReplicateStatusType = "Processing"
)

// ImageReplicateSpec defines the desired state of ImageReplicate
type ImageReplicateSpec struct {
	// Source image information
	FromImage ImageInfo `json:"fromImage"`
	// Destination image information
	ToImage ImageInfo `json:"toImage"`
	// The name of the signer to sign the image you moved. This field is available only if destination registry's `RegistryType` is `HpcdRegistry`
	Signer string `json:"signer,omitempty"`
}

// ImageInfo consists of registry information and image information.
type ImageInfo struct {
	// +kubebuilder:validation:Enum=HpcdRegistry;DockerHub;Docker;HarborV2
	// Registry type like HarborV2
	RegistryType RegistryType `json:"registryType"`
	// metadata name of external registry or hpcd registry
	RegistryName string `json:"registryName"`
	// metadata namespace of external registry or hpcd registry
	RegistryNamespace string `json:"registryNamespace"`
	// Image path (example: library/alpine:3)
	Image string `json:"image"`
	// Certificate secret name for private registry. Secret's data key must be 'ca.crt' or 'tls.crt'.
	CertificateSecret string `json:"certificateSecret,omitempty"`
	// Login id and password secret object for registry
	ImagePullSecret string `json:"imagePullSecret,omitempty"`
}

// ImageReplicateStatus defines the observed state of ImageReplicate
type ImageReplicateStatus struct {
	// Conditions are status of subresources
	Conditions status.Conditions `json:"conditions,omitempty"`
	// ImageSignRequestName is ImageSignRequest's name if exists
	ImageSignRequestName string `json:"imageSignRequestName,omitempty"`
	// State is a status of external registry
	State ImageReplicateStatusType `json:"state,omitempty"`
	// StateChangedAt is the time when state was changed
	StateChangedAt metav1.Time `json:"stateChangedAt,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=imgrepl
// +kubebuilder:printcolumn:name="AGE",type=date,JSONPath=`.metadata.creationTimestamp`

// ImageReplicate is the Schema for the imagereplicates API
type ImageReplicate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ImageReplicateSpec   `json:"spec,omitempty"`
	Status ImageReplicateStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ImageReplicateList contains a list of ImageReplicate
type ImageReplicateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ImageReplicate `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ImageReplicate{}, &ImageReplicateList{})
}
