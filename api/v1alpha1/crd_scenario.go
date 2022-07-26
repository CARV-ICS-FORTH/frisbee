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

// Scenario is the Schema for the Scenarios API.
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type Scenario struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ScenarioSpec   `json:"spec,omitempty"`
	Status ScenarioStatus `json:"status,omitempty"`
}

type ActionType string

const (
	// ActionService creates a new service.
	ActionService ActionType = "Service"
	// ActionCluster creates multiple services running in a shared context.
	ActionCluster ActionType = "Cluster"
	// ActionChaos injects failures into the running system.
	ActionChaos ActionType = "Chaos"
	// ActionCascade injects multiple failures into the running system.
	ActionCascade ActionType = "Cascade"
	// ActionDelete deletes a created Frisbee resource (i.e services, clusters,).
	ActionDelete ActionType = "Delete"
	// ActionCall starts a remote process execution, from the controller to the targeted services.
	ActionCall ActionType = "Call"
)

// Action is a step in a workflow that defines a particular part of a testing process.
type Action struct {
	// ActionType refers to a category of actions that can be associated with a specific controller.
	// +kubebuilder:validation:Enum=Service;Cluster;Chaos;Cascade;Delete;Call
	ActionType ActionType `json:"action"`

	// Name is a unique identifier of the action
	Name string `json:"name"`

	// DependsOn defines the conditions for the execution of this action
	// +optional
	DependsOn *WaitSpec `json:"depends,omitempty"`

	// Assert defines the conditions that must be maintained after the action has been started.
	// If the evaluation of the condition is false, the Scenario will abort immediately.
	// +optional
	Assert *ConditionalExpr `json:"assert,omitempty"`

	*EmbedActions `json:",inline"`
}

type WaitSpec struct {
	// Running waits for the given groups to be running
	// +optional
	Running []string `json:"running,omitempty"`

	// Success waits for the given groups to be succeeded
	// +optional
	Success []string `json:"success,omitempty"`

	// After is the time offset since the beginning of this action.
	// +optional
	After *metav1.Duration `json:"after,omitempty"`
}

type DeleteSpec struct {
	// Jobs is a list of jobs to be deleted. The format is {"kind":"name"}, e.g, {"service","client"}
	Jobs []string `json:"jobs"`
}

type EmbedActions struct {
	// +optional
	Service *GenerateFromTemplate `json:"service,omitempty"`

	// +optional
	Cluster *ClusterSpec `json:"cluster,omitempty"`

	// +optional
	Chaos *GenerateFromTemplate `json:"chaos,omitempty"`

	// +optional
	Cascade *CascadeSpec `json:"cascade,omitempty"`

	// +optional
	Delete *DeleteSpec `json:"delete,omitempty"`

	// +optional
	Call *CallSpec `json:"call,omitempty"`
}

// ScenarioSpec defines the desired state of Scenario.
type ScenarioSpec struct {
	// Actions are the tasks that will be taken.
	Actions []Action `json:"actions"`

	// Suspend flag tells the controller to suspend subsequent executions, it does
	// not apply to already started executions.  Defaults to false.
	// +optional
	Suspend *bool `json:"suspend,omitempty"`
}

// ScenarioStatus defines the observed state of Scenario.
type ScenarioStatus struct {
	Lifecycle `json:",inline"`

	// ScheduledJobs is a list of references to the names of executed actions.
	// +optional
	ScheduledJobs []string `json:"scheduledJobs,omitempty"`

	// GrafanaEndpoint points to the local Grafana instance
	GrafanaEndpoint string `json:"grafanaEndpoint,omitempty"`

	// PrometheusEndpoint points to the local Prometheus instance
	PrometheusEndpoint string `json:"prometheusEndpoint,omitempty"`
}

func (in *Scenario) GetReconcileStatus() Lifecycle {
	return in.Status.Lifecycle
}

func (in *Scenario) SetReconcileStatus(lifecycle Lifecycle) {
	in.Status.Lifecycle = lifecycle
}

// +kubebuilder:object:root=true

// ScenarioList contains a list of Scenario.
type ScenarioList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Scenario `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Scenario{}, &ScenarioList{})
}
