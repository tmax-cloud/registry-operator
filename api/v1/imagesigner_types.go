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

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ImageSignerSpec defines the desired state of ImageSigner
type ImageSignerSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Name        string `json:"name,omitempty"`
	Email       string `json:"email,omitempty"`
	Phone       string `json:"phone,omitempty"`
	Team        string `json:"team,omitempty"`
	Description string `json:"description,omitempty"`
	Owner       string `json:"owner,omitempty"`
}

// ImageSignerStatus defines the observed state of ImageSigner
type ImageSignerStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	*SignerKeyState `json:"signerKeyState,omitempty"`
}

type SignerKeyState struct {
	Created   bool         `json:"created,omitempty"`
	Reason    string       `json:"reason,omitempty"`
	Message   string       `json:"message,omitempty"`
	RootKeyID string       `json:"rootKeyId,omitempty"`
	CreatedAt *metav1.Time `json:"createdAt,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,shortName=is

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
