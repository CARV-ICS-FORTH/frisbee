# Tutorial

This tutorial will guide you through deploying and running Frisbee on a local Kubernetes installation.

# Install Dependencies

#### 1. microk8s

[Microk8s](https://microk8s.io/docs)  is the simplest production-grade conformant K8s.  **It runs entirely on your
workstation or edge device.**

```bash
# Install microk8s v.1.22
>> sudo snap install microk8s --classic --channel=1.22/stable

# Create kubectl alias 
>> sudo snap alias microk8s.kubectl kubectl

# Use microk8s config as the default kubernetes config
>> microk8s config > ~/.kube/config

# Enable Dependencies
>> microk8s enable dns ingress helm3

# Start microk8s
>> microk8s start
```

#### 2. Helm

[Helm](https://helm.sh/docs/intro/install/)  is a package manager for Kubernetes. Helm uses **a packaging format called
charts**.

A chart is a collection of files that describe a related set of Kubernetes resources.

```bash
>> sudo snap install helm --classic
```

#### 3. Frisbee platform

Although Frisbee can be installed directly from a Helm repository, for demonstration purposes we favor the git-based
method.

```bash
# Download the source code
>> git clone git@github.com:CARV-ICS-FORTH/frisbee.git

# Move to the Frisbee project
>> cd frisbee

# Have a look at the installation configuration
>> less charts/platform/values.yaml 
```

> **Note:** Make sure that the dir "/mnt/local" exists. The error will not appear until the execution of the test.



Now, it's time to deploy the platform, on the **default** namespace.

```bash
# Wait until the installation is complete
>> helm  upgrade --install --wait my-frisbee ./charts/platform/ --debug -n default
```

If everything works normally, you should be able to access the following **dashboards**:

* [Dashboard](https://dashboard-frisbee.localhost) is a web-based Kubernetes user interface. You can use Dashboard to
  deploy containerized applications to a
  Kubernetes cluster, troubleshoot your containerized application, and manage the cluster resources.
* [Chaos Dashboard](http://chaos-frisbee.localhost)  is a one-step web UI for managing, designing, and monitoring chaos
  experiments on *Chaos Mesh*.

> **Note:** Both dashboards will ask for config or token. In that case, copy the token from .kube/config file.

```bash
>> grep token  ~/.kube/config
```

Now are ready to deploy the tests.

## Testing a System

Before running any test, we need to install the System Under Testing (SUT).

As a reference, we will use the Frisbee chart
for [CockroachDB](https://github.com/CARV-ICS-FORTH/frisbee/tree/main/charts/cockroachdb)

> [*CockroachDB*](https://github.com/cockroachdb/cockroach) is a distributed database with standard SQL for cloud
> applications.

#### 1. Prepare a namespace for the SUT

Firstly, we need to create a dedicated namespace for the test.

The different namespaces allows us to run multiple tests in parallel.

However, because templates are isolated to the namespace they are installed to, we must install the system templates to
the testing namespace.

We combine the creation of the namespace and the installation of system templates (e.g, telemetry, chaos) in one
command.

```bash
>> helm upgrade --install --wait my-system ./charts/system --debug -n mytest --create-namespace
```

#### 2. Deploy the SUT

The commands are to be executed from the *Frisbee* directory.

```bash
# Install Cockroach servers
>> helm upgrade --install --wait my-cockroach ./charts/cockroachdb --debug -n mytest

# Install YCSB for creating workload
>> helm upgrade --install --wait my-ycsb ./charts/ycsb --debug -n mytest
```

Then you can verify that all the packages are successfully installed

```bash
>> helm list
NAME            NAMESPACE       REVISION        UPDATED                                         STATUS          CHART             
my-frisbee      default         1               2022-06-10 20:37:26.298297945 +0300 EEST        deployed        platform-0.0.0 


>> helm list -n mytest
NAME            NAMESPACE       REVISION        UPDATED                                         STATUS          CHART            
my-cockroach    mytest          1               2022-06-10 20:40:29.741398162 +0300 EEST        deployed        cockroachdb-0.0.0 
my-system       mytest          1               2022-06-10 20:40:19.981077906 +0300 EEST        deployed        defaults-0.0.0   
my-ycsb         mytest          1               2022-06-10 20:40:36.97639544 +0300 EEST         deployed        ycsb-0.0.0       
```

> **Note:** if you modify the templates of a chart you must re-install it. examples can be modified without
> re-installation.

#### Run a Test

You now select which scenario you wish to run.

```bash
>> ls -1a ./charts/cockroachdb/examples/
...
10.bitrot.yml
11.network.yml
12.bitrot-logs.yml
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
>> kubectl -f ./charts/cockroachdb/examples/12.bitrot-logs.yml apply -n mytest

persistentvolumeclaim/shared-dir created
testplan.frisbee.io/cockroach-bitrot-logs created
```

#### Observe a Test

*Frisbee* provides 3 methods for observing the progress of a test.

* **Event-based:** Consumes information from the Kubernetes API

    * [Dashboard](https://dashboard-frisbee.localhost/#/pod?namespace=mytest)
    * [Chaos Dashboard](http://chaos-frisbee.localhost/experiments)
    * `kubectl get events`

* **Metrics-based:** Consumes information from distributed performance metrics.

    * [Prometheus](http://prometheus-mytest.localhost)
    * [Grafana](http://grafana-mytest.localhost/d/crdb-console-runtime/crdb-console-runtime)

* **Log-based:** Consumes information from distributed logs.

    * [Logviewer](http://logviewer-mytest.localhost) (admin/admin)

        

>
> You may notice that it takes **long time for the experiment to start**. This is due to preparing the NFS volume for collecting the logs from the various services.  Also note that the lifecycle of the volume is bind to that of the test. If the test is deleted, the volume will be garbage collected automatically.



#### Pass/Fail a Test

The above tools are for understanding the behavior of a system, but do not help with test automation.

Besides the visual information, we need something that can be used in external scripts.

We will use `kubectl` since is the most common CLI interface between Kubernetes API and third-party applications.

Firstly, let's inspect the test plan.

```bash
>> kubectl describe testplan.frisbee.io/cockroach-bitrot-logs -n mytest

...
Status:
  Conditions:
    Last Transition Time:  2022-06-10T20:05:07Z
    Message:               failed jobs: [bitrot]
    Reason:                JobHasFailed
    Status:                True
    Type:                  UnexpectedTermination
  Executed Actions:
    Bitrot:
    Boot:
    Import - Workload:
    Logviewer:
    Masters:
  Grafana Endpoint:     grafana-mytest.localhost
  Message:              failed jobs: [bitrot]
  Phase:                Failed
  Prometheus Endpoint:  prometheus-mytest.localhost
  Reason:               JobHasFailed
```

We are interested in the `Phase` and `Conditions` fields that provides information about the present status of a test.
The **Phase** describes the lifecycle of a Test.

|  Phase  |                         Description                          |
| :-----: | :----------------------------------------------------------: |
|   ""    |      The request is not yet accepted by the controller       |
| Pending | The request has been accepted by the Kubernetes cluster, but one of the child jobs has not been created. This includes the time waiting for logical dependencies, Ports discovery, data rewiring, and placement of Pods. |
| Running | All the child jobs  have been created, and at least one job is still running. |
| Success |              All jobs have voluntarily exited.               |
| Failed  | At least one job of the CR has terminated in a failure (exited with a  non-zero exit code or was stopped by the system). |

#### 

The **Phase** is a top-level description calculated based on some **Conditions**. The **Conditions** describe the
various stages the Test has been through.

|       Condition       |                     Description                     |
| :-------------------: | :-------------------------------------------------: |
|      Initialized      |          The workflow has been initialized          |
|  AllJobsAreScheduled  |     All jobs have been successfully scheduled.      |
|  AllJobsAreCompleted  |     All jobs have been successfully completed.      |
| UnexpectedTermination | At least job that has been unexpectedly terminated. |

To avoid continuous inspection via polling, we use the `wait` function of `kubectl`.

In the specific **bitrot** scenario, the test will pass only it has reached an **UnexpectedTermination** within 10
minutes of execution.

```bash
>> kubectl wait --for=condition=UnexpectedTermination --timeout=10m testplan.frisbee.io/cockroach-bitrot-logs -n mytest

testplan.frisbee.io/cockroach-bitrot-logs condition met
```

Indeed, the condition is met, meaning that the test has failed. We can visually verify it from
the [Dashboard](https://dashboard-frisbee.localhost) .

![image-20220525170302089](tutorial.assets/image-20220525170302089.png)

To reduce the noise when debugging a failed test, *Frisbee* automatically deletes all the jobs, expect for the failed
one (masters-1), and the telemetry stack (grafana/prometheus).



> If the condition is not met within the specified timeout, `kubectl` will exit with failure code (1) and the following
> error message:
>
> "error: timed out waiting for the condition on testplans/cockroach-bitrot"

#### Delete a Test

The deletion is as simple as the creation of a test.

```bash
>> kubectl -f ./charts/cockroachdb/examples/12.bitrot-logs.yml -n mytest delete --cascade=foreground

persistentvolumeclaim "shared-dir" deleted
testplan.frisbee.io "cockroach-bitrot-logs" deleted
```

The flag `cascade=foreground` will wait until the experiment is actually deleted. Without this flag, the deletion will
happen in the background. Use this flag if you want to run sequential tests, without interference.



## Parallel Tests.

For the time being, the safest to run multiple experiments is to run each test on a **dedicated namespace**.

To do so, you have to repeat this tutorial, replacing the  `-n ....`  flag with a different namespace.

