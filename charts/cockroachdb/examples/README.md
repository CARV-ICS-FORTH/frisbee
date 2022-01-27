# Examples



## List of Experiments



#### Baseline Single

This experiments runs a sequence of YCSB workloads (A,B, C,F, D, E) to a single  cockroach databases.



#### Baseline Cluster Deterministic 

This experiments runs a sequence of YCSB workloads (A,B, C,F, D, E) to a cluster of cockroach databases.

All the traffic goes to masters-0.


Observations
* 1. If we run the load with 400 threads then the entire system blocks after 66.2K requests.

* 2. This behavior is not reproducible on baseline-single.

#### Baseline Cluster Deterministic-OutOfOrder

This experiments runs a sequence of YCSB workloads (A,B, C,F, D, E) to a cluster of cockroach databases.

All the traffic goes to masters-2.

* 1. The experiment blocks after 65K requests. 


#### Baseline Cluster Non-Deterministic 

This experiments runs a sequence of YCSB workloads (A,B, C,F, D, E) to a cluster of cockroach databases.

The traffic goes to different database instances.



## Observations

* Beware not to include invalid server names on the inputs. The failure is not detected. 

#### Deadlock 



   

#### Server selection affects the experiment

1) If workloadA goes to serverA, and workloadB to serverB, the experiment blocks.
2) If all workloads go to the same server, the experiment proceeds.