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

// RegistryCronJobSpec defines the desired state of RegistryJob
type RegistryCronJobSpec struct {
	// Schedule is a cron spec for periodic jobs
	Schedule string `json:"schedule"`

	// JobSpec is a spec for the job
	JobSpec RegistryJobSpec `json:"jobSpec"`
}

// RegistryCronJobStatus defines the observed state of RegistryJob
type RegistryCronJobStatus struct {
	// LastScheduledTime is the latest time when the job is scheduled
	LastScheduledTime *metav1.Time `json:"lastScheduledTime,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=rcj
// +kubebuilder:printcolumn:name="LastScheduledTime",type=string,JSONPath=`.status.lastScheduledTime`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// RegistryCronJob is the Schema for the jobs
type RegistryCronJob struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RegistryCronJobSpec   `json:"spec,omitempty"`
	Status RegistryCronJobStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// RegistryCronJobList contains a list of RegistryCronJob
type RegistryCronJobList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RegistryCronJob `json:"items"`
}

func init() {
	SchemeBuilder.Register(&RegistryCronJob{}, &RegistryCronJobList{})
}
