/*
Copyright 2021-2023 ICS-FORTH.

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
	netv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Service is the Schema for the services API.
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type Service struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ServiceSpec   `json:"spec,omitempty"`
	Status ServiceStatus `json:"status,omitempty"`
}

type SetField struct {
	// Field is the path to the field whose value will be replaced.
	// Examples: Containers.0.Ports.0
	Field string `json:"field"`
	Value string `json:"value"`
}

// Decorators takes-in a PodSpec, add some functionality and returns it.
type Decorators struct {
	// +optional
	Labels map[string]string `json:"labels,omitempty"`

	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`

	// SetFields is used to populate fields. Used for dynamic assignment based templated inputs.
	// +optional
	SetFields []SetField `json:"setFields,omitempty"`

	// Telemetry is a list of referenced agents responsible to monitor the Service.
	// Agents are sidecar services will be deployed in the same Pod as the Service container.
	// +optional
	Telemetry []string `json:"telemetry,omitempty"`

	// IngressPort builds an ingress for making the service's port accessible outside the Kubernetes cluster.
	// +optional
	IngressPort *netv1.ServiceBackendPort `json:"ingressPort,omitempty"`
}

// Callable is a script that is executed within the service container, and returns a value.
// For example, a callable can be a command for stopping the containers that run in the Pod.
type Callable struct {
	// Container specific the name of the container to which we will run the command
	Container string `json:"container"`

	// Container specifies a command and arguments to stop the targeted container in an application-specific manner.
	Command []string `json:"command"`
}

// ServiceSpec defines the desired state of Service.
type ServiceSpec struct {
	// +optional
	Decorators Decorators `json:"decorators,omitempty"`

	// +optional
	Callables map[string]Callable `json:"callables,omitempty"`

	corev1.PodSpec `json:",inline"`
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

// ServiceList contains a list of Service.
type ServiceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Service `json:"items"`
}

func (list ServiceList) Names() []string {
	names := make([]string, len(list.Items))

	for i, item := range list.Items {
		names[i] = item.GetName()
	}

	return names
}

func init() {
	SchemeBuilder.Register(&Service{}, &ServiceList{})
}
