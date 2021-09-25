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
	SchemeBuilder.Register(&Cluster{}, &ClusterList{})
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

type Cluster struct {
	metav1.TypeMeta `json:",inline"`

	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the behavior of the object
	Spec ClusterSpec `json:"spec"`

	// Most recently observed status of the object
	// +optional
	Status ClusterStatus `json:"status,omitempty"`
}

type ClusterSpec struct {
	// TemplateRef refers to a service template. It conflicts with Service.
	TemplateRef string `json:"templateRef"`

	// Instances dictate the number of objects to be created for the service. If Env is specified, the values
	// with be identical across the spawned instances. For instances with different parameters, use Inputs.
	// +optional
	Instances int `json:"instances"`

	// Inputs are list of inputs passed to the objects. When used in conjunction with Instances, there can be
	// only one input and all the instances will run with identical parameters. If Instances is defined and there are
	// more than one inputs, the request will be rejected.
	// +optional
	Inputs []map[string]string `json:"inputs,omitempty"`

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

	// This flag tells the controller to suspend subsequent executions, it does
	// not apply to already started executions.  Defaults to false.
	// +optional
	Suspend *bool `json:"suspend,omitempty"`
}

type ClusterStatus struct {
	Lifecycle `json:",inline"`

	// Expected is a list of services scheduled for creation by the cluster.
	// +optional
	Expected []ServiceSpec `json:"expected,omitempty"`

	// A list of pointers to currently running services.
	// +optional
	Active []*Service `json:"active,omitempty"`

	// LastScheduleTime provide information about  the last time a Service was successfully scheduled.
	LastScheduleTime *metav1.Time `json:"lastScheduleTime,omitempty"`
}

func (in *Cluster) GetLifecycle() []*Lifecycle {
	return []*Lifecycle{&in.Status.Lifecycle}
}

func (in *Cluster) SetLifecycle(lifecycle Lifecycle) {
	in.Status.Lifecycle = lifecycle
}

// ClusterList returns a list of Cluster objects
// +kubebuilder:object:root=true
type ClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Cluster `json:"items"`
}
