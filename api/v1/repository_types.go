package v1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

type RepositorySpec struct {
	Name     string         `json:"name,omitempty"`
	Versions []ImageVersion `json:"versions,omitempty"`
	Registry string         `json:"registry,omitempty"`
}

type ImageVersion struct {
	CreatedAt metav1.Time `json:"createdAt,omitempty"`
	Version   string      `json:"version"`
	Delete    bool        `json:"delete,omitempty"`
	Signer    string      `json:"signer,omitempty"`
}

// +kubebuilder:object:root=true

// Repository is the Schema for the repositories API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=repositories,scope=Namespaced,shortName=repo
// +kubebuilder:printcolumn:name="REPOSITORY",type=string,JSONPath=`.spec.name`
// +kubebuilder:printcolumn:name="REGISTRY",type=string,JSONPath=`.spec.registry`
// +kubebuilder:printcolumn:name="AGE",type=date,JSONPath=`.metadata.creationTimestamp`
type Repository struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec RepositorySpec `json:"spec"`
}

// +kubebuilder:object:root=true

type RepositoryList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Repository `json:"items"`
}
