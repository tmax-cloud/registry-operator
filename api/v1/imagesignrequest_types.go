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
	// Image name to sign (example: docker.io/library/alpine:3)
	Image string `json:"image"`
	// ImageSigner's metadata name to sign image
	Signer string `json:"signer"`
	// Secrets to login registry
	RegistrySecret `json:"registryLogin,omitempty"`
}

type RegistrySecret struct {
	// Registry's imagePullSecret for login
	// If you don't have dockerconfigjson type's secret in this namespace,
	// you should refer to https://kubernetes.io/docs/tasks/configure-pod-container/pull-image-private-registry/
	// to make it first.
	DcjSecretName string `json:"dcjSecretName"`
	// If you want to trust registry's certificate, enter certifiacete's secret name
	CertSecretName string `json:"certSecretName,omitempty"`
}

// ImageSignRequestStatus defines the observed state of ImageSignRequest
type ImageSignRequestStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	*ImageSignResponse `json:"imageSignResponse,omitempty"`
}

type ResponseResult string

const (
	ResponseResultSigning = ResponseResult("Signing")
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
// +kubebuilder:printcolumn:name="IMAGE",type=string,JSONPath=`.spec.image`
// +kubebuilder:printcolumn:name="SIGNER",type=string,JSONPath=`.spec.signer`
// +kubebuilder:printcolumn:name="STATUS",type=string,JSONPath=`.status.imageSignResponse.result`
// +kubebuilder:printcolumn:name="AGE",type=date,JSONPath=`.metadata.creationTimestamp`

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
