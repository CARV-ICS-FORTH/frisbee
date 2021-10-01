// Licensed to FORTH/ICS under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. FORTH/ICS licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

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

// Ingress is a collection of routing rules that govern how external users access services running in a Kubernetes cluster.
type Ingress struct {
	// Host is the postfix from which the ingress will be available.
	// Example: grafana.localhost, grafana.{MYIP}.nip.io, grafana.platform.science-hangar.eu
	Host string `json:"host,omitempty"`

	// UseAmbassador if set annotates ingresses with 'kubernetes.io/ingress.class=ambassador'
	// so to be managed by the Ambassador Ingress controller.
	// +optional
	UseAmbassador bool `json:"useAmbassador"`
}

type WorkflowSpec struct {
	// ImportMonitors are references to monitoring packages that will be used in the monitoring stack.
	// +optional
	ImportMonitors []string `json:"importMonitors,omitempty"`

	// Actions are the tasks that will be taken.
	Actions ActionList `json:"actions"`

	// Ingress defines how to get traffic into your Kubernetes cluster.
	Ingress *Ingress `json:"ingress,omitempty"`

	// Suspend flag tells the controller to suspend subsequent executions, it does
	// not apply to already started executions.  Defaults to false.
	// +optional
	Suspend *bool `json:"suspend,omitempty"`
}

// Action delegates arguments to the proper action handler
type Action struct {
	ActionType string `json:"action"`

	// Name is a unique identifier of the action
	Name string `json:"name"`

	// DependsOn defines the conditions for the execution of this action
	// +optional
	DependsOn *WaitSpec `json:"depends,omitempty"`

	*EmbedActions `json:",inline"`
}

type EmbedActions struct {
	// +optional
	Service *ServiceSpec `json:"service,omitempty"`

	// +optional
	Cluster *ClusterSpec `json:"cluster,omitempty"`

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

	// Scheduled is a list of scheduled actions.
	// Do no add "omitempty" as it will break the initialization
	// +optional
	Scheduled map[string]bool `json:"scheduled"`
}

func (in *Workflow) GetReconcileStatus() Lifecycle {
	return in.Status.Lifecycle
}

func (in *Workflow) SetReconcileStatus(lifecycle Lifecycle) {
	in.Status.Lifecycle = lifecycle
}

// WorkflowList returns a list of Workflow objects
// +kubebuilder:object:root=true
type WorkflowList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Workflow `json:"items"`
}
