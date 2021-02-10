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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// RegistryJobState is a state of the RegistryJob
type RegistryJobState string

// RegistryJob's states
const (
	RegistryJobStatePending   = RegistryJobState("Pending")
	RegistryJobStateRunning   = RegistryJobState("Running")
	RegistryJobStateCompleted = RegistryJobState("Completed")
	RegistryJobStateFailed    = RegistryJobState("Failed")
)

// RegistryJobSyncRepository is a job type of synchronizing repository list of external registry
type RegistryJobSyncRepository struct {
	// ExternalRegistry refers to the ExternalRegistry
	ExternalRegistry corev1.LocalObjectReference `json:"externalRegistry"`
}

// RegistryJobSpec defines the desired state of RegistryJob
type RegistryJobSpec struct {
	// TTL is a time-to-live (in seconds)
	// If 0, it is deleted immediately
	// If -1, it is not deleted
	// If ttl > 0, it is deleted after ttl seconds
	TTL int `json:"ttl"`

	// Priority is an integer value, greater or equal to 0
	Priority int `json:"priority,omitempty"`

	// SyncRepository is a repository sync type job
	SyncRepository *RegistryJobSyncRepository `json:"syncRepository,omitempty"`
}

// RegistryJobStatus defines the observed state of RegistryJob
type RegistryJobStatus struct {
	// State is a state of the RegistryJob
	State RegistryJobState `json:"state"`

	// Message is a message for the RegistryJob (normally an error string)
	Message string `json:"message,omitempty"`

	// StartTime is actual time the task started
	StartTime *metav1.Time `json:"startTime,omitempty"`

	// CompletionTime is a time when the job is completed
	CompletionTime *metav1.Time `json:"completionTime,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=rj
// +kubebuilder:printcolumn:name="State",type=string,JSONPath=`.status.state`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// RegistryJob is the Schema for the jobs
type RegistryJob struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RegistryJobSpec   `json:"spec,omitempty"`
	Status RegistryJobStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// RegistryJobList contains a list of RegistryJob
type RegistryJobList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RegistryJob `json:"items"`
}

func init() {
	SchemeBuilder.Register(&RegistryJob{}, &RegistryJobList{})
}
