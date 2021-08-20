package v1alpha1

import (
	"strings"

	v1 "k8s.io/api/core/v1"
)

type NamespacedName struct {
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
}

const (
	Separator = '/'
)

// String returns the general purpose string representation
func (n NamespacedName) String() string {
	return n.Namespace + string(Separator) + n.Name
}

type ServiceSpec struct {
	// NamespacedName is the name of the desired service. If unspecified, it is automatically set by the respective controller
	// +optional
	NamespacedName `json:"namespacedName"`

	// PortRef is a list of names of Ports that participate in the Mesh (autodiscovery + rewiring)
	// +optional
	PortRefs []string `json:"addPorts"`

	// List of volumes that can be mounted by containers belonging to the pod.
	// +optional
	Volumes []v1.Volume `json:"volumes,omitempty"`

	// Container is the container running the application
	Container v1.Container `json:"container,omitempty"`

	// MonitorTemplateRef is a list of references to monitoring packages
	// +optional
	MonitorTemplateRefs []string `json:"monitorTemplateRef,omitempty"`

	// Domain specifies the location where Service will be placed. For this to work,
	// the nodes included in the domain must have the label domain:{{domain-name}}
	// +optional
	Domain string `json:"domain,omitempty"`

	// Resources specifies limitations as to how the container will access host resources
	// +optional
	Resources *Resources `json:"resources,omitempty"`
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

/*
type ServiceStatus struct {
	Lifecycle `json:",inline"`

	// IP is the IP of the kubernetes service (not the kubernetes pod).
	// This convention allows to avoid blocking until the pod ip becomes known.
	IP string `json:"ip"`
}

func (s *Service) GetLifecycle() Lifecycle {
	return s.Status.Lifecycle
}

func (s *Service) SetLifecycle(lifecycle Lifecycle) {
	s.Status.Lifecycle = lifecycle
}

// +kubebuilder:object:root=true
type ServiceSpecList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Service `json:"items"`
}

*/

const (
	idListSeparator = " "
)

type ServiceSpecList []*ServiceSpec

func (list ServiceSpecList) String() string {
	return strings.Join(list.GetNames(), idListSeparator)
}

func (list ServiceSpecList) GetNames() []string {
	names := make([]string, len(list))

	for i, service := range list {
		names[i] = service.NamespacedName.Name
	}

	return names
}

func (list ServiceSpecList) GetNamespaces() []string {
	namespace := make([]string, len(list))

	for i, service := range list {
		namespace[i] = service.NamespacedName.Namespace
	}

	return namespace
}

// ByNamespace return the services by the namespace they belong to.
func (list ServiceSpecList) ByNamespace() map[string][]string {
	all := make(map[string][]string)

	for _, s := range list {
		// get namespace
		sublist := all[s.NamespacedName.Namespace]

		// append service to the namespace
		sublist = append(sublist, s.NamespacedName.Name)

		// update namespace
		all[s.NamespacedName.Namespace] = sublist
	}

	return all
}
