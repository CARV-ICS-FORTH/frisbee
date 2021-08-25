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
	Inputs Inputs `json:"inputs"`

	// Spec is the Service specification whose values will be replaced by arameters.
	Spec string `json:"spec"`
}

type TemplateSpec struct {
	// Services are lookups to service specifications
	// +optional
	Services map[string]Scheme `json:"services,omitempty"`

	// Monitors are lookups to monitoring packages
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

func (s *Template) GetLifecycle() []*Lifecycle {
	return []*Lifecycle{&s.Status.Lifecycle}
}

func (s *Template) SetLifecycle(lifecycle Lifecycle) {
	s.Status.Lifecycle = lifecycle
}

// +kubebuilder:object:root=true
type TemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Template `json:"items"`
}
