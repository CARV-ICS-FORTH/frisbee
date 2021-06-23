package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func init() {
	SchemeBuilder.Register(&ServiceGroup{}, &ServiceGroupList{})
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

type ServiceGroup struct {
	metav1.TypeMeta `json:",inline"`

	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the behavior of the object
	Spec ServiceGroupSpec `json:"spec"`

	// Most recently observed status of the object
	// +optional
	Status ServiceGroupStatus `json:"status,omitempty"`
}

type ServiceGroupSpec struct {
	// TemplateRef refers to a service template.
	TemplateRef string `json:"templateRef"`

	// Env defines inputs for the services. When used with instances, it will spawn identical services.
	// For services with different parameters, use Inputs.
	// +optional
	Env map[string]string `json:"env,omitempty"`

	// Instances dictate the number of objects to be created for the service. If Env is specified, the values
	// with be identical across the spawned instances. For instances with different parameters, use Inputs.
	// +optional
	Instances int `json:"instances"  validate:"required_without=Inputs"`

	// Inputs are list of inputs passed to the objects. It cannot be used in conjunction
	// with Instances of Env. If Instances are defined, the size of the given array must be equal to the number of
	// instances. Otherwise, Instances are automatically  set to the number of the given array.
	// +optional
	Inputs []map[string]string `json:"inputs,omitempty" validate:"required_without=Instances"`

	// Interval defines the interval between the creation of services within the group
	// +optional
	Interval string `json:"interval"`
}

type ServiceGroupStatus struct {
	EtherStatus `json:",inline"`
}

func (s *ServiceGroup) GetStatus() *EtherStatus {
	return &s.Status.EtherStatus
}

// +kubebuilder:object:root=true
type ServiceGroupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ServiceGroup `json:"items"`
}
