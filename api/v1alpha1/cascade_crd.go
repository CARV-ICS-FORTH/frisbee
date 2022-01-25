/*
Copyright 2021 ICS-FORTH.

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

// CascadeSpec defines the desired state of Cascade.
type CascadeSpec struct {
	GenerateFromTemplate `json:",inline"`

	// Schedule defines the interval between the creation of services within the group. Executed creation is not
	// supported in collocated mode. Since Pods are intended to be disposable and replaceable, we cannot add a
	// container to a Pod once it has been created
	// +optional
	Schedule *SchedulerSpec `json:"schedule,omitempty"`

	// Suspend flag tells the controller to suspend subsequent executions, it does
	// not apply to already started executions.  Defaults to false.
	// +optional
	Suspend *bool `json:"suspend,omitempty"`
}

// CascadeStatus defines the observed state of Cascade.
type CascadeStatus struct {
	Lifecycle `json:",inline"`

	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// QueuedJobs is a list of Chaos jobs scheduled for creation by the cascade.
	// +optional
	QueuedJobs []ChaosSpec `json:"queuedJobs,omitempty"`

	// ScheduledJobs points to the next QueuedJobs.
	ScheduledJobs int `json:"scheduledJobs,omitempty"`

	// LastScheduleTime provide information about  the last time a Chaos job was successfully scheduled.
	LastScheduleTime *metav1.Time `json:"lastScheduleTime,omitempty"`
}

func (in *Cascade) GetReconcileStatus() Lifecycle {
	return in.Status.Lifecycle
}

func (in *Cascade) SetReconcileStatus(lifecycle Lifecycle) {
	in.Status.Lifecycle = lifecycle
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Cascade is the Schema for the clusters API.
type Cascade struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CascadeSpec   `json:"spec,omitempty"`
	Status CascadeStatus `json:"status,omitempty"`
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
