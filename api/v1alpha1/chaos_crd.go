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
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type Fault unstructured.Unstructured

// ChaosSpec defines the desired state of Kaos
type ChaosSpec struct {
	Fault `json:",inline"`
}

// ChaosStatus defines the observed state of Chaos
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

// Chaos is the Schema for the chaos API
type Chaos struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ChaosSpec   `json:"spec,omitempty"`
	Status ChaosStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ChaosList contains a list of Chaos
type ChaosList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Chaos `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Chaos{}, &ChaosList{})
}
