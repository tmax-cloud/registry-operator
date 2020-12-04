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

// ImageSignRequestSpec defines the desired state of ImageSignRequest
type ImageSignRequestSpec struct {
	// Image example: docker.io/library/alpine:3
	Image          string `json:"image"`
	Signer         string `json:"signer"`
	RegistrySecret `json:"registryLogin,omitempty"`
}

type RegistrySecret struct {
	DcjSecretName  string `json:"dcjSecretName"`
	CertSecretName string `json:"certSecretName"`
}

// ImageSignRequestStatus defines the observed state of ImageSignRequest
type ImageSignRequestStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	*ImageSignResponse `json:"imageSignResponse,omitempty"`
}

type ResponseResult string

const (
	ResponseResultSuccess = ResponseResult("Success")
	ResponseResultFail    = ResponseResult("Fail")
)

type ImageSignResponse struct {
	// Result: Success / Fail
	Result  ResponseResult `json:"result,omitempty"`
	Reason  string         `json:"reason,omitempty"`
	Message string         `json:"message,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=isr

// ImageSignRequest is the Schema for the imagesignrequests API
type ImageSignRequest struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ImageSignRequestSpec   `json:"spec,omitempty"`
	Status ImageSignRequestStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ImageSignRequestList contains a list of ImageSignRequest
type ImageSignRequestList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ImageSignRequest `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ImageSignRequest{}, &ImageSignRequestList{})
}
