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
*/