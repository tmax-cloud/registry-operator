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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ImageSignerSpec defines the desired state of ImageSigner
type ImageSignerSpec struct {
	// ImageSigner's email
	Email string `json:"email,omitempty"`
	// ImageSigner's phone number
	Phone string `json:"phone,omitempty"`
	// ImageSigner's team
	Team string `json:"team,omitempty"`
	// Additional information of ImageSigner
	Description string `json:"description,omitempty"`
	// Don't deal with this field. If Owner field is set or manipulated, could not be recovered.
	Owner string `json:"owner,omitempty"`
}

// ImageSignerStatus defines the observed state of ImageSigner
type ImageSignerStatus struct {
	*SignerKeyState `json:"signerKeyState,omitempty"`
}

// SignerKeyState is ehe status information about whether signer key is created
type SignerKeyState struct {
	// Whether SignerKey is created
	Created bool `json:"created,omitempty"`
	// Reason failed to create SignerKey
	Reason string `json:"reason,omitempty"`
	// Message failed to create SignerKey
	Message string `json:"message,omitempty"`
	// SignerKey's root key ID
	RootKeyID string `json:"rootKeyId,omitempty"`
	// Created time
	CreatedAt *metav1.Time `json:"createdAt,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,shortName=is
// +kubebuilder:printcolumn:name="SIGNER_KEY_CREATED",type=string,JSONPath=`.status.signerKeyState.created`

// ImageSigner is the Schema for the imagesigners API
type ImageSigner struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ImageSignerSpec   `json:"spec,omitempty"`
	Status ImageSignerStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ImageSignerList contains a list of ImageSigner
type ImageSignerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ImageSigner `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ImageSigner{}, &ImageSignerList{})
}
