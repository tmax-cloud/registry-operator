package v1

import (
	"github.com/operator-framework/operator-lib/status"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	RegistryLoginUrl = CustomObjectGroup + "/registry-login-url"
	RegistryKind     = "Registry"
)

// RegistrySpec defines the desired state of Registry
type RegistrySpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html

	// Registry's image name
	Image string `json:"image,omitempty"`
	// Description for registry
	Description string `json:"description,omitempty"`
	// Login ID for registry
	LoginId string `json:"loginId"`
	// Login password for registry
	LoginPassword string `json:"loginPassword"`
	// Settings for notary service
	Notary RegistryNotary `json:"notary,omitempty"`
	// The name of the configmap where the registry config.yml content
	CustomConfigYml string `json:"customConfigYml,omitempty"`

	// Settings for registry's deployemnt
	RegistryDeployment RegistryDeployment `json:"registryDeployment,omitempty"`
	// Service type to expose registry
	RegistryService RegistryService `json:"service"`
	// Settings for registry pvc
	PersistentVolumeClaim RegistryPVC `json:"persistentVolumeClaim"`
}

type RegistryNotary struct {
	// Activate notary service to sign images
	Enabled bool `json:"enabled"`
	// Use Ingress or LoadBalancer
	// +kubebuilder:validation:Enum=Ingress;LoadBalancer
	ServiceType NotaryServiceType `json:"serviceType"`
	// Settings for notary pvc
	PersistentVolumeClaim NotaryPVC `json:"persistentVolumeClaim"`
}

type RegistryDeployment struct {
	// Deployment's label
	Labels map[string]string `json:"labels,omitempty"`
	// Registry pod's node selector
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`
	// Deployment's label selector
	Selector metav1.LabelSelector `json:"selector,omitempty"`
	// Deployment's toleration configuration
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`
}

type RegistryServiceType string

const (
	RegServiceTypeLoadBalancer = "LoadBalancer"
	RegServiceTypeIngress      = "ClusterIP"
)

type RegistryService struct {
	// Use Ingress or LoadBalancer
	// +kubebuilder:validation:Enum=Ingress;LoadBalancer
	ServiceType RegistryServiceType `json:"serviceType"`
	// use ingress service type
	// (Deprecated)
	// Ingress Ingress `json:"ingress,omitempty"`

	// (Deprecated)
	// LoadBalancer LoadBalancer `json:"loadBalancer,omitempty"`
}

type RegistryPVC struct {
	// Registry's pvc mount path (default: /var/lib/registry)
	MountPath string `json:"mountPath,omitempty"`

	// +kubebuilder:validation:OneOf
	Exist *ExistPvc `json:"exist,omitempty"` // [TODO] One Of

	// +kubebuilder:validation:OneOf
	Create *CreatePvc `json:"create,omitempty"` // [TODO] One Of
}

// RegistryStatus defines the observed state of Registry
type RegistryStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html

	Conditions          status.Conditions `json:"conditions,omitempty"`
	Phase               string            `json:"phase,omitempty"`
	Message             string            `json:"message,omitempty"`
	Reason              string            `json:"reason,omitempty"`
	PhaseChangedAt      metav1.Time       `json:"phaseChangedAt,omitempty"`
	Capacity            string            `json:"capacity,omitempty"`
	ClusterIP           string            `json:"clusterIP,omitempty"`
	LoadBalancerIP      string            `json:"loadBalancerIP,omitempty"`
	PodRecreateRequired bool              `json:"podRecreateRequired,omitempty"`
	ServerURL           string            `json:"serverURL,omitempty"`
	NotaryURL           string            `json:"notaryURL,omitempty"`
}

// +kubebuilder:object:root=true

// Registry is the Schema for the registries API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=registries,scope=Namespaced,shortName=reg
// +kubebuilder:printcolumn:name="IMAGE",type=string,priority=1,JSONPath=`.spec.image`
// +kubebuilder:printcolumn:name="REGISTRY_URL",type=string,JSONPath=`.status.serverURL`
// +kubebuilder:printcolumn:name="NOTARY_URL",type=string,JSONPath=`.status.notaryURL`
// +kubebuilder:printcolumn:name="CAPACITY",type=string,priority=1,JSONPath=`.status.capacity`
// +kubebuilder:printcolumn:name="STATUS",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="AGE",type=date,JSONPath=`.metadata.creationTimestamp`
type Registry struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RegistrySpec   `json:"spec"`
	Status RegistryStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// RegistryList contains a list of Registry
type RegistryList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Registry `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Registry{}, &RegistryList{}, &Repository{}, &RepositoryList{})
}
