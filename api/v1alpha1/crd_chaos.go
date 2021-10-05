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
	SchemeBuilder.Register(&Chaos{}, &ChaosList{})
}

type FaultType string

const (
	FaultPartition = FaultType("partition")
	FaultKill      = FaultType("kill")
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

type Chaos struct {
	metav1.TypeMeta `json:",inline"`

	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the behavior of the object
	Spec ChaosSpec `json:"spec"`

	// Most recently observed status of the object
	Status ChaosStatus `json:"status,omitempty"`
}

type ChaosSpec struct {
	// Type indicate the type of the injected fault
	// +kubebuilder:validation:Enum=partition;kill;
	Type FaultType `json:"type"`

	*EmbedFaultType `json:",inline"`
}

type EmbedFaultType struct {
	// +optional
	Partition *PartitionSpec `json:"partition,omitempty"`

	Kill *KillSpec `json:"kill,omitempty"`
}

type PartitionSpec struct {
	Selector ServiceSelector `json:"selector"`

	// +optional
	Duration *metav1.Duration `json:"duration,omitempty"`
}

type KillSpec struct {
	Selector ServiceSelector `json:"selector,omitempty"`

	Schedule *SchedulerSpec `json:"schedule,omitempty"`
}

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

// ChaosList returns a list of Chaos objects
// +kubebuilder:object:root=true
type ChaosList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Chaos `json:"items"`
}
