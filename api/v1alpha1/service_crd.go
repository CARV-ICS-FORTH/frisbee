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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type PVC struct {
	Name string                           `json:"name"`
	Spec corev1.PersistentVolumeClaimSpec `json:"spec,omitempty"`
}

// Requirements points to Kinds and their respective configurations required for the Service operation.
// For example, this field can be used to create PVCs dedicated to this service.
type Requirements struct {
	// +optional
	PVC *PVC `json:"persistentVolumeClaim,omitempty"`
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
	// +optional
	Memory string `json:"memory,omitempty"`
	// +optional
	CPU string `json:"cpu,omitempty"`
	// +optional
	NIC *NIC `json:"nic,omitempty"`
	// +optional
	Disk *Disk `json:"disk,omitempty"`
}

type SetField struct {
	// Field is the path to the field whose value will be replaced.
	// Examples: Containers.0.Ports.0
	Field string `json:"field"`
	Value string `json:"value"`
}

type GracefulStop struct {
	// Container specific the name of the container to which we will run the command
	Container string `json:"container"`

	// Container specifies a command and arguments to stop the targeted container in an application-specific manner.
	Command []string `json:"command"`
}

// Decorators takes in a PodSpec, add some functionality and returns it.
type Decorators struct {
	// Resources specifies limitations as to how the container will access host resources.
	// +optional
	Resources *Resources `json:"resources,omitempty"`

	// Telemetry is a list of referenced agents responsible to monitor the Service.
	// Agents are sidecar services will be deployed in the same Pod as the Service container.
	// +optional
	Telemetry []string `json:"telemetry,omitempty"`

	// Dashboard is dashboard payload that will be installed in Grafana.
	// This option is only applicable to Agents.
	// +optional
	Dashboards metav1.LabelSelector `json:"dashboards,omitempty"`

	// SetFields is used to populate fields. Used for dynamic assignment based templated inputs.
	// +optional
	SetFields []SetField `json:"setFields,omitempty"`

	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`

	// +optional
	Labels map[string]string `json:"labels,omitempty"`

	// GracefulStop allows you to define a command for stopping the containers that run in the Pod.
	// Used by the Stop command.
	// +optional
	GracefulStop *GracefulStop `json:"gracefulStop,omitempty"`
}

// ServiceSpec defines the desired state of Service.
type ServiceSpec struct {
	// +optional
	Requirements *Requirements `json:"requirements,omitempty"`

	// +optional
	Decorators *Decorators `json:"decorators,omitempty"`

	// +kubebuilder:validation:Optional
	// +optional
	corev1.PodSpec `json:",inline,omitempty"`
}

// ServiceStatus defines the observed state of Service.
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

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Service is the Schema for the services API.
type Service struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ServiceSpec   `json:"spec,omitempty"`
	Status ServiceStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ServiceList contains a list of Service.
type ServiceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Service `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Service{}, &ServiceList{})
}
