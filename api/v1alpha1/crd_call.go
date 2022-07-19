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

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Call is the Schema for the Call API.
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type Call struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CallSpec   `json:"spec,omitempty"`
	Status CallStatus `json:"status,omitempty"`
}

// MatchOutputs defined a set of remote command outputs that must be matched. The limit for both Stdout and Stderr
// is 1024 characters.
type MatchOutputs struct {
	// Stdout is a regex that describes the expected output from stdout. It cannot be longer than 1024 characters.
	// +optional
	// +kubebuilder:validation:MaxLength=1024
	Stdout *string `json:"stdout,omitempty"`

	// Stderr is a regex that describes the expected output from stderr. It cannot be longer than 1024 characters.
	// +optional
	// +kubebuilder:validation:MaxLength=1024
	Stderr *string `json:"stderr,omitempty"`
}

// CallSpec defines the desired state of Call.
type CallSpec struct {
	// Callable is the name of the endpoint that will be called
	// +kubebuilder:validation:minlength=1
	Callable string `json:"callable,omitempty"`

	// Services is a list of services that will be stopped.
	// +kubebuilder:validation:minimum=1
	Services []string `json:"services"`

	// Expect declares a list of expected outputs. The number of expected outputs must be the same
	// as the number of defined services.
	// +optional
	Expect []MatchOutputs `json:"expect,omitempty"`

	// Tolerate specifies the conditions under which the call will fail. If undefined, the call fails
	// immediately when a call to service has failed.
	// +optional
	Tolerate *TolerateSpec `json:"tolerate,omitempty"`

	// Until defines the conditions under which the CR will stop spawning new jobs.
	// If used in conjunction with inputs, it will loop over inputs until the conditions are met.
	// +optional
	Until *ConditionalExpr `json:"until,omitempty"`

	// Schedule defines the interval between the invocations of the callable.
	// +optional
	Schedule *SchedulerSpec `json:"schedule,omitempty"`

	// Suspend flag tells the controller to suspend subsequent executions, it does
	// not apply to already started executions.  Defaults to false.
	// +optional
	Suspend *bool `json:"suspend,omitempty"`
}

// CallStatus defines the observed state of Call.
type CallStatus struct {
	Lifecycle `json:",inline"`

	// QueuedJobs is a list of services scheduled for stopping.
	// +optional
	QueuedJobs []Callable `json:"queuedJobs,omitempty"`

	// ScheduledJobs points to the next QueuedJobs.
	ScheduledJobs int `json:"scheduledJobs,omitempty"`

	// LastScheduleTime provide information about  the last time a Service was successfully scheduled.
	LastScheduleTime *metav1.Time `json:"lastScheduleTime,omitempty"`
}

func (in *Call) GetReconcileStatus() Lifecycle {
	return in.Status.Lifecycle
}

func (in *Call) SetReconcileStatus(lifecycle Lifecycle) {
	in.Status.Lifecycle = lifecycle
}

// +kubebuilder:object:root=true

// CallList contains a list of Call jobs.
type CallList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Call `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Call{}, &CallList{})
}
