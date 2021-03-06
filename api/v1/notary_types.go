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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// NotarySpec defines the desired state of Notary
type NotarySpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// Root CA certificate secret for notary
	RootCASecret string `json:"rootCASecret"`
	// Settings for registry authentication config
	AuthConfig AuthConfig `json:"authConfig"`
	// Service type to expose notary
	// +kubebuilder:validation:Enum=Ingress;LoadBalancer
	ServiceType           NotaryServiceType `json:"serviceType"`
	PersistentVolumeClaim NotaryPVC         `json:"persistentVolumeClaim"`

	// Settings for notary server
	Server NotaryServer `json:"server,omitempty"`
	// Settings for notary signer
	Signer NotarySigner `json:"signer,omitempty"`
	// Settings for notary database
	DB NotaryDB `json:"db,omitempty"`
}

type NotaryServiceType string

const (
	NotaryServiceTypeIngress      = NotaryServiceType("Ingress")
	NotaryServiceTypeLoadBalancer = NotaryServiceType("LoadBalancer")
)

type NotaryPVC struct {
	// Use exist pvc
	// +kubebuilder:validation:OneOf
	Exist *ExistPvc `json:"exist,omitempty"` // [TODO] One Of

	// Create new pvc
	// +kubebuilder:validation:OneOf
	Create *CreatePvc `json:"create,omitempty"` // [TODO] One Of
}

type AuthConfig struct {
	Realm   string `json:"realm"`
	Service string `json:"service"`
	Issuer  string `json:"issuer"`
}

type NotaryServer struct {
	// resource requirements
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`
}
type NotarySigner struct {
	// resource requirements
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`
}
type NotaryDB struct {
	// resource requirements
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`
}

// NotaryStatus defines the observed state of Notary
type NotaryStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Conditions           status.Conditions `json:"conditions,omitempty"`
	ServerClusterIP      string            `json:"serverClusterIP,omitempty"`
	ServerLoadBalancerIP string            `json:"serverLoadBalancerIP,omitempty"`
	SignerClusterIP      string            `json:"signerClusterIP,omitempty"`
	SignerLoadBalancerIP string            `json:"signerLoadBalancerIP,omitempty"`
	NotaryURL            string            `json:"notaryURL,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=notaries,scope=Namespaced,shortName=not
// +kubebuilder:printcolumn:name="AGE",type=date,JSONPath=`.metadata.creationTimestamp`

// Notary is the Schema for the notaries API
type Notary struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NotarySpec   `json:"spec,omitempty"`
	Status NotaryStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// NotaryList contains a list of Notary
type NotaryList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Notary `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Notary{}, &NotaryList{})
}
