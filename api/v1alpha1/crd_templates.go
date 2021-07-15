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

type TemplateSpec struct {
	// Services are lookups to service specifications
	// +optional
	Services map[string]ServiceSpec `json:"services"`

	// Monitors are lookups to monitoring packages
	// +optional
	Monitors map[string]MonitorSpec `json:"monitors"`
}

type MonitorSpec struct {
	// Agent is the sidecar that will be deployed in the same pod as the app
	Agent ServiceSpec `json:"agent,omitempty"`

	// Dashboard is dashboard payload that will be installed in Grafana.
	Dashboard DashboardSpec `json:"dashboard,omitempty"`
}

type DashboardSpec struct {
	File string `json:"file"`

	Payload string `json:"payload"`
}

type TemplateStatus struct {
	IsRegistered bool `json:"isRegistered"`
}

// +kubebuilder:object:root=true
type TemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Template `json:"items"`
}
