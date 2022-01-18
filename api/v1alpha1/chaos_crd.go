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

type FaultType string

const (
	FaultPartition = FaultType("partition")
	FaultKill      = FaultType("kill")
)

// PartitionSpec separate the given Pod from the rest of the network. This chaos typeis retractable
// (either manually or after a duration) and can be waited at both Running and Pass Phase.
// Running phase begins when the failure is injected. Pass begins when the failure is retracted.
// If anything goes wrong in between, the chaos goes into Failed phase.
type PartitionSpec struct {
	Service string `json:"service,omitempty"`

	// Duration is the time after which Frisbee will roll back the injected fault.
	// +optional
	Duration string `json:"duration,omitempty"`
}

// KillSpec terminates the selected Pod. Because this failure is permanent, it can only be waited in the
// Running Phase. It does not go through Pass.
type KillSpec struct {
	Service string `json:"service,omitempty"`
}

type EmbedFaultType struct {
	// +optional
	Partition *PartitionSpec `json:"partition,omitempty"`

	// +optional
	Kill *KillSpec `json:"kill,omitempty"`
}

// ChaosSpec defines the desired state of Chaos.
type ChaosSpec struct {
	// Type indicate the type of the injected fault
	// +kubebuilder:validation:Enum=partition;kill;
	Type FaultType `json:"type"`

	*EmbedFaultType `json:",inline"`
}

// ChaosStatus defines the observed state of Chaos.
type ChaosStatus struct {
	Lifecycle `json:",inline"`

	// LastScheduleTime provide information about  the last time a Pod was scheduled.
	LastScheduleTime *metav1.Time `json:"lastScheduleTime,omitempty"`
}

func (in *Chaos) GetReconcileStatus() Lifecycle {
	return in.Status.Lifecycle
}

func (in *Chaos) SetReconcileStatus(lifecycle Lifecycle) {
	in.Status.Lifecycle = lifecycle
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Chaos is the Schema for the chaos API.
type Chaos struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ChaosSpec   `json:"spec,omitempty"`
	Status ChaosStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ChaosList contains a list of Chaos.
type ChaosList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Chaos `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Chaos{}, &ChaosList{})
}
