package v1alpha1

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func init() {
	SchemeBuilder.Register(&Service{}, &ServiceList{})
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

type Service struct {
	metav1.TypeMeta `json:",inline"`

	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the behavior of the object
	Spec ServiceSpec `json:"spec"`

	// Most recently observed status of the object
	// +optional
	Status ServiceStatus `json:"status,omitempty"`
}

type ServiceSpec struct {
	// Image describes the name of the container for this node
	Image string `json:"image"`

	// Ports lists the ports required for the service to work
	// +optional
	Ports []Port `json:"ports,omitempty"`

	// Resources specifies limitations as to how the container will access host resources
	// +optional
	Resources *Resources `json:"resources,omitempty"`

	// Env defines environment variables for the containers that run the Service
	// +optional
	Env []v1.EnvVar `json:"env,omitempty"`

	// Domain specifies the location where Service will be placed. For this to work,
	// the nodes included in the domain must have the label domain:{{domain-name}}
	// +optional
	Domain string `json:"domain,omitempty"`

	// Command overwrites the default command of the image
	// +optional
	Command []string `json:"command,omitempty"`

	// Args define arguments to the command of the image
	// +optional
	Args []string `json:"args,omitempty"`
}

type Port struct {
	Port int32  `json:"port"`
	Name string `json:"name"`
}

// NIC specifies the capabilities of the emulated network interface.
type NIC struct {
	Rate    string `json:"rate,omitempty"`
	Latency string `json:"latency,omitempty"`
}

// Disk specifies the capabilities of the emulated storage device.
type Disk struct {
	// ReadBPS limits read rate (bytes per second)
	ReadBPS string `json:"readbps,omitempty"`

	// ReadIOPS limits read rate (IO per second)
	ReadIOPS string `json:"readiops,omitempty"`

	// WriteBPS limits write rate (bytes per second)
	WriteBPS string `json:"writebps,omitempty"`

	// WriteIOPS limits write rate (IO per second)
	WriteIOPS string `json:"writeiops,omitempty"`
}

// Resources specifies limitations as to how the container will access host resources.
type Resources struct {
	Memory string `json:"memory,omitempty"`
	CPU    string `json:"cpu,omitempty"`
	NIC    NIC    `json:"nic,omitempty"`
	Disk   Disk   `json:"disk,omitempty"`
}

type ServiceStatus struct {
	EtherStatus `json:",inline"`
}

func (s *Service) GetStatus() EtherStatus {
	return s.Status.EtherStatus
}

func (s *Service) SetStatus(status EtherStatus) {
	s.Status.EtherStatus = status
}

// +kubebuilder:object:root=true
type ServiceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Service `json:"items"`
}
