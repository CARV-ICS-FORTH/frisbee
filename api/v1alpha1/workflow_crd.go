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
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type DNSPrefix string

func (a DNSPrefix) Convert(service string) string {
	return fmt.Sprintf("%s.%s", service, a)
}

// Ingress is a collection of routing rules that govern how external users access services
// running in a Kubernetes cluster.
type Ingress struct {
	// DNSPrefix is the postfix from which the ingress will be available.
	// Example: grafana.localhost, grafana.{MYIP}.nip.io, grafana.platform.science-hangar.eu
	DNSPrefix DNSPrefix `json:"host,omitempty"`

	// UseAmbassador if set annotates ingresses with 'kubernetes.io/ingress.class=ambassador'
	// so to be managed by the Ambassador Ingress controller.
	// +optional
	UseAmbassador bool `json:"useAmbassador"`
}

// Assert is a source of information about whether the state of the workflow after a given time is correct or not.
// This is needed because some workflows may run in infinite-horizons.
type Assert struct {
	// SLA is a Grafana alert that will be triggered if the SLA condition is met.
	// SLA assertion is applicable throughout the execution of an action.
	SLA string `json:"sla"`

	// State describe the runtime condition that should be met after the action has been executed
	State string `json:"state"`
}

// Action delegates arguments to the proper action handler.
type Action struct {
	ActionType string `json:"action"`

	// Name is a unique identifier of the action
	Name string `json:"name"`

	// DependsOn defines the conditions for the execution of this action
	// +optional
	DependsOn *WaitSpec `json:"depends,omitempty"`

	// Assert defines the conditions under which the workflow will terminate with a "passed" or "failed" message
	// +optional
	Assert *Assert `json:"assert,omitempty"`

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

type EmbedActions struct {
	// +optional
	Service *ServiceSpec `json:"service,omitempty"`

	// +optional
	Cluster *ClusterSpec `json:"cluster,omitempty"`

	// +optional
	Chaos *ChaosSpec `json:"chaos,omitempty"`
}

// WorkflowSpec defines the desired state of Workflow
type WorkflowSpec struct {
	WithTelemetry *TelemetrySpec `json:"withTelemetry,omitempty"`

	// Actions are the tasks that will be taken.
	Actions ActionList `json:"actions"`

	// Suspend flag tells the controller to suspend subsequent executions, it does
	// not apply to already started executions.  Defaults to false.
	// +optional
	Suspend *bool `json:"suspend,omitempty"`
}

// WorkflowStatus defines the observed state of Workflow
type WorkflowStatus struct {
	Lifecycle `json:",inline"`

	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// Executed is a list of executed actions.
	// +optional
	Executed map[string]metav1.Time `json:"scheduled,omitempty"`
}

func (in *Workflow) GetReconcileStatus() Lifecycle {
	return in.Status.Lifecycle
}

func (in *Workflow) SetReconcileStatus(lifecycle Lifecycle) {
	in.Status.Lifecycle = lifecycle
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Workflow is the Schema for the workflows API
type Workflow struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   WorkflowSpec   `json:"spec,omitempty"`
	Status WorkflowStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// WorkflowList contains a list of Workflow
type WorkflowList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Workflow `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Workflow{}, &WorkflowList{})
}
