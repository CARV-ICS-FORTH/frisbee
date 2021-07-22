package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func init() {
	SchemeBuilder.Register(&DataPort{}, &DataPortList{})
}

type PortType string

const (
	Inport = PortType("input")

	Outport = PortType("output")
)

type PortProtocol string

const (
	Direct = PortProtocol("direct")

	Kafka = PortProtocol("kafka")
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

type DataPort struct {
	metav1.TypeMeta `json:",inline"`

	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the behavior of the object
	Spec DataPortSpec `json:"spec"`

	// Most recently observed status of the object
	Status DataPortStatus `json:"status,omitempty"`
}

type DataPortSpec struct {
	// Type indicate the role of the DstPort. It can be Input or Output.
	// +kubebuilder:validation:Enum=input;output
	Type PortType `json:"type"`

	*EmbedType `json:",inline"`

	Protocol PortProtocol `json:"protocol"`

	*ProtocolSpec `json:",inline"`
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
	Selector *metav1.LabelSelector `json:"selector"`
}

type Output struct{}

// //////////////////////////
// Protocol Spec
// //////////////////////////

type ProtocolSpec struct {
	// +optional
	Direct *DirectSpec `json:"direct,omitempty"`

	// +optional
	Kafka *KafkaSpec `json:"kafka,omitempty"`
}

type DirectSpec struct {
	// +optional
	DstAddr string `json:"dstAddr"`

	// +optional
	DstPort int `json:"dstPort"`
}

type KafkaSpec struct {
	Host string `json:"host"`

	Port int `json:"port"`

	Queue string `json:"queue"`
}

// //////////////////////////
// Protocol Status
// //////////////////////////

type DataPortStatus struct {
	Lifecycle `json:",inline"`

	ProtocolStatus `json:",inline"`
}

type ProtocolStatus struct {
	// +optional
	Direct *DirectStatus `json:"direct"`

	// +optional
	Kafka *KafkaStatus `json:"kafka"`
}

type DirectStatus struct {
	// LocalAddr is the IP of the associated target
	LocalAddr string `json:"localAddr"`

	// LocalPort is the DstPort of the associated target
	LocalPort int `json:"localPort"`

	// RemoteAddr is the IP of the remote target
	RemoteAddr string `json:"remoteAddr"`

	// RemotePort is the DstPort of the remote target
	RemotePort int `json:"remotePort"`
}

type KafkaStatus struct{}

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
