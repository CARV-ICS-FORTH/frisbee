# Examples

### 1. Baseline Single

1) Start a standalone Redis database server
2) Load the database with keys
3) Run a sequence of YCSB workloads in the following order (A,B, C,F, D, E)

#### Observations

* The requests are served so fast that the telemetry agents can send only one sample before the workload is done.
* The re-loader fails with an i/o timeout

### 2. Baseline Cluster Deterministic

1) Start a set of individual Redis servers
2) Combine the individual servers into a Redis cluster.
3) Run a sequence of YCSB workloads (A,B, C,F, D, E), sending traffic to masters-1

#### Observations

### 3. Baseline Cluster Deterministic-OutOfOrder

1) Start a set of individual Redis servers
2) Combine the individual servers into a Redis cluster.
3) Run a sequence of YCSB workloads (A,B, C,F, D, E), sending traffic to masters-2

#### Observations

### 4. Baseline Cluster Non-Deterministic

1) Start a set of individual Redis servers
2) Combine the individual servers into a Redis cluster.
3) Run a sequence of YCSB workloads (A,B, C,F, D, E), sending traffic to all servers.

#### Observations

### 5. Scale-up Scheduled

1) Start a set of individual Redis servers
2) Combine the individual servers into a Redis cluster.
3) Run workload A sending traffic to a random node
4) Scale-up the topology by periodically adding new nodes

#### Observations

* Redis does not support load distributed. The additional servers remain idle, albeit joining the cluster.

### 6. Scale-up Conditional

1) Start a set of individual Redis servers
2) Combine the individual servers into a Redis cluster.
3) Stress the topology by periodically adding new clients sending traffic to a random node
4) Scale-up the topology when the tail-latency goes above a given threshold

#### Observations

* Redis does not support load distributed. The additional servers remain idle, albeit joining the cluster.

### 7. Elastic Scale-down (Delete)

1) Start a set of individual Redis servers
2) Combine the individual servers into a Redis cluster.
3) Run workload A sending traffic to a random node
4) Scale-up the topology by adding 5 more services (group a).
5) Scale-up the topology by adding 5 more services (group b).
6) Scale-down the topology by **deleting** services in group a  (remove it from the Kubernetes API)

#### Observations

* Sometimes group B cannot join the cluster and fails.

### 8. Elastic Scale-down (Stop)

1) Start a set of individual Redis servers
2) Combine the individual servers into a Redis cluster.
3) Run workload A sending traffic to a random node
4) Scale-up the topology by adding new nodes
5) Scale-down the topology by periodically **stopping** some nodes (run command within target container to drain the
node)

#### **Observations**

### 9. Elastic Scale-down (Kill)

1) Start a set of individual Redis servers
2) Combine the individual servers into a Redis cluster.
3) Run workload A sending traffic to a random node
4) Scale-up the topology by adding new nodes
5) Scale-down the topology by periodically **killing**  some nodes (use Chaos-Mesh to forcibly kill the container)

#### Observations

* Sometimes group "more-servers" cannot join the cluster and fails.

### 10. Availability Failover (Single)

1) Start a primary Redis database server
2) Start a replicated Redis database server
3) Start a failover manager
4) Hammer the server with requests
5) Cause partition A on the 3rd minute, lasting 2 minutes
6) Cause partition B on the 6th minute, lasting 1 minute

#### Observations

* The client starts with a few INSERT_ERROR. It then goes into INSERT.
* Albeit this is an indication of a race condition, the client retries the request and remains operational.
* The INSERT_ERROR reappear when we inject the network failure.

### TODO

#### Failure at hotspot

This experiment configures a cluster of 3 Redis nodes and sends traffic to one of them.

After 2 minutes, we inject a failure on the most used server.

Observations

* Not currently supported. We must change the macros to select services based on Grafana information.
