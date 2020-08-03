package v1

import (
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

	Image         string `json:"image"`
	Description   string `json:"description,omitempty"`
	LoginId       string `json:"loginId"`
	LoginPassword string `json:"loginPassword"`

	// The name of the configmap where the registry config.yml content
	CustomConfigYml string `json:"customConfigYml,omitempty"`

	DomainName         string             `json:"domainName,omitempty"`
	RegistryReplicaSet RegistryReplicaSet `json:"registryReplicaSet,omitempty"`

	// Supported service types are ingress and loadBalancer
	RegistryService       RegistryService `json:"service"`
	PersistentVolumeClaim RegistryPVC     `json:"persistentVolumeClaim"`
}

type RegistryReplicaSet struct {
	Labels       map[string]string    `json:"labels"`
	NodeSelector map[string]string    `json:"nodeSelector"`
	Selector     metav1.LabelSelector `json:"selector"`
	Tolerations  []corev1.Toleration  `json:"tolerations"`
}

type RegistryService struct {
	// use ingress service type
	Ingress Ingress `json:"ingress,omitempty"` // [TODO] One Of

	//
	LoadBalancer LoadBalancer `json:"loadBalancer,omitempty"` // [TODO] One Of
}

type RegistryPVC struct {
	// (default: /var/lib/registry)
	MountPath string `json:"mountPath,omitempty"`

	// Use the pvc you have created
	Exist ExistPvc `json:"exist,omitempty"` // [TODO] One Of

	Create CreatePvc `json:"create,omitempty"` // [TODO] One Of
}

// RegistryStatus defines the observed state of Registry
type RegistryStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html

	Conditions     []RegistryCondition `json:"conditions"`
	Phase          string              `json:"phase"`
	Message        string              `json:"message"`
	Reason         string              `json:"reason"`
	PhaseChangedAt metav1.Time         `json:"phaseChangedAt"`
	Capacity       string              `json:"capacity"`
}

type RegistryCondition struct {
	LastProbeTime      metav1.Time `json:"lastProbeTime"`
	LastTransitionTime metav1.Time `json:"lastTransitionTime"`
	Message            string      `json:"message"`
	Reason             string      `json:"reason"`
	Status             string      `json:"status"`
	Type               string      `json:"type"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Registry is the Schema for the registries API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=registries,scope=Namespaced,shortName=reg
// +kubebuilder:printcolumn:name="Image",type=string,JSONPath=`.spec.image`
// +kubebuilder:printcolumn:name="Capacity",type=string,JSONPath=`.status.capacity`
// +kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
type Registry struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RegistrySpec   `json:"spec"`
	Status RegistryStatus `json:"status,omitempty"`

	OperatorStartTime string `json:"operatorStartTime,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// RegistryList contains a list of Registry
type RegistryList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Registry `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Registry{}, &RegistryList{})
}
