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

const (
	RegistryTypeHarborV2 RegistryType = "HarborV2"
	RegistryTypeDocker   RegistryType = "Docker"
)

type RegistryType string

// ExternalRegistrySpec defines the desired state of ExternalRegistry
type ExternalRegistrySpec struct {
	// +kubebuilder:validation:Enum=HarborV2
	// Registry type like Harbor
	RegistryType RegistryType `json:"registryType"`
	// Registry URL (example: docker.io)
	RegistryURL string `json:"registryUrl"`
	// Certificate secret name for private registry. Secret's data key must be 'ca.crt' or 'tls.crt'.
	CertificateSecret string `json:"certificateSecret,omitempty"`
	// Do not verify tls certificates
	Insecure bool `json:"insecure,omitempty"`
	// Login id and password secret object for registry
	ImagePullSecret string `json:"imagePullSecret"`
	// Sync period
	SyncPeriod int `json:"syncPeriod,omitempty"`
}

// ExternalRegistryStatus defines the observed state of ExternalRegistry
type ExternalRegistryStatus struct {
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=externalregistries,scope=Namespaced,shortName=exreg
// +kubebuilder:printcolumn:name="REGISTRY_URL",type=string,JSONPath=`.spec.registryUrl`
// +kubebuilder:printcolumn:name="IMAGE_PULL_SECRET",type=string,JSONPath=`.spec.imagePullSecret`

// ExternalRegistry is the Schema for the externalregistries API
type ExternalRegistry struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ExternalRegistrySpec   `json:"spec,omitempty"`
	Status ExternalRegistryStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ExternalRegistryList contains a list of ExternalRegistry
type ExternalRegistryList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ExternalRegistry `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ExternalRegistry{}, &ExternalRegistryList{})
}
