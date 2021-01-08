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
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

type ScanRequestStatusType string

const (
	ScanRequestSuccess ScanRequestStatusType = "Success"
	ScanRequestError   ScanRequestStatusType = "Error"
)

type Vulnerability struct {
	Name          string               `json:"Name,omitempty"`
	NamespaceName string               `json:"NamespaceName,omitempty"`
	Description   string               `json:"Description,omitempty"`
	Link          string               `json:"Link,omitempty"`
	Severity      string               `json:"Severity,omitempty"`
	Metadata      runtime.RawExtension `json:"Metadata,omitempty"`
	FixedBy       string               `json:"FixedBy,omitempty"`
}

type Vulnerabilities []Vulnerability

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ImageScanRequestSpec defines the desired state of ImageScanRequest
type ImageScanRequestSpec struct {
	ImageUrl         string        `json:"imageUrl"`
	AuthUrl          string        `json:"authUrl,omitempty"`
	Insecure         bool          `json:"insecure,omitempty"`
	ForceNonSSL      bool          `json:"forceNonSSL,omitempty"`
	Username         string        `json:"username,omitempty"`
	Password         string        `json:"password,omitempty"`
	Debug            bool          `json:"debug,omitempty"`
	SkipPing         bool          `json:"skipPing,omitempty"`
	TimeOut          time.Duration `json:"timeOut,omitempty"`
	FixableThreshold int           `json:"fixableThreshold,omitempty"`
	ElasticSearch    bool          `json:"elasticSearch,omitempty"`
}

// ImageScanRequestStatus defines the observed state of ImageScanRequest
type ImageScanRequestStatus struct {
	Message         string                     `json:"message,omitempty"`
	Reason          string                     `json:"reason,omitempty"`
	Status          ScanRequestStatusType      `json:"status,omitempty"`
	Summary         map[string]int             `json:"summary,omitempty"`
	Fatal           []string                   `json:"fatal,omitempty"`
	Vulnerabilities map[string]Vulnerabilities `json:"vulnerabilities,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="STATUS",type=string,JSONPath=`.status.status`

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
