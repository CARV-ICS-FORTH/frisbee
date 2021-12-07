# The Frisbee Library for Kubernetes

Popular applications, provided by Frisbee, ready to launch on Kubernetes
using [Kubernetes Helm](https://github.com/helm/helm).

## TL;DR

```bash
$ helm repo add frisbee https://carv-ics-forth.github.io/frisbee/charts
$ helm search repo frisbee
$ helm install my-release frisbee/<chart>
```

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
kind: Workflow
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
            - { master: .service.master.any }

    # The sentinel is Redis failover manager. Notice that we can have multiple dependencies.
    - action: Service
      name: sentinel
      depends: { running: [ master, slave ] }
      service:
        
          templateRef: redis.single.sentinel
          inputs:
            - { master: .service.master.any }

    # Cluster creates a list of services that run a shared context. 
    # In this case, we create a cluster of YCSB loaders to populate the master with keys. 
    - action: Cluster
      name: "loaders"
      depends: { running: [ master ] }
      cluster:
        templateRef: ycsb.redis.loader
        inputs:
          - { server: .service.master.any, recordcount: "100000000", offset: "0" }
          - { server: .service.master.any, recordcount: "100000000", offset: "100000000" }
          - { server: .service.master.any, recordcount: "100000000", offset: "200000000" }

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
            macro: .service.master.any
          duration: "2m"

    # Here we repeat the partition, a few minutes after the previous fault has been recovered.
    - action: Chaos
      name: partition1
      depends: { running: [ master, slave ], success: [ partition0 ], after: "6m" }
      chaos:
        type: partition
        partition:
          selector: { macro: .service.master.any }
          duration: "1m"

  # Here we declare the Grafana dashboards that Workflow will make use of.
  withTelemetry:
    importMonitors: [ "platform.telemetry.container", "ycsb.telemetry.client",  "redis.telemetry.server" ]

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


