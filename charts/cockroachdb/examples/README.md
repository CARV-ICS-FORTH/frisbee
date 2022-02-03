# Examples

### 1. Baseline Single

1) Start a standalone Cockroach database server
2) Load the database with keys
3) Run a sequence of YCSB workloads in the following order (A,B, C,F, D, E)

#### Observations

* Nothing strange

### 2. Baseline Cluster Deterministic

1) Start a set of individual Cockroach servers
2) Combine the individual servers into a Cockroach cluster.
3) Run a sequence of YCSB workloads (A,B, C,F, D, E), sending traffic to masters-0

#### Observations

* If we run the load with 400 threads then the entire system blocks after 66.2K requests.
* This behavior is not reproducible on baseline-single.
* Beware not to include invalid server names on the inputs. The failure is not detected.

### 3. Baseline Cluster Deterministic-OutOfOrder

1) Start a set of individual Cockroach servers
2) Combine the individual servers into a Cockroach cluster.
3) Run a sequence of YCSB workloads (A,B, C,F, D, E), sending traffic to masters-2

#### Observations

* The experiment blocks after 65K requests.

### 4. Baseline Cluster Non-Deterministic

1) Start a set of individual Cockroach servers
2) Combine the individual servers into a Cockroach cluster.
3) Run a sequence of YCSB workloads (A,B, C,F, D, E), sending traffic to all servers.

#### Observations

* Deadlock, like in example 3.

### 5. Scale-up Scheduled

1) Start a set of individual Cockroach servers
2) Combine the individual servers into a Cockroach cluster.
3) Run workload A sending traffic to a random node
4) Scale-up the topology by periodically adding new nodes

#### Observations

* The experiment requires every service to set the "advertise host" equal to the pod's name.
* The experiment blocks arbitrarily -- sometimes at 387K keys, sometimes at 900K.

### 6. Scale-up Conditional

1) Start a set of individual Cockroach servers
2) Combine the individual servers into a Cockroach cluster.
3) Stress the topology by periodically adding new clients sending traffic to a random node
4) Scale-up the topology when the tail-latency goes above a given threshold

#### Observations

* In the current implementation, the more-servers cluster will keep creating database nodes for as long as the alert
  remains active. Which means that if the performance does not recover (and acknowledged back to the controller) before
  the next reconciliation cycle begins, the controller will keep creating nodes. Is this the desired behavior, or do we
  need one new node per alert? Or do we need a creation interval within the same alert ?

* For the moment, you can implement this functionality using multiple clusters with logical dependencies between them.
  In this case, you have group a, group b, group c, with each group depending the previous (to become Running), and
  scheduled conditions with alerts.

### 7. Elastic Scale-down (Delete)

1) Start a set of individual Cockroach servers
2) Combine the individual servers into a Cockroach cluster.
3) Run workload A sending traffic to a random node
4) Scale-up the topology by adding 5 more services (group a).
5) Scale-up the topology by adding 5 more services (group b).
6) Scale-down the topology by **deleting** services in group a  (remove it from the Kubernetes API)

#### Observations

* Currently, deletion is supported only at the level of "Actions"/Jobs. You may delete a cluster, but not services
  within a cluster.
* If the "Action" is of type Service, then you can delete it directly.

### 8. Elastic Scale-down (Stop)

1) Start a set of individual Cockroach servers
2) Combine the individual servers into a Cockroach cluster.
3) Run workload A sending traffic to a random node
4) Scale-up the topology by adding new nodes
5) Scale-down the topology by periodically **stopping** some nodes (run command within target container to drain the
   node)

#### **Observations**

* With the present cockroach container it is difficult to terminate the cockroach process gracefully (lack of pidof,
  pkill, or ps)

* The cockroach's native way is to drain the node, without stopping the process. Given that, the services appear as "
  live" to Kubernetes, but without producing any data for Grafana.
* The experiment blocks randomly.

### 9. Elastic Scale-down (Kill)

1) Start a set of individual Cockroach servers
2) Combine the individual servers into a Cockroach cluster.
3) Run workload A sending traffic to a random node
4) Scale-up the topology by adding new nodes
5) Scale-down the topology by periodically **killing**  some nodes (use Chaos-Mesh to forcibly kill the container)

#### Observations

* We need to setup the "toleration" field in the Cluster. Otherwise, when a service fails, the cluster will immediately,
  before the experiment's activities are done.

### TODO

#### Failure at hotspot

This experiment configures a cluster of 3 cockroach nodes and sends traffic to one of them.

After 2 minutes, we inject a failure on the most used server.

Observations

* Not currently supported. We must change the macros to select services based on Grafana information.





   