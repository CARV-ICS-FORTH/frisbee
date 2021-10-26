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
	SchemeBuilder.Register(&Telemetry{}, &TelemetryList{})
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

type Telemetry struct {
	metav1.TypeMeta `json:",inline"`

	metav1.ObjectMeta `json:"metadata,omitempty"`

	// MonitorSpec defines the Telemetry of services
	Spec TelemetrySpec `json:"spec"`

	// Most recently observed status of the object
	// +optional
	Status TelemetryStatus `json:"status,omitempty"`
}

type TelemetrySpec struct {
	// Ingress defines how to get traffic into your Kubernetes cluster.
	Ingress *Ingress `json:"ingress,omitempty"`

	// ImportMonitors are references to monitoring packages that will be used in the monitoring stack.
	// +optional
	ImportMonitors []string `json:"importMonitors,omitempty"`
}

type TelemetryStatus struct {
	Lifecycle `json:",inline"`

	PrometheusURI string `json:"prometheusURI"`
	GrafanaURI    string `json:"grafanaURI"`
}

func (in *Telemetry) GetReconcileStatus() Lifecycle {
	return in.Status.Lifecycle
}

func (in *Telemetry) SetReconcileStatus(lifecycle Lifecycle) {
	in.Status.Lifecycle = lifecycle
}

// TelemetryList returns a list of Telemetry objects
// +kubebuilder:object:root=true
type TelemetryList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Telemetry `json:"items"`
}
