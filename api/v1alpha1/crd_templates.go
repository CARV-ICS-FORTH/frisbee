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
	SchemeBuilder.Register(&Template{}, &TemplateList{})
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

type Template struct {
	metav1.TypeMeta `json:",inline"`

	metav1.ObjectMeta `json:"metadata,omitempty"`

	// MonitorSpec defines the template of services
	Spec TemplateSpec `json:"spec"`

	// Most recently observed status of the object
	// +optional
	Status TemplateStatus `json:"status,omitempty"`
}

type Inputs struct {
	// Parameters define dynamically valued fields. The values are given by higher level entities, such as the workflow.
	// +optional
	Parameters map[string]string `json:"parameters"`
}

type Scheme struct {
	// Inputs are dynamic fields that populate the spec.
	// +optional
	Inputs *Inputs `json:"inputs,omitempty"`

	// Spec is the Service specification whose values will be replaced by parameters.
	Spec string `json:"spec"`
}

type TemplateSpec struct {
	// Services are indices to service specifications
	// +optional
	Services map[string]Scheme `json:"services,omitempty"`

	// Monitors are indices to monitoring packages
	// +optional
	Monitors map[string]Scheme `json:"monitors,omitempty"`
}

type MonitorSpec struct {
	// Agent is the sidecar that will be deployed in the same pod as the app
	Agent ServiceSpec `json:"agent,omitempty"`

	// Dashboard is dashboard payload that will be installed in Grafana.
	Dashboard DashboardSpec `json:"dashboard,omitempty"`
}

type DashboardSpec struct {
	FromConfigMap string `json:"fromConfigMap"`

	File string `json:"file"`
}

type TemplateStatus struct {
	Lifecycle `json:",inline"`

	IsRegistered bool `json:"isRegistered"`
}

func (in *Template) GetReconcileStatus() Lifecycle {
	return in.Status.Lifecycle
}

func (in *Template) SetReconcileStatus(lifecycle Lifecycle) {
	in.Status.Lifecycle = lifecycle
}

// TemplateList returns a list of Template objects
// +kubebuilder:object:root=true
type TemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Template `json:"items"`
}
