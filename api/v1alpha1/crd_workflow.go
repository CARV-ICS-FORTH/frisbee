package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func init() {
	SchemeBuilder.Register(&Workflow{}, &WorkflowList{})
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:shortName=wf
// +kubebuilder:subresource:status

type Workflow struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the behavior of a workflow
	Spec WorkflowSpec `json:"spec,omitempty"`

	// Most recently observed status of the workflow
	// +optional
	Status WorkflowStatus `json:"status,omitempty"`
}

type WorkflowSpec struct {
	// ImportMonitors are references to monitoring packages that will be used in the monitoring stack.
	// +optional
	ImportMonitors []string `json:"importMonitors,omitempty"`

	// Actions are the tasks that will be taken.
	Actions []Action `json:"actions"`

	// Ingress defines external access to the services in a cluster, typically HTTP
	// Example: grafana.localhost, grafana.{MYIP}.nip.io,
	Ingress string `json:"ingress,omitempty"`
}

// Action delegates arguments to the proper action handler
type Action struct {
	ActionType string `json:"actiontype"`

	// Name is a unique identifier of the action
	Name string `json:"name"`

	// Depends define the conditions for the execution of this action
	// +optional
	Depends *WaitSpec `json:"depends,omitempty"`

	*EmbedActions `json:",inline"`
}

type EmbedActions struct {
	// +optional
	ServiceGroup *ServiceGroupSpec `json:"servicegroup,omitempty"`

	// +optional
	Stop *StopSpec `json:"stop,omitempty"`

	// +optional
	Wait *WaitSpec `json:"wait,omitempty"`

	// +optional
	Chaos *ChaosSpec `json:"chaos,omitempty"`
}

type StopSpec struct {
	Selector *ServiceSelector `json:"selector,omitempty"`

	Schedule *SchedulerSpec `json:"schedule,omitempty"`
}

type WaitSpec struct {
	// Running waits for the given groups to be running
	// +optional
	Running []string `json:"running,omitempty"`

	// Success waits for the given groups to be succeeded
	// +optional
	Success []string `json:"success,omitempty"`

	// Duration blocks waiting for the duration to expire
	// +optional
	Duration *metav1.Duration `json:"duration,omitempty"`
}

type WorkflowStatus struct {
	Lifecycle `json:",inline"`

	IsRunning bool `json:"isRunning"`
}

func (s *Workflow) GetLifecycle() Lifecycle {
	return s.Status.Lifecycle
}

func (s *Workflow) SetLifecycle(lifecycle Lifecycle) {
	s.Status.Lifecycle = lifecycle
}

// +kubebuilder:object:root=true
type WorkflowList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Workflow `json:"items"`
}
