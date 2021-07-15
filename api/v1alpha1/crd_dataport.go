package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func init() {
	SchemeBuilder.Register(&DataPort{}, &DataPortList{})
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

type DataPort struct {
	metav1.TypeMeta `json:",inline"`

	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the behavior of the object
	Spec DataPortSpec `json:"spec"`

	// Most recently observed status of the object
	// +optional
	Status DataPortStatus `json:"status,omitempty"`
}

type DataPortSpec struct {
	// Type indicate the role of the Port. It can be Input or Output.
	// +kubebuilder:validation:Enum=input;output
	Type string `json:"type"`

	*EmbedType `json:",inline"`

	Protocol string `json:"protocol"`

	*EmbedProtocol `json:",inline"`
}

type EmbedType struct {
	// +optional
	Input *Input `json:"input,omitempty"`

	// +optional
	Output *Output `json:"output,omitempty"`
}

type Input struct {
	// Selector is used to discover services in the DataMesh based on given criteria
	// +optional
	Selector *ServiceSelector `json:"selector"`
}

type Output struct{}

type EmbedProtocol struct {
	// +optional
	Direct *Direct `json:"direct"`
}

type Direct struct {
	// Spec defines the behavior of the object
	Spec DirectSpec `json:"spec"`

	// +optional
	Status DirectStatus `json:"status,omitempty"`
}

type DirectSpec struct {
	Port int `json:"port"`
}

type DirectStatus struct {
	LocalAddr string `json:"localAddr"`

	RemoteAddr string `json:"remoteAddr"`
}

type Kafka struct {
	// Spec defines the behavior of the object
	Spec KafkaSpec `json:"spec"`

	// +optional
	Status KafkaStatus `json:"status,omitempty"`
}

type KafkaSpec struct {
	Host string `json:"host"`

	Port int `json:"port"`

	Queue string `json:"queue"`
}

type KafkaStatus struct{}

type DataPortStatus struct {
	Lifecycle `json:",inline"`
}

func (s *DataPort) GetLifecycle() Lifecycle {
	return s.Status.Lifecycle
}

func (s *DataPort) SetLifecycle(lifecycle Lifecycle) {
	s.Status.Lifecycle = lifecycle
}

// +kubebuilder:object:root=true
type DataPortList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DataPort `json:"items"`
}
