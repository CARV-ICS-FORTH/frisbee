# Examples

The examples are incremental and are tailed to drive you through the various capabilities of Frisbee.



To emphasize the general nature of Frisbee, we designed the examples to cover various distributed systems, including:

* **Iperf**: network benchmark
* **CockroachDB**: distributed databases
* **Flower**: federated learning frameworks



Before going further, please install the respective packages.

```bash
>> helm upgrade --install --wait my-iperf2 ./charts/iperf2/ --debug

>> helm upgrade --install --wait my-cockroach ./charts/cockroachdb --debug

>> helm upgrade --install --wait my-ycsb ./charts/ycsb/ --debug

>> helm upgrade --install --wait my-flower ./charts/flower --debug
```





## Example 1: Dependencies

A simple server-client deployment. So familiar, but so distant when you consider the initialization order, the name discovery, etc etc !

The example shows how you can consistently start one dependent service after another. 



## Example 2: Stages





