package v1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

type RepositorySpec struct {
	// Repository name
	Name string `json:"name,omitempty"`
	// Versions(=Tags) of image
	Versions []ImageVersion `json:"versions,omitempty"`
	// Name of Registry which owns repository
	Registry string `json:"registry,omitempty"`
}

type ImageVersion struct {
	// Created time of image version
	CreatedAt metav1.Time `json:"createdAt,omitempty"`
	// Version(=Tag) name
	Version string `json:"version"`
	// If true, this version will be deleted soon.
	Delete bool `json:"delete,omitempty"`
	// If signed image, image signer name is set.
	Signer string `json:"signer,omitempty"`
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
