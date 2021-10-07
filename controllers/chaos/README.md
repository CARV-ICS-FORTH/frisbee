/*
The dynamic package on the other hand, uses a simple type, unstructured.Unstructured, to represent all object values
from the API server. Type Unstructured uses a collection of nested map[string]interface{} values to create an internal
structure that closely resemble the REST payload from the server.

The dynamic package defers all data bindings until runtime. This means programs that use the dynamic client will not get
any of the benefits of type validation until the program is running. This may be a problem for certain types of
applications that require strong data type check and validation.

Being loosely coupled, however, means that programs that uses the dynamic package do not require recompilation when the
client API changes. The client program has more flexibility in handling updates to the API surface without knowing ahead
of time what those changes are.

// PartitionSpec separate the given Pod from the rest of the network. This chaos typeis retractable // (either manually
or after a duration) and can be waited at both Running and Success Phase. // Running phase begins when the failure is
injected. Success begins when the failure is retracted. // If anything goes wrong in between, the chaos goes into Failed
phase. type PartitionSpec struct { Selector ServiceSelector `json:"selector"`

	// +optional
	Duration *metav1.Duration `json:"duration,omitempty"`

}

// KillSpec terminates the selected Pod. Because this failure is permanent, it can only be waited in the // Running
Phase. It does not go through Success. type KillSpec struct { Selector ServiceSelector `json:"selector,omitempty"`
}
*/