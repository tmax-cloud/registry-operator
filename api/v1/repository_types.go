package v1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

type RepositorySpec struct {
	Name     string         `json:"name"`
	Versions []ImageVersion `json:"versions"`
	Registry string         `json:"registry"`
}

type ImageVersion struct {
	CreatedAt metav1.Time `json:"createdAt"`
	Version   string      `json:"version"`
	Delete    bool        `json:"delete"`
}

// +kubebuilder:object:root=true

// Repository is the Schema for the repositories API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=repositories,scope=Namespaced,shortName=repo
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
