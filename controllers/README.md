# Controllers

In Kubernetes, controllers are control loops that watch the state of
your [cluster](https://kubernetes.io/docs/reference/glossary/?all=true#term-cluster), then make or request changes where
needed. A sync loop (also called reconciliation loop) is what must be done in  order to synchronize the ‘desired state’ with the ‘observed state’.

The reconciliation cycle is where the framework gives us back control after a watch has passed up an event. At this
point we don’t have the information about the type of event because we are working on level-based triggers. Below is a
model of what a common reconciliation cycle for a controller that manages a CRD could look like



<img src="https://assets.openshift.com/hubfs/Imported_Blog_Media/rafop3.png" alt="img" style="zoom: 150%;" />

As we can see from the diagram the main steps are:

1. Retrieve the interested CR instance.
2. Manage the instance validity. We don’t want to try to do anything on an instance that does not carry valid values.
3. Manage instance initialization. If some values of the instance are not initialized, this section will take care of
   it.
4. Manage instance deletion. If the instance is being deleted, and we need to do some specific clean up, this is where
   we manage it.
5. Manage controller business logic. If the above steps all pass we can finally manage and execute the reconciliation
   logic for this particular instance. This will be very controller specific.

source: https://cloud.redhat.com/blog/kubernetes-operators-best-practices

## Available Controllers

* Service: deploys and manage all the necessary components (dns service, pod) to run a service instance

* Cluster: deploys and manage a set of Services

* Chaos: inject faults into the Services

* Workflow: orchestrate the testbed

## Controller Families

Apart from implemented the described functionality, each of the available controller may act as the skeleton for
building more controllers.

Next, we present the family of controllers that each available controller belongs to.

* Service: provides the skeleton for managing an external object (e.g, Pod) via the Typed client's API.

* Chaos: provides a skeleton for managing an external object (e.g, Chaos) via
  the [Unstructured](https://pkg.go.dev/k8s.io/apimachinery/pkg/apis/meta/v1/unstructured#Unstructured) client's API.  
  This method allows objects that do not have Golang structs registered to be manipulated generically. This can be used
  to deal with the API objects from a plug-in. Unstructured objects still have functioning TypeMeta features-- kind,
  version, etc.

* Cluster: provides the skeleton for managing a collection of similar objects (e.g, Frisbee Services). In contrast to
  the previous cases, the cluster lifecycle is not tied to a particular Service, but it rather depends on the aggregated
  outcome. For example, a cluster may be running even if some services have failed.

* Workflow:  provides the skeleton for a Frisbee controller that manage a collection of unsimilar objects (e.g,
  Services, Chaos, Clusters, ...).

## Be Careful

* Although we can remove a service when is complete, we recommend not to id as it will break the cluster manager -- it
  needs to track the total number of created services.