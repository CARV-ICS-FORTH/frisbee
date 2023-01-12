/*
Copyright 2021-2023 ICS-FORTH.

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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Cascade is the Schema for the clusters API.
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type Cascade struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CascadeSpec   `json:"spec,omitempty"`
	Status CascadeStatus `json:"status,omitempty"`
}

// CascadeSpec defines the desired state of Cascade.
type CascadeSpec struct {
	GenerateObjectFromTemplate `json:",inline"`

	// Schedule defines the interval between the creation of services within the group.
	// +optional
	Schedule *TaskSchedulerSpec `json:"schedule,omitempty"`

	// Suspend forces the Controller to stop scheduling any new jobs until it is resumed. Defaults to false.
	// +optional
	Suspend *bool `json:"suspend,omitempty"`

	// SuspendWhen automatically sets Suspend to True, when certain conditions are met.
	// +optional
	SuspendWhen *ConditionalExpr `json:"suspendWhen,omitempty"`
}

// CascadeStatus defines the observed state of Cascade.
type CascadeStatus struct {
	Lifecycle `json:",inline"`

	// QueuedJobs is a list of Chaos jobs scheduled for creation by the cascade.
	// +optional
	QueuedJobs []ChaosSpec `json:"queuedJobs,omitempty"`

	// ExpectedTimeline is the result of evaluating a timeline distribution into specific points in time.
	// +optional
	ExpectedTimeline Timeline `json:"expectedTimeline,omitempty"`

	// ScheduledJobs points to the next QueuedJobs.
	ScheduledJobs int `json:"scheduledJobs,omitempty"`

	// LastScheduleTime provide information about  the last time a Chaos job was successfully scheduled.
	LastScheduleTime metav1.Time `json:"lastScheduleTime,omitempty"`
}

func (in *Cascade) GetReconcileStatus() Lifecycle {
	return in.Status.Lifecycle
}

func (in *Cascade) SetReconcileStatus(lifecycle Lifecycle) {
	in.Status.Lifecycle = lifecycle
}

// +kubebuilder:object:root=true

// CascadeList contains a list of Cascades.
type CascadeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Cascade `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Cascade{}, &CascadeList{})
}
