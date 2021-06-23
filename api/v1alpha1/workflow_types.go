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
	Actions []Action `json:"actions"`
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
	CreateService *ServiceSpec `json:"createService,omitempty"`

	// +optional
	CreateServiceGroup *ServiceGroupSpec `json:"createServiceGroup,omitempty"`

	// +optional
	Stop *StopSpec `json:"stop,omitempty"`

	// +optional
	Wait *WaitSpec `json:"wait,omitempty"`
}

type StopSpec struct {
	ServiceSelector `json:",inline"`

	Schedule *SchedulerSpec `json:"schedule,omitempty"`
}

type WaitSpec struct {
	// Ready waits for the given groups to be running
	// +optional
	Ready []string `json:"ready,omitempty"`

	// Complete waits for the given groups to be succeeded
	// +optional
	Complete []string `json:"complete,omitempty"`
}

type WorkflowStatus struct {
	EtherStatus `json:",inline"`
}

func (s *Workflow) GetStatus() *EtherStatus {
	return &s.Status.EtherStatus
}

// +kubebuilder:object:root=true
type WorkflowList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Workflow `json:"items"`
}
