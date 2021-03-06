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

// VirtualObject is a CRD without a dedicated controller. Practically, it is just an entry in the Kubernetes API
// that is used as placeholder for action like Delete and Call.
type VirtualObject struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VirtualObjectSpec   `json:"spec,omitempty"`
	Status VirtualObjectStatus `json:"status,omitempty"`
}

type VirtualObjectSpec struct{}

type VirtualObjectStatus struct {
	Lifecycle `json:",inline"`
}

func (in *VirtualObject) GetReconcileStatus() Lifecycle {
	return in.Status.Lifecycle
}

func (in *VirtualObject) SetReconcileStatus(lifecycle Lifecycle) {
	in.Status.Lifecycle = lifecycle
}

// +kubebuilder:object:root=true

// VirtualObjectList contains a list of Virtual Objects.
type VirtualObjectList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VirtualObject `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VirtualObject{}, &VirtualObjectList{})
}
