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
	SchemeBuilder.Register(&CollocatedGroup{}, &CollocatedGroupList{})
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

type CollocatedGroup struct {
	metav1.TypeMeta `json:",inline"`

	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the behavior of the object
	Spec CollocatedGroupSpec `json:"spec"`

	// Most recently observed status of the object
	// +optional
	Status CollocatedGroupStatus `json:"status,omitempty"`
}

type CollocatedGroupSpec struct {
	// ServiceSpec includes a service specification. It conflicts with TemplateRef.
	// +optional
	ServiceSpec *ServiceSpec `json:"service"`

	// TemplateRef refers to a service template. It conflicts with Service.
	// +optional
	TemplateRef string `json:"templateRef"`

	// Instances dictate the number of objects to be created for the service. If Env is specified, the values
	// with be identical across the spawned instances. For instances with different parameters, use Inputs.
	// +optional
	Instances int `json:"instances"  validate:"required_without=Offers"`

	// Inputs are list of inputs passed to the objects. When used in conjunction with Instances, there can be
	// only one input and all the instances will run with identical parameters. If Instances is defined and there are
	// more than one inputs, the request will be rejected.
	Inputs []map[string]string `json:"inputs,omitempty" validate:"required_without=Instances"`
}

type CollocatedGroupStatus struct {
	Lifecycle `json:",inline"`

	// ExpectedServices is a list of services that belong to this group.
	// These services will be located in the same namespace as the group.
	ExpectedServices ServiceSpecList `json:"expectedServices"`
}

func (in *CollocatedGroup) GetLifecycle() []*Lifecycle {
	return []*Lifecycle{&in.Status.Lifecycle}
}

func (in *CollocatedGroup) SetLifecycle(lifecycle Lifecycle) {
	in.Status.Lifecycle = lifecycle
}

// CollocatedGroupList returns a list of CollocatedGroup objects
// +kubebuilder:object:root=true
type CollocatedGroupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CollocatedGroup `json:"items"`
}
