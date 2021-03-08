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

const (
	// RegistryTypeHarborV2 is harbor v2 registry type
	RegistryTypeHarborV2 RegistryType = "HarborV2"
	// RegistryTypeDockerHub is docker hub registry type
	RegistryTypeDockerHub RegistryType = "DockerHub"
	// RegistryTypeDocker is docker registry type
	RegistryTypeDocker RegistryType = "Docker"
)

// RegistryType is a type of external registry
type RegistryType string

// ExternalRegistryStatusType is status type of external registry
type ExternalRegistryStatusType string

const (
	// ExternalRegistryPending is
	ExternalRegistryPending ExternalRegistryStatusType = "Pending"
	// ExternalRegistryReady is
	ExternalRegistryReady ExternalRegistryStatusType = "Ready"
	// ExternalRegistryNotReady is
	ExternalRegistryNotReady ExternalRegistryStatusType = "NotReady"
)

// ExternalRegistrySpec defines the desired state of ExternalRegistry
type ExternalRegistrySpec struct {
	// +kubebuilder:validation:Enum=HarborV2;DockerHub;Docker
	// Registry type like HarborV2
	RegistryType RegistryType `json:"registryType"`
	// Registry URL (example: https://192.168.6.100:5000)
	// If ReigstryType is DockerHub, this value must be "https://registry-1.docker.io"
	RegistryURL string `json:"registryUrl"`
	// Certificate secret name for private registry. Secret's data key must be 'ca.crt' or 'tls.crt'.
	CertificateSecret string `json:"certificateSecret,omitempty"`
	// Do not verify tls certificates
	Insecure bool `json:"insecure,omitempty"`
	// Login ID for registry
	LoginID string `json:"loginId,omitempty"`
	// Login password for registry
	LoginPassword string `json:"loginPassword,omitempty"`
	// Schedule is a cron spec for periodic sync
	// If you want to synchronize repository every 5 minute, enter "*/5 * * * *".
	// Cron spec ref: https://ko.wikipedia.org/wiki/Cron
	Schedule string `json:"schedule,omitempty"`
}

// ExternalRegistryStatus defines the observed state of ExternalRegistry
type ExternalRegistryStatus struct {
	// Login id and password secret object for registry
	LoginSecret string `json:"loginSecret,omitempty"`
	// Conditions are status of subresources
	Conditions status.Conditions `json:"conditions,omitempty"`
	// State is a status of external registry
	State ExternalRegistryStatusType `json:"state,omitempty"`
	// StateChangedAt is the time when state was changed
	StateChangedAt metav1.Time `json:"stateChangedAt,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=externalregistries,scope=Namespaced,shortName=exreg
// +kubebuilder:printcolumn:name="REGISTRY_URL",type=string,JSONPath=`.spec.registryUrl`
// +kubebuilder:printcolumn:name="REGISTRY_TYPE",type=string,JSONPath=`.spec.registryType`
// +kubebuilder:printcolumn:name="STATUS",type=string,JSONPath=`.status.state`
// +kubebuilder:printcolumn:name="AGE",type=date,JSONPath=`.metadata.creationTimestamp`

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
