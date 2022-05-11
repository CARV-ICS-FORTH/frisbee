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

type ActionType string

// List of supported actions
const (
	ActionService ActionType = "Service"
	ActionCluster ActionType = "Cluster"
	ActionChaos   ActionType = "Chaos"
	ActionCascade ActionType = "Cascade"
	ActionDelete  ActionType = "Delete"
	ActionCall    ActionType = "Call"
)

// Action delegates arguments to the proper action handler.
type Action struct {
	ActionType ActionType `json:"action"`

	// Name is a unique identifier of the action
	Name string `json:"name"`

	// DependsOn defines the conditions for the execution of this action
	// +optional
	DependsOn *WaitSpec `json:"depends,omitempty"`

	// Assert defines the conditions under which the Plan will terminate with a "passed" or "failed" message
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

// TestPlanSpec defines the desired state of TestPlan.
type TestPlanSpec struct {
	// Actions are the tasks that will be taken.
	Actions []Action `json:"actions"`

	// Suspend flag tells the controller to suspend subsequent executions, it does
	// not apply to already started executions.  Defaults to false.
	// +optional
	Suspend *bool `json:"suspend,omitempty"`
}

// TestPlanStatus defines the observed state of TestPlan.
type TestPlanStatus struct {
	Lifecycle `json:",inline"`

	WithTelemetry bool `json:"withTelemetry"`

	// Executed is a list of executed actions.
	// +optional
	Executed map[string]ConditionalExpr `json:"executed"`
}

func (in *TestPlan) GetReconcileStatus() Lifecycle {
	return in.Status.Lifecycle
}

func (in *TestPlan) SetReconcileStatus(lifecycle Lifecycle) {
	in.Status.Lifecycle = lifecycle
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// TestPlan is the Schema for the TestPlans API.
type TestPlan struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TestPlanSpec   `json:"spec,omitempty"`
	Status TestPlanStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// TestPlanList contains a list of TestPlan.
type TestPlanList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TestPlan `json:"items"`
}

func init() {
	SchemeBuilder.Register(&TestPlan{}, &TestPlanList{})
}
