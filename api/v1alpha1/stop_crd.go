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

// StopSpec defines the desired state of Stop.
type StopSpec struct {
	// Services is a list of services that will be stopped.
	Services []string `json:"services"`

	// Until defines the conditions under which the CR will stop spawning new jobs.
	// If used in conjunction with inputs, it will loop over inputs until the conditions are met.
	// +optional
	Until *ConditionalExpr `json:"until,omitempty"`

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

// StopStatus defines the observed state of Stop.
type StopStatus struct {
	Lifecycle `json:",inline"`

	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// QueuedJobs is a list of services scheduled for stopping.
	// +optional
	QueuedJobs []*GracefulStop `json:"queuedJobs,omitempty"`

	// ScheduledJobs points to the next QueuedJobs.
	ScheduledJobs int `json:"scheduledJobs,omitempty"`

	// LastScheduleTime provide information about  the last time a Service was successfully scheduled.
	LastScheduleTime *metav1.Time `json:"lastScheduleTime,omitempty"`
}

func (in *Stop) GetReconcileStatus() Lifecycle {
	return in.Status.Lifecycle
}

func (in *Stop) SetReconcileStatus(lifecycle Lifecycle) {
	in.Status.Lifecycle = lifecycle
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Stop is the Schema for the Stop API.
type Stop struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   StopSpec   `json:"spec,omitempty"`
	Status StopStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// StopList contains a list of Stop jobs.
type StopList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Stop `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Stop{}, &StopList{})
}
