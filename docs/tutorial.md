# Tutorial



This tutorial will guide you through deploying and running Frisbee on a local Kubernetes installation.



# Install Dependencies

#### microk8s

 [Microk8s](https://microk8s.io/docs)  is the simplest production-grade conformant K8s.  **It runs entirely on your workstation or edge device.**

```bash
# Install microk8s v.1.22
>> sudo snap install microk8s --classic --channel=1.22/stable

# Enable Dependencies
>> microk8s enable dns ingress

# Start microk8s
>> microk8s start
```



Configure `kubectl ` to point on microk8s.

```bash
# Create alias 
>> sudo snap alias microk8s.kubectl kubectl

# Use microk8s config as the default kubernetes config
>> microk8s config > ~/.kube/config
```



#### Helm

[Helm](https://helm.sh/docs/intro/install/)  is a package manager for Kubernetes. Helm uses **a packaging format called charts**. 

A chart is a collection of files that describe a related set of Kubernetes resources.

```bash
>> sudo snap install helm --classic
```



#### Frisbee platform

Although Frisbee can be installed directly from a Helm repository, for demonstration purposes we favor the git-based method.

```bash
# Download the source code
>> git clone git@github.com:CARV-ICS-FORTH/frisbee.git

# Move to the Frisbee project
>> cd frisbee

# Have a look at the installation configuration
>> less charts/platform/values.yaml 
```

> **Note:** Make sure that the dir "/mnt/local" exists.



Now, it's time to deploy the platform

```bash
# Wait until the installation is complete
>> helm  upgrade --install --wait my-frisbee ./charts/platform/ --debug

# Make sure that every is ok
>> helm list
```



If everything works normally, you should be able to access the following **dashboards**:

* [Dashboard](https://dashboard-frisbee.localhost) is a web-based Kubernetes user interface. You can use Dashboard to deploy containerized applications to a
  Kubernetes cluster, troubleshoot your containerized application, and manage the cluster resources.
* [Chaos Dashboard](http://chaos-frisbee.localhost)  is a one-step web UI for managing, designing, and monitoring chaos experiments on *Chaos Mesh*.

> **Note:** Both dashboards will ask for config or token. In that case, copy the token from .kube/config file.

```bash
>> grep token  ~/.kube/config
```



Now are ready to deploy the tests.



## Testing a System

Before running any test, but install the System Under Testing (SUT) in Frisbee.

We will use the Frisbee chart for [CockroachDB](https://github.com/CARV-ICS-FORTH/frisbee/tree/main/charts/cockroachdb)

> [*CockroachDB*](https://github.com/cockroachdb/cockroach) is a distributed database with standard SQL for cloud applications.



#### Deploy the SUT

The commands are to be executed from the *Frisbee* directory.

```bash
# Install Cockroach servers
>> helm upgrade --install --wait my-cockroach ./charts/cockroachdb --debug

# Install YCSB for creating workload
>> helm upgrade --install --wait my-ycsb ./charts/ycsb --debug
```



Then you can verify that all the packages are successfully installed

```bash
>> helm list
NAME            NAMESPACE       REVISION        UPDATED                                         STATUS          CHART             
my-cockroach    default         2               2022-05-25 16:15:58.682969153 +0300 EEST        deployed        cockroachdb-0.0.0 
my-frisbee      default         1               2022-05-25 15:46:54.4600888 +0300 EEST          deployed        platform-0.0.0  
my-ycsb         default         1               2022-05-25 16:16:13.364123735 +0300 EEST        deployed        ycsb-0.0.0      
```



#### Run a Test

You now select which scenario you wish to run. 

```bash
>> ls ./charts/cockroachdb/examples/
...
10.bitrot.yml
11.network.yml
12.withlogs.yml
1.baseline-single.yml
2.baseline-cluster-deterministic.yml
3.baseline-cluster-deterministic-outoforder.yml
4.baseline-cluster-nondeterministic.yml
5.scaleup-scheduled.yml
6.scaleup-conditional.yml
7.scaledown-delete.yml
8.scaledown-stop.yml
9.scaledown-kill.yml
```



Let's run a **bitrot** scenario.

```bash
>> kubectl -f ./charts/cockroachdb/examples/10.bitrot.yml apply
testplan.frisbee.io/cockroach-bitrot created
```



#### Observe a Test

*Frisbee* provides two methods for observing the progress of a test.

* **State-based:** Consumes information from the Kubernetes API 

  * [Dashboard](https://dashboard-frisbee.localhost) 
  * [Chaos Dashboard](http://chaos-frisbee.localhost) 

* **Metrics-based:** Consumes information from distributed performance metrics.

  * [Prometheus](http://prometheus-frisbee.localhost)

  * [Grafana](http://grafana-frisbee.localhost)

    

The above tools are for understanding the behavior of a system, but do not help with test automation.

Besides the visual information, we need something that can be used in external scripts.



We will use `kubectl` since is the most common CLI interface between Kubernetes API and third-party applications.

Firstly, let's inspect the test plan.

```bash
>> kubectl describe testplan.frisbee.io/cockroach-bitrot
```



> Status:
>   Conditions:
>     Last Transition Time:  2022-05-25T13:20:52Z
>     Message:               failed jobs: [masters]
>     Reason:                JobHasFailed
>     Status:                True
>     Type:                  UnexpectedTermination
>   Configuration:
>     Advertised Host:      frisbee-operator
>     Grafana Endpoint:     http://grafana:3000
>     Prometheus Endpoint:  http://prometheus:9090
>   Executed:
>     Bitrot:
>     Boot:
>     Import - Workload:
>     Masters:
>     Run - Workload:
>   Message:         failed jobs: [masters]
>   Phase:           Failed
>   Reason:          JobHasFailed
>   With Telemetry:  true



We are interested in the `Phase` and `Conditions` fields that provides information about the present status of a test.

* **Phase** describes the lifecycle of a Test. 
  * **""**:  the request is not yet accepted by the controller
  * **"Pending"**:  the request has been accepted by the Kubernetes cluster, but one of the child jobs has not been created. This includes the time waiting for logical dependencies, Ports discovery,  data rewiring, and placement of Pods.
  * **"Running"**: all the child jobs  have been created, and at least one job is still running.
  * **"Success"**: all jobs have voluntarily exited.
  * **"Failed"**:  at least one job of the CR has terminated in a failure (exited with a  non-zero exit code or was stopped by the system).
* The **Phase** is a top-level description calculated based on some **Conditions**. The **Conditions** describe the various stages the Test has been through.
  * **"Initialized"**:  the workflow has been initialized
  * **"AllJobsAreScheduled"** : all jobs have been successfully scheduled.
  * **"AllJobsAreCompleted"**:  all jobs have been successfully completed.
  * **"UnexpectedTermination"** : a least job that has been unexpectedly terminated.



#### Pass/Fail a Test

To avoid continuous inspection via polling, we use the `wait` function of `kubectl`.

In the specific **bitrot** scenario,  the test will pass only it has reached an **UnexpectedTermination** within 10 minutes of execution.

```bash
>> kubectl wait --for=condition=UnexpectedTermination --timeout=10m testplan.frisbee.io/cockroach-bitrot
testplan.frisbee.io/cockroach-bitrot condition met
```



Indeed, the condition is met, meaning that the test has failed. We can visually verify it from the [Dashboard](https://dashboard-frisbee.localhost) .



![image-20220525170302089](tutorial.assets/image-20220525170302089.png)



To reduce the noise when debugging a failed test, *Frisbee* automatically deletes all the jobs, expect for the failed one (masters-1), and the telemetry stack (grafana/prometheus). 



> If the condition is not met within the specified timeout, `kubectl` will exit with failure code (1) and the following error message:
>
> "error: timed out waiting for the condition on testplans/cockroach-bitrot"



#### Delete a Test

The deletion is as simple as the creation of a test.

```bash
>> kubectl -f ./charts/cockroachdb/examples/10.bitrot.yml delete --cascade=foreground
testplan.frisbee.io "cockroach-bitrot" deleted
```

The flag `cascade=foreground` will wait until the experiment is actually deleted. Without this flag, the deletion will
happen in the background. Use this flag if you want to run sequential tests, without interference.
