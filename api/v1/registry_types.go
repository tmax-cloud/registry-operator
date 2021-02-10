package v1

import (
	"github.com/operator-framework/operator-lib/status"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Status is registry status type
type Status string

const (
	// StatusNotReady is a status that registry is not ready
	StatusNotReady = Status("NotReady")
	// StatusRunning is a status taht registry is running
	StatusRunning = Status("Running")
	// StatusCreating is a status that registry subresources are being created
	StatusCreating = Status("Creating")
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
	LoginID string `json:"loginId"`
	// Login password for registry
	LoginPassword string `json:"loginPassword"`
	// If ReadOnly is true, clients will not be allowed to write(push) to the registry.
	ReadOnly bool `json:"readOnly,omitempty"`
	// Settings for notary service
	Notary RegistryNotary `json:"notary,omitempty"`
	// The name of the configmap where the registry config.yml content
	CustomConfigYml string `json:"customConfigYml,omitempty"`

	// Settings for registry's deployemnt
	RegistryDeployment RegistryDeployment `json:"registryDeployment,omitempty"`
	// Service type to expose registry
	RegistryService RegistryService `json:"service"`
	// Settings for registry pvc. Either `Exist` or `Create` must be entered.
	PersistentVolumeClaim RegistryPVC `json:"persistentVolumeClaim"`
}

// RegistryNotary is notary service configuration
type RegistryNotary struct {
	// Activate notary service to sign images
	Enabled bool `json:"enabled"`
	// Use Ingress or LoadBalancer
	// +kubebuilder:validation:Enum=Ingress;LoadBalancer
	ServiceType NotaryServiceType `json:"serviceType"`
	// Settings for notary pvc. Either `Exist` or `Create` must be entered.
	PersistentVolumeClaim NotaryPVC `json:"persistentVolumeClaim"`
}

// RegistryDeployment is deployment settings of registry server
type RegistryDeployment struct {
	// Deployment's label
	Labels map[string]string `json:"labels,omitempty"`
	// Registry pod's node selector
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`
	// Deployment's label selector
	Selector metav1.LabelSelector `json:"selector,omitempty"`
	// Deployment's toleration configuration
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`
	// Deployment's resource requirements (default: Both limits and requests are `cpu:100m` and `memory:512Mi`)
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`
}

// RegistryServiceType is type of registry service
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

	// Conditions are status of subresources
	Conditions status.Conditions `json:"conditions,omitempty"`
	// Phase is status of registry
	Phase string `json:"phase,omitempty"`
	// Message is a message of registry status
	Message string `json:"message,omitempty"`
	// Reason is a reason of registry status
	Reason string `json:"reason,omitempty"`
	// PhaseChangedAt is the time when phase was changed
	PhaseChangedAt metav1.Time `json:"phaseChangedAt,omitempty"`
	// Capacity is registry's srotage size
	Capacity string `json:"capacity,omitempty"`
	// ReadOnly is whether the registry is readonly
	ReadOnly bool `json:"readOnly,omitempty"`
	// ClusterIP is cluster ip of service
	ClusterIP string `json:"clusterIP,omitempty"`
	// LoadBalancerIP is external ip of service
	LoadBalancerIP string `json:"loadBalancerIP,omitempty"`
	// PodRecreateRequired is set if the registry pod is required to be recreated
	PodRecreateRequired bool `json:"podRecreateRequired,omitempty"`
	// ServerURL is registry server URL
	ServerURL string `json:"serverURL,omitempty"`
	// NotaryURL is notary server URL
	NotaryURL string `json:"notaryURL,omitempty"`
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

type Authorizer struct {
	Username string
	Password string
}
