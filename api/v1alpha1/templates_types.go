package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func init() {
	SchemeBuilder.Register(&Template{}, &TemplateList{})
}

// +kubebuilder:object:root=true

type Template struct {
	metav1.TypeMeta `json:",inline"`

	metav1.ObjectMeta `json:"metadata,omitempty"`

	// TemplateSpec defines the template of services
	Spec TemplateSpec `json:"spec"`
}

type TemplateSpec struct {
	// Services include the templates for all the services in the experiment
	// +optional
	Services map[string]ServiceSpec `json:"services,omitempty"`
}

// +kubebuilder:object:root=true
type TemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Template `json:"items"`
}
