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
	SchemeBuilder.Register(&DistributedGroup{}, &DistributedGroupList{})
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

type DistributedGroup struct {
	metav1.TypeMeta `json:",inline"`

	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the behavior of the object
	Spec DistributedGroupSpec `json:"spec"`

	// Most recently observed status of the object
	// +optional
	Status DistributedGroupStatus `json:"status,omitempty"`
}

type DistributedGroupSpec struct {
	// ServiceSpec includes a service specification. It conflicts with TemplateRef.
	// +optional
	ServiceSpec *ServiceSpec `json:"service,omitempty"`

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

	// Schedule defines the interval between the creation of services within the group. Scheduled creation is not
	// supported in collocated mode. Since Pods are intended to be disposable and replaceable, we cannot add a
	// container to a Pod once it has been created
	// +optional
	Schedule *SchedulerSpec `json:"schedule,omitempty"`

	// Domain specifies the location where Service will be placed. For this to work,
	// the nodes included in the domain must have the label domain:{{domain-name}}.
	// for the moment simply match domain to a specific node. this will change in the future
	// +optional
	Domain string `json:"domain,omitempty"`
}

type DistributedGroupStatus struct {
	Lifecycle `json:",inline"`

	// ExpectedServices is a list of services that belong to this group.
	// These services will be located in the same namespace as the group.
	// +optional
	ExpectedServices ServiceSpecList `json:"expectedServices,omitempty"`
}

func (in *DistributedGroup) GetLifecycle() []*Lifecycle {
	return []*Lifecycle{&in.Status.Lifecycle}
}

func (in *DistributedGroup) SetLifecycle(lifecycle Lifecycle) {
	in.Status.Lifecycle = lifecycle
}

// DistributedGroupList returns a list of DistributedGroup objects
// +kubebuilder:object:root=true
type DistributedGroupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DistributedGroup `json:"items"`
}
