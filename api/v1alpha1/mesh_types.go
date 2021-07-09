package v1alpha1

// Kind indicate the connection dependencies between the ports of different processes.
// We adhere to the established conventions of systemd units.
// https://www.freedesktop.org/software/systemd/man/systemd.unit.html
type Kind string

const (
	// MeshOffer indicates that the Port is active and provides a certain functionality.
	MeshOffer = Kind("offer")

	// MeshWants configures (weak) requirement dependencies on other units.
	// Units listed in this option will be started if the configuring unit is.
	// However, if the listed units fail to start or cannot be added to the transaction,
	// this has no impact on the validity of the transaction as a whole,
	// and this unit will still be started.
	//
	// This is the recommended way to hook the start-up of one unit to the start-up of another unit.
	MeshWants = Kind("wants")

	// MeshRequires is similar to Wants=, but declares a stronger requirement dependency.
	//
	// If this unit gets activated, the units listed will be activated as well.
	// If one of the other units fails to activate, and an ordering dependency After= on the failing unit is set,
	// this unit will not be started. Besides, with or without specifying After=,
	// this unit will be stopped if one of the other units is explicitly stopped.
	//
	// Often, it is a better choice to use Wants= instead of Requires= in order to achieve a system that
	// is more robust when dealing with failing services.
	//
	// Note that this dependency type does not imply that the other unit always has to be in active state when
	// this unit is running. Specifically: failing condition checks do not cause the start job of a unit with
	// a Requires= dependency on it to fail.
	MeshRequires = Kind("requires")

	// MeshRequisite is similar to Requires=. However, if the units listed here are not started already,
	// they will not be started and the starting of this unit will fail immediately. Requisite= does not
	// imply an ordering dependency, even if both units are started in the same transaction.
	// Hence this setting should usually be combined with After=, to ensure this unit is not started before the other unit.
	MeshRequisite = Kind("requisite")
)

// Mesh provides discovery capabilities to a service
type Mesh struct {
	// +optional
	Inputs []Port `json:"inputs,omitempty"`

	// +optional
	Outputs []Port `json:"outputs,omitempty"`
}

// Port represents a property of a Component or Graph through which it communicates with the outer world.
type Port struct {
	// Name is a unique names that describes the port.
	Name string `json:"name"`

	// Labels are slices of values that can be used to organize and categorize
	// (scope and select) objects. May match selectors of replication controllers and services.
	// +optional
	Labels map[string]string `json:"labels,omitempty"`

	// Selector is used to discover services in the Mesh based on given criteria
	// +optional
	Selector *ServiceSelector `json:"selector"`

	// Type indicate the dependencies of the Port.
	Type string `json:"type"`

	// Protocol represents the protocol associated with this port.
	*EmbedProtocol `json:",inline"`
}

type EmbedProtocol struct {
	// Direct indicates a point-to-point traffic, where the server runs a label, and the client a label selector.
	// +optional
	Direct *Direct `json:"direct,omitempty"`

	// TCPProxy is an OSI level 4 proxy for routing remote clients to local servers
	TCPProxy *TCPProxy `json:"tcpproxy,omitempty"`

	// +optional
	File *File `json:"file,omitempty"`
}

// Direct allows for service discovery and point-to-point communication between two services in a Mesh.
// In this mode, the servers must expose some labels, and the clients will select them.
// It does not support multiplexing.
type Direct struct {
	// SetDstIP points the annotation of the remote client to which the server will write its ip.
	SetDstIP string `json:"setDst"`

	// SetDstPort points the annotation of the remote client to which the server will write its port.
	SetDstPort string `json:"setPort"`
}

type TCPProxy struct {
	//	tcpproxy.TCPProxy
}

type File struct{}
