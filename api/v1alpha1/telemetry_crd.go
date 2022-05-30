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

// TelemetrySpec defines the desired state of Telemetry
type TelemetrySpec struct {
	// Import are references to monitoring packages that will be used in the monitoring stack.
	// +optional
	Import []string `json:"import,omitempty"`
}

// TelemetryStatus defines the observed state of Telemetry
type TelemetryStatus struct {
	Lifecycle `json:",inline"`

	GrafanaEndpoint string `json:"grafanaEndpoint"`

	PrometheusEndpoint string `json:"prometheusEndpoint"`
}

func (in *Telemetry) GetReconcileStatus() Lifecycle {
	return in.Status.Lifecycle
}

func (in *Telemetry) SetReconcileStatus(lifecycle Lifecycle) {
	in.Status.Lifecycle = lifecycle
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Telemetry is the Schema for the telemetries API
type Telemetry struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TelemetrySpec   `json:"spec,omitempty"`
	Status TelemetryStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// TelemetryList contains a list of Telemetry
type TelemetryList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Telemetry `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Telemetry{}, &TelemetryList{})
}
