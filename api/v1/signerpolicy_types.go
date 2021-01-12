package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type SignerPolicySpec struct {
	Signers []string `json:"signers"`
}
type SignerPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec SignerPolicySpec `json:"spec"`
}

type SignerPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SignerPolicy `json:"items"`
}
