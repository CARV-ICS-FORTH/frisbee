## Guide for the Frisbee Chart Developers


<!-- toc -->
  - [What is a Helm Chart ?](#what-is-a-helm-chart-)
      - [Lint Charts](#lint-charts)
      - [Working with MicroK8s’ built-in registry](#working-with-microk8s-built-in-registry)
  - [Run a Test](#run-a-test)
    - [Debugging an Installation](#debugging-an-installation)
  - [Debug a Test](#debug-a-test)
  - [Write a test](#write-a-test)
- [Change the Code](#change-the-code)
  - [Run a test](#run-a-test-1)
      - [Step 1:  Install Dependencies](#step-1--install-dependencies)
      - [Step 2: Update Helm repo](#step-2-update-helm-repo)
      - [Step 3:](#step-3)
      - [This step will install the following components:](#this-step-will-install-the-following-components)
      - [Step 4:  Install the testing components](#step-4--install-the-testing-components)
      - [Step 5: Run the Test Plan](#step-5-run-the-test-plan)
- [Observe a Testplan](#observe-a-testplan)
    - [Kubernetes Dashboard](#kubernetes-dashboard)
      - [Controller Logs](#controller-logs)
    - [Grafana Dashboard &amp; Alerts](#grafana-dashboard--alerts)
- [HELM](#helm)
  - [Baseline](#baseline)
      - [YCSB](#ycsb)
      - [FIO](#fio)
  - [Stress](#stress)
  - [Scaleout](#scaleout)
  - [Elasticity](#elasticity)
  - [Chaos](#chaos)
  - [Guide for the Frisbee Plan Developers](#guide-for-the-frisbee-plan-developers)
- [Periodically kill some nodes.](#periodically-kill-some-nodes)
    - [Do not set dependencies on cascades](#do-not-set-dependencies-on-cascades)

<!-- /toc -->


This is a guide for those who wish to contribute new Charts in Frisbee.

Because there is an overlap, we advise you to have a look at the Guide for Code Developers first.

## What is a Helm Chart ?

Helm is a package manager for Kubernetes.

Helm uses **a packaging format called charts**. A chart is a collection of files that describe a related set of
Kubernetes resources. A single chart might be used to deploy something simple, like a memcached pod, or something
complex, like a full web app stack with HTTP servers, databases, caches, and so on.

The best way to start is by reading the official [Helm documentation](https://helm.sh/docs/).

Control how to visualize the services.

```
// DrawAs hints how to mark points on the Grafana dashboard.
DrawAs string = "frisbee.io/draw/"

// DrawAsPoint will mark the creation and deletion of a service as distinct events.
DrawAsPoint string = "pointInTime"
// DrawAsRegion will draw a region starting from the creation of a service and ending to the deletion of the service.
DrawAsRegion string = "timeRegion"
```

#### Lint Charts

```bash
yamllint ./platform/Chart.yaml
```

```
docker run quay.io/helmpack/chart-testing:lates ct lint --target-branch=main --check-version-increment=false
```

#### Working with MicroK8s’ built-in registry

```bash
# Install the registry
microk8s enable registry

# To upload images we have to tag them with localhost:32000/your-image before pushing them:
docker build . -t localhost:32000/mynginx:registry

# Now that the image is tagged correctly, it can be pushed to the registry:
docker push localhost:32000/mynginx
```

Pushing to this insecure registry may fail in some versions of Docker unless the daemon is explicitly configured to
trust this registry.

To address this we need to edit `/etc/docker/daemon.json` and add:

```json
{
"insecure-registries": [
"localhost:32000"
]
}
```

The new configuration should be loaded with a Docker daemon restart:

```bash
sudo systemctl restart docker
```

Source: https://microk8s.io/docs/registry-built-in

## Run a Test

The easiest way to begin with is by have a look at the examples. Let's assume you are interested in testing TiKV.

```bash
>> cd examples/tikv/
>> ls
templates  plan.baseline.yml  plan.elasticity.yml  plan.saturation.yml  plan.scaleout.yml
```

You will some `plan.*.yml` files, and a sub-directory called `templates`.

* **Templates:** are libraries of frequently-used specifications that are reusable throughout the testing plan.
* **Plans:** are lists of actions that define what will happen throughout the test.

In general, a plan may dependent on numerous templates, and the templates depend on other templates.

To run the test will all the dependencies satisfied:

```bash
# Create a dedicated Frisbee name
>> kubectl create namespace karvdash-fnikol

# Run the test
>> kubectl -n karvdash-fnikol apply -f ../core/observability/ -f ../core/ycsb/ -f ./templates/ -f templates/telemetry/ -f plan.baseline.yml
```

For TikV, the dependencies are:

* `./templates/` : TiKV servers and clients
* `./templates/telemetry` : Telemetry for TiKV servers (TiKV-specific metrics)

* `examples/templates/core/observability` : Telemetry for TiKV containers (system-wise metrics)
* `examples/templates/core/ycsb` : Telemetry for TiKV clients (YCSB-specific metrics)

> BEWARE:
>
> 1) flag `-f` does not work recursively. You must explicitly declare the telemetry directory.
> 2) If you modify a template, you must re-apply it

### Debugging an Installation

Error: UPGRADE FAILED: unable to recognize "": no matches for kind "Template" in version "frisbee.io/v1alpha1"

In the next step, you should validate that CRDs are successfully installed.

```bash
# Validate the CRDs are properly installed
>> kubectl get crds  | grep frisbee.io
```

chaos.frisbee.io 2021-12-17T12:30:06Z clusters.frisbee.io 2021-12-17T12:30:06Z services.frisbee.io 2021-12-17T12:30:07Z
telemetries.frisbee.io 2021-12-17T12:30:07Z workflows.frisbee.io 2021-12-17T12:30:07Z

## Debug a Test

At this point, the workflow is installed. You can go to the controller's terminal and see some progress.

If anything fails, you will it see it from there.

```
$ sudo curl -s https://raw.githubusercontent.com/ncarlier/webhookd/master/install.sh | bash
```

## Write a test

helm install --dry-run --debug --dependency-update ./ ../observability/

https://github.com/helm/chartmuseum

https://medium.com/@maanadev/how-set-up-a-helm-chart-repository-using-apache-web-server-670ffe0e63c7

Deploy as helm chart helm repo index ./ --url https://carv-ics-forth.github.io/frisbee

# Change the Code

````
The easiest way to begin with is by have a look at the examples. It consists of two sub-directories:

* **Templates:** are libraries of frequently-used specifications that are reusable throughout the testing plan.
* **Testplans:** are lists of actions that define what will happen throughout the test.

We will use the `examples/testplans/3.failover.yml` as a reference.

This plans uses the following templates:

* `examples/templates/core/sysmon.yml`
* `examples/templates/redis/redis.cluster.yml`
* `examples/templates/ycsb/redis.client.yml`

Because these templates are deployed as Kubernetes resources, they are references by name rather than by the relative
path.

This is why we need to have them installed before running the experiment. (for installation instructions
check [here](docs/singlenode-deployment.md).)

```yaml
# Standard Kubernetes boilerplate
apiVersion: frisbee.io/v1alpha1
kind: TestPlan
metadata:
name: redis-failover
spec:

# Here we specify the workflow as a directed-acyclic graph (DAG) by specifying the dependencies of each action.
actions:
# Service creates an instance of a Redis Master
# To create the instance we use the redis.single.master with the default parameters.
- action: Service
name: master
service:

templateRef: redis.single.master

# This action is same as before, with two additions.
# 1. The `depends' keyword ensure that the action will be executed only after the `master' action
# has reached a Running state.
# 2. The `inputs' keyword initialized the instance with custom parameters.
- action: Service
name: slave
depends: { running: [ master ] }
service:

templateRef: redis.single.slave
inputs:
- { master: .service.master.one }

# The sentinel is Redis failover manager. Notice that we can have multiple dependencies.
- action: Service
name: sentinel
depends: { running: [ master, slave ] }
service:

templateRef: redis.single.sentinel
inputs:
- { master: .service.master.one }

# Cluster creates a list of services that run a shared context.
# In this case, we create a cluster of YCSB loaders to populate the master with keys.
- action: Cluster
name: "loaders"
depends: { running: [ master ] }
cluster:
templateRef: ycsb.redis.loader
inputs:
- { server: .service.master.one, recordcount: "100000000", offset: "0" }
- { server: .service.master.one, recordcount: "100000000", offset: "100000000" }
- { server: .service.master.one, recordcount: "100000000", offset: "200000000" }

# While the loaders are running, we inject a network partition fault to the master node.
# The "after" dependency adds a delay so to have some keys before injecting the fault.
# The fault is automatically retracted after 2 minutes.
- action: Chaos
name: partition0
depends: { running: [ loaders ], after: "3m" }
chaos:
type: partition
partition:
selector:
macro: .service.master.one
duration: "2m"

# Here we repeat the partition, a few minutes after the previous fault has been recovered.
- action: Chaos
name: partition1
depends: { running: [ master, slave ], success: [ partition0 ], after: "6m" }
chaos:
type: partition
partition:
selector: { macro: .service.master.one }
duration: "1m"

# Here we declare the Grafana dashboards that Workflow will make use of.
withTelemetry:
importDashboards: [ "system.telemetry.agent", "ycsb.telemetry.client",  "redis.telemetry.server" ]

```

# Run the experiment

Firstly, you'll need a Kubernetes deployment and `kubectl` set-up

* For a single-node deployment click [here](docs/singlenode-deployment.md).

* For a multi-node deployment click [here](docs/cluster-deployment.md).

In this walk-through, we assume you have followed the instructions for the single-node deployment.

In one terminal, run the Frisbee controller.

If you want to run the webhooks locally, you’ll have to generate certificates for serving the webhooks, and place them
in the right directory (/tmp/k8s-webhook-server/serving-certs/tls.{crt,key}, by default).

_If you’re not running a local API server, you’ll also need to figure out how to proxy traffic from the remote cluster
to your local webhook server. For this reason, we generally recommend disabling webhooks when doing your local
code-run-test cycle, as we do below._

```bash
# Run the Frisbee controller
>>  make run ENABLE_WEBHOOKS=false
```

We can use the controller's output to reason about the experiments transition.

On the other terminal, you can issue requests.

```bash
# Create a dedicated Frisbee name
>> kubectl create namespace frisbee

# Run a testplan (from Frisbee directory)
>> kubectl -n frisbee apply -f examples/testplans/3.failover.yml
workflow.frisbee.io/redis-failover created

# Confirm that the workflow is running.
>> kubectl -n frisbee get pods
NAME         READY   STATUS    RESTARTS   AGE
prometheus   1/1     Running   0          12m
grafana      1/1     Running   0          12m
master       3/3     Running   0          12m
loaders-0    3/3     Running   0          11m
slave        3/3     Running   0          11m
sentinel     1/1     Running   0          11m


# Wait until the test oracle is triggered.
>> kubectl -n frisbee wait --for=condition=oracle workflows.frisbee.io/redis-failover
...
```

## How can I understand what happened ?

One way, is to access the workflow's description

```bash
>> kubectl -n frisbee describe workflows.frisbee.io/validate-local
```

But why bother if you can access Grafana directly ?
````

````
# Frisbee in a Nutshell

This tutorial introduces the basic functionalities of Frisbee:

- **Write tests:**  for stressing complex topologies and dynamic operating conditions.
- **Run tests:**  provides seamless scaling from a single workstation to hundreds of machines.
- **Debug tests:**  through extensive monitoring and comprehensive dashboards.

For the rest of this tutorial we will use the Frisbee package of TiKV key/value store.

#### Frisbee Installation

Before anything else, we need to install the Frisbee platform and the Frisbee packages for testing.

```

```


Then you have to go install the Frisbee system.

```bash
>> cd charts/frisbee/charts/frisbee
>> helm install frisbee ./ --dependency-update
```

You will see a YAML output that describe the components to be installed.

#### Run the controller

After the Frisbee CRDs and their dependencies are installed, you can start running the Frisbee controller.

From the project's directory run

```bash
>> make run
```

#### Modify an examples test

Perhaps the best way is to modify an existing test. We use the `iperf` benchmark as a reference.

From another terminal (do not close the controller), go to

```bash
>> cd charts/iperf/
```

You will see the following files:

* **templates**:  libraries of frequently-used specifications that are reusable throughout the testing plan.
* **plans**: lists of actions that define what will happen throughout the test.
* **dashboards**: application-specific dashboards for Grafana

Because templates are used by the plans, we must have them installed before the running the tests.

```bash
>> helm install iperf ./ --dependency-update
```

Then, run the test to become familiar with the procedure.

```bash
>> kubectl apply -f plans/plan.validate-network.yml
```

If everything works fine, you will see the logs flowing in the **controller**.

Then you will get a message like

> Error: found in Chart.yaml, but missing in charts/ directory: chaos-mesh, openebs


This is because the Frisbee installation are not yet downloaded. To get them automatically run the previous command
with`--dependency-update` flag. Also, remove the `--dry-run` run to execute the actual installation.

```bash
>>  helm install frisbee ./ --dependency-update
```

Run the controller

== Operatator-SDK

=== Create a new Controller

* operator-sdk create api --group frisbee --version v1alpha1 --kind MyNewController --resource --controller
* operator-sdk create webhook --group frisbee --version v1alpha1 --kind MyNewController --defaulting
--programmatic-validation

docker save frisbee:latest -o image.tar
````

== Notes ==

1) Cadvisor does not support for NFS mounts.
2) Check how we can use block devices

## Run a test

#### Step 1:  Install Dependencies

Make sure that [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl-linux/)
and are installed on your system, and that you have access to a Kubernetes
installation.

* **Local Installation** If you want a local installation you can use


* **Remote Installation**: Set  `~/.kube/config` appropriately, and create tunnel for sending requests to Kubernetes
API.

```bash
# Create tunnel for sending requests to Kubernetes API.
>> ssh -L 6443:192.168.1.213:6443 [USER@]SSH_SERVER
```

#### Step 2: Update Helm repo

#### Step 3:

#### This step will install the following components:

* Frisbee CRDS
* Frisbee Controller
* Frisbee Dependency stack (e.g, Chaos toolkits, dynamic volume provisioning, observability stack)
* Ingress for making the observability stack accessible from outside the Kubernetes

By default the platform sets the Ingress to `localhost`.

If you use a non-local cluster, you can these the ingress via the  `global.ingress` flag.

```bash
# Install the platform with non-local ingress
>> helm upgrade --install --wait my-frisbee frisbee/platform --set global.ingress=platform.science-hangar.eu
```

#### Step 4:  Install the testing components

```bash

```

#### Step 5: Run the Test Plan

This url points
to : https://raw.githubusercontent.com/CARV-ICS-FORTH/frisbee/main/charts/tikv/examples/plan.baseline.yml

```bash
# Create a plan
>> curl -sSL https://tinyurl.com/t3xrtmny | kubectl -f - apply
```

# Observe a Testplan

### Kubernetes Dashboard

If you use a microk8s installation of Kubernetes, then the procedure is slightly different.

```bash
# Deploy the dashboard
>> microk8s dashboard-proxy

# Start the dashboard
>> microk8s dashboard-proxy

# Now access Dashboard at:
https://localhost:10443
```

#### Controller Logs

The logs of the controller are accessible by the terminal on which the controller is running.

### Grafana Dashboard & Alerts

Grafana is a multi-platform open source analytics and interactive visualization web application.

To access it, use the format `http://grafana.${INGRESS}` where `Ingress` is the value you defined in step 3.

For example,

```bash
# Access Grafana via your browser
http://grafana.platform.science-hangar.eu
```

Optionally, validate that everything works.

```bash
# Deploy a hello world
>>  kubectl create deployment hello-node --image=k8s.gcr.io/echoserver:1.4
deployment.apps/hello-node created

# Verify that a hell-node deployment exists
>> kubectl get deployments
NAME         READY   UP-TO-DATE   AVAILABLE   AGE
hello-node   1/1     1            1           36s

# Delete the deployment
>> kubectl delete deployments hello-node
deployment.apps "hello-node" deleted
```

This step will install the Frisbee CRDs and all the necessary tools.

```bash
# Update Helm repo
>> helm repo add frisbee https://carv-ics-forth.github.io/frisbee/charts

# Install the platform with local ingress
>> helm upgrade --install --wait my-frisbee frisbee/platform
```

KUBECONFIG="/home/fnikol/.kube/config.evolve.admin" make run

ahelm upgrade --install --wait my-frisbee ./charts/platform/ --debug --set operator.enabled=false -f
./charts/platform/values-baremetal.yaml
ehelm upgrade --install --wait my-system ./charts/system --debug


Cool hacks:

https://github.com/antrea-io/antrea/tree/main/hack


# HELM


Create a public Helm chart repository with GitHub Pages

What is Helm and a Helm chart repository

Helm, as per official claim, is “The package manager for Kubernetes”.

Indeed Helm really helps handling application deployments on Kubernetes, not only because it simplifies the application
release phase but also because Helm makes possible to easily manage variables in the Kubernetes manifest YAML files.

Once the charts are ready and you need to share them, the easiest way to do so is by uploading them to a chart
repository. A Helm chart repository is where we host and share Helm packages and any HTTP server will do. Unluckily Helm
does not include natively a tool for uploading charts to a remote chart repository server (Helm can serve the local
repository via $ helm serve though).

We’ll take advantage of GitHub Pages for the purpose to share our charts.

What is GitHub Pages

GitHub Pages is a static site hosting service provided to you by GitHub, designed to host your pages directly from a
GitHub repository. GitHub Pages is a perfect match for our purpose, since everything we need is to host a single
index.yaml file along with a bunch of .tgz files.

Why not hosting all this stuff in your own web server? A managed service helps to reduce your operational overhead and
risk (think to monitoring, patch management, security, backup services…) so you can focus on code and in what makes your
business relevant for your customers.

Useful links:
https://medium.com/@mattiaperi/create-a-public-helm-chart-repository-with-github-pages-49b180dbb417
https://github.com/technosophos/tscharts


A **Test Plan** documents the strategy that will be used to verify a specific test for
a [software](https://en.wikipedia.org/wiki/Software) product.

The plan typically contains a detailed understanding of the eventual [workflow](https://en.wikipedia.org/wiki/Workflow)
that models the deployment scenario for a software product, the test strategy, the resources required to perform
testing, and the Key Performance Indicators required to ensure that a product or system meets its design specifications
and other requirements.

By codifying the test plan in a YAML-based syntax, Frisbee carriers three main benefits to teams:

1. Help people outside the test team such as developers, business managers, customers **understand** the details of
testing.

2. Test Plan **guides** our thinking. It is like a rule book, which needs to be followed.

3. Important aspects like test estimation, test
scope,[Test Strategy](https://www.guru99.com/how-to-create-test-strategy-document.html)are **documented** in Test
Plan, so it can be reviewed by Management Team and re-used for other projects.

A test plan may include a strategy for one or more of the following:

* Baseline: to be performed during the development or approval stages of the product, typically on a small sample of
units.
* Stress: to be performed during preparation or assembly of the product, in an ongoing manner for purposes of
performance verification and quality control.

## Baseline

A baseline is **a fixed point of reference that is used for comparison purposes**.

#### YCSB

Cloud Serving Benchmark (YCSB) is **an open-source specification and program suite for evaluating retrieval and
maintenance capabilities of computer programs**. It is often used to compare relative performance of NoSQL database
management systems.

All six workloads have a data set which is similar. Workloads D and E insert records during the test run.

Thus, to keep the database size consistent, we apply the following sequence:

0. Bootstrap the database.

1. Load the database, using workload A's parameter file (workloads/workloada) and the "-load" switch to the client.
2. Run workload A (using workloads/workloada and "-t") for a variety of throughputs.
3. Run workload B (using workloads/workloadb and "-t") for a variety of throughputs.
4. Run workload C (using workloads/workloadc and "-t") for a variety of throughputs.
5. Run workload F (using workloads/workloadf and "-t") for a variety of throughputs.
6. Run workload D (using workloads/workloadd and "-t") for a variety of throughputs. This workload inserts records,
increasing the size of the database.
7. Delete the data in the database. Otherwise, the remaining data of the cluster might affect the results of the
following workload. For the deletion, instead of destroying the cluster, we destroy and recreate the cluster.
8. Reload the database, using workload E's parameter file (workloads/workloade) and the "-load switch to the client.
9. Run workload E (using workloads/workloadd and "-t") for a variety of throughputs. This workload inserts records,
increasing the size of the database.

In general, these steps remain the same for the various databases. The difference is how we bootstrap each database.

#### FIO

To *benchmark* persistent disk performance, use *FIO* instead of other disk *benchmarking* tools such as dd .

Fio spawns a number of threads or processes doing a particular type of I/O action as specified by the user.

## Stress

## Scaleout

## Elasticity

## Chaos


## Guide for the Frisbee Plan Developers

<!-- toc -->

<!-- /toc -->

* Spurious Alert may be risen if the expr evaluation frequency is less than the scheduled interval.
* In this case, Grafana faces an idle period, and raises a NoData Alert.
* The controller ignores such messages.

# Periodically kill some nodes.
- action: Cascade name: killer depends: { running: [ clients ] } cascade:
templateRef: system.chaos.pod.kill instances: 3 inputs:
- { target: .cluster.clients.one }

This can be wrong because Frisbee selects a single client -- and will be used 3 times, without error. Instead, we must
use as many inputs as the number of instances -- or omit instances.

In general, because when you use one input for multiple instances.

Macros select only Running objects

### Do not set dependencies on cascades

This is because Kills are always running -- therefore cascades that involve kill actions are always running