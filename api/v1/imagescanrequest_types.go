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

// ScanRequestStatusType is status type of scan request
type ScanRequestStatusType string

const (
	ScanRequestSuccess    ScanRequestStatusType = "Success"
	ScanRequestFail       ScanRequestStatusType = "Fail"
	ScanRequestPending    ScanRequestStatusType = "Pending"
	ScanRequestProcessing ScanRequestStatusType = "Processing"
	// ScanRequestError is scan request is failed
	ScanRequestError ScanRequestStatusType = "Error"
)

// ScanTarget is a target setting to scan images
type ScanTarget struct {
	// Registry URL (example: docker.io)
	RegistryURL string `json:"registryUrl"`
	// Image path (example: library/alpine:3)
	Images []string `json:"images"`
	// The name of certificate secret for private registry.
	CertificateSecret string `json:"certificateSecret,omitempty"`
	// The name of secret containing login credential of registry
	ImagePullSecret string `json:"imagePullSecret,omitempty"`
}

// ScanResult is result of scanning an image
type ScanResult struct {
	//Scan summary
	Summary map[string]int `json:"summary,omitempty"`
	//Scan fatal message
	Fatal []string `json:"fatal,omitempty"`
	//Scan vulnerabilities
	Vulnerabilities map[string]Vulnerabilities `json:"vulnerabilities,omitempty"`
}

// Vulnerability is the information of the vulnerability found.
type Vulnerability struct {
	// Severity name
	Name string `json:"Name,omitempty"`
	// Severity namespace
	NamespaceName string `json:"NamespaceName,omitempty"`
	// Description for severity
	Description string `json:"Description,omitempty"`
	// Description link
	Link string `json:"Link,omitempty"`
	// Severity degree
	Severity string `json:"Severity,omitempty"`
	// Metadata
	//Metadata runtime.RawExtension `json:"Metadata,omitempty"`
	// Fixed version
	FixedBy string `json:"FixedBy,omitempty"`
}

// Vulnerabilities is a set of Vulnerability instances
type Vulnerabilities []Vulnerability

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ImageScanRequestSpec defines the desired state of ImageScanRequest
type ImageScanRequestSpec struct {
	ScanTargets []ScanTarget `json:"scanTargets"`
	// Do not verify registry server's certificate
	Insecure bool `json:"insecure",omitempty`
	// The number of fixable issues allowable
	MaxFixable int `json:"maxFixable,omitempty"`
	// Whether to send result to report server
	SendReport bool `json:"sendReport,omitempty"`
}

// ImageScanRequestStatus defines the observed state of ImageScanRequest
type ImageScanRequestStatus struct {
	//Scan message for status
	Message string `json:"message,omitempty"`
	//Scan error reason
	Reason string `json:"reason,omitempty"`
	//Scan status
	Status ScanRequestStatusType `json:"status,omitempty"`
	//Scna results {docker.io/library/alpine:3: {summary : {"Low" : 1, "Medium" : 2, ...}}
	Results map[string]ScanResult `json:"results,omitempty"`
}

// ImageScanRequestESReport is a report to send the result to Elastic Search
type ImageScanRequestESReport struct {
	Image string `json:"image,omitempty"`
	//Scna results {docker.io/library/alpine:3: {summary : {"Low" : 1, "Medium" : 2, ...}}
	Result ScanResult `json:"result,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=icr
// +kubebuilder:printcolumn:name="STATUS",type=string,JSONPath=`.status.status`
// +kubebuilder:printcolumn:name="AGE",type=date,JSONPath=`.metadata.creationTimestamp`

// ImageScanRequest is the Schema for the imagescanrequests API
type ImageScanRequest struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ImageScanRequestSpec   `json:"spec,omitempty"`
	Status ImageScanRequestStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ImageScanRequestList contains a list of ImageScanRequest
type ImageScanRequestList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ImageScanRequest `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ImageScanRequest{}, &ImageScanRequestList{})
}
