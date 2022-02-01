# Examples

## List of Experiments

#### Baseline Single

This experiments runs a sequence of YCSB workloads (A,B, C,F, D, E) to a single cockroach databases.

#### Baseline Cluster Deterministic

This experiments runs a sequence of YCSB workloads (A,B, C,F, D, E) to a cluster of cockroach databases.

All the traffic goes to masters-0.

Observations

*
    1. If we run the load with 400 threads then the entire system blocks after 66.2K requests.

*
    2. This behavior is not reproducible on baseline-single.

#### Baseline Cluster Deterministic-OutOfOrder

This experiments runs a sequence of YCSB workloads (A,B, C,F, D, E) to a cluster of cockroach databases.

All the traffic goes to masters-2.

Observations

*
    1. The experiment blocks after 65K requests.
*
    2. Beware not to include invalid server names on the inputs. The failure is not detected.

#### Baseline Cluster Non-Deterministic

This experiments runs a sequence of YCSB workloads (A,B, C,F, D, E) to a cluster of cockroach databases.

The traffic goes to different database instances.

#### Cluster Elastic Scale-up

This experiments runs YCSB workload A (A,B, C,F, D, E) to an initial cluster of cockroach databases. The cluster
periodically scales-up by adding new nodes.

Observations

* The experiment requires every service to set the "advertise host" equal to the pod's name.
* The experiment blocks arbitrarily -- sometimes at 387K keys, sometimes at 900K.

## Observations
 



   