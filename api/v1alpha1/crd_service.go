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
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func init() {
	SchemeBuilder.Register(&Service{}, &ServiceList{})
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

type Service struct {
	metav1.TypeMeta `json:",inline"`

	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the behavior of the object
	Spec ServiceSpec `json:"spec"`

	// Most recently observed status of the object
	// +optional
	Status ServiceStatus `json:"status,omitempty"`
}

// Agents are sidecar services will be deployed in the same Pod as the Service container.
type Agents struct {
	// Telemetry is a list of references to monitoring packages.
	// +optional
	Telemetry []string `json:"telemetry,omitempty"`
}

type ServiceFromTemplate struct {
	// TemplateRef refers to a service template. It conflicts with Service.
	TemplateRef string `json:"templateRef"`

	// Inputs is a list of inputs passed to the objects.
	// +optional
	Inputs map[string]string `json:"inputs,omitempty"`
}

type ServiceSpec struct {
	// ServiceFromTemplate populates the service fields from a template. This is used for backward compatibility
	// with Cluster with just one instance. This field cannot be used in conjunction with other fields.
	// +optional
	*ServiceFromTemplate `json:"fromTemplate,omitempty"`

	// List of sidecar agents
	// +optional
	Agents *Agents `json:"agents,omitempty"`

	// Container is the container running the application
	Container v1.Container `json:"container,omitempty"`

	// Resources specifies limitations as to how the container will access host resources.
	// +optional
	Resources *Resources `json:"resources,omitempty"`

	// List of volumes that can be mounted by containers belonging to the pod.
	// +optional
	Volumes []v1.Volume `json:"volumes,omitempty"`

	// Domain specifies the location where Service will be placed. For this to work,
	// the nodes included in the domain must have the label domain:{{domain-name}}.
	// for the moment simply match domain to a specific node. this will change in the future
	// +optional
	Domain string `json:"domain,omitempty"`
}

// NIC specifies the capabilities of the emulated network interface.
type NIC struct {
	Rate    string `json:"rate,omitempty"`
	Latency string `json:"latency,omitempty"`
}

// Disk specifies the capabilities of the emulated storage device.
type Disk struct {
	// ReadBPS limits read rate (bytes per second)
	ReadBPS string `json:"readbps,omitempty"`

	// ReadIOPS limits read rate (IO per second)
	ReadIOPS string `json:"readiops,omitempty"`

	// WriteBPS limits write rate (bytes per second)
	WriteBPS string `json:"writebps,omitempty"`

	// WriteIOPS limits write rate (IO per second)
	WriteIOPS string `json:"writeiops,omitempty"`
}

// Resources specifies limitations as to how the container will access host resources.
type Resources struct {
	Memory string `json:"memory,omitempty"`
	CPU    string `json:"cpu,omitempty"`
	NIC    *NIC   `json:"nic,omitempty"`
	Disk   *Disk  `json:"disk,omitempty"`
}

type ServiceStatus struct {
	Lifecycle `json:",inline"`

	// LastScheduleTime provide information about  the last time a Pod was scheduled.
	LastScheduleTime *metav1.Time `json:"lastScheduleTime,omitempty"`
}

func (in *Service) GetReconcileStatus() Lifecycle {
	return in.Status.Lifecycle
}

func (in *Service) SetReconcileStatus(lifecycle Lifecycle) {
	in.Status.Lifecycle = lifecycle
}

// ServiceList returns a list of Service objects
// +kubebuilder:object:root=true
type ServiceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           SList `json:"items"`
}
