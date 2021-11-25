# Installation

This tutorial describes how to deploy Frisbee and start running tests.

# Requirements

Firstly, we need to install the requirements. For convenience, we have packaged them into Helm Charts.

## Installation

```bash
>> helm install my-deployment --create-namespace -n frisbee-testing ./
```

By default helm will use the Kubernetes configuration from `~/.kube/config`. If you wish to run Frisbee on a remote
cluster, use `--kubeconfig=~/.kube/config.remote`

#### Run the controller

At first, we need to run the Frisbee operator.

There are three ways to run the operator:

- As Go program outside a cluster
- As a Deployment inside a Kubernetes cluster
- Managed by
  the [Operator Lifecycle Manager (OLM)](https://sdk.operatorframework.io/docs/olm-integration/tutorial-bundle/#enabling-olm)
  in [bundle](https://sdk.operatorframework.io/docs/olm-integration/quickstart-bundle) format.

To run the controller outside the cluster.

```bash
# Run Frisbee controller outside a cluster (from Frisbee directory)
>> make run
```

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

#### Wait for completion

After you have successful ran a test, the next question that comes is about its completion.

```bash
# View deployed plans
>> kubectl -n karvdash-fnikol get workflows.frisbee.io
NAME            AGE
tikv-baseline   1m

# Inspect a plan
>> kubectl -n karvdash-fnikol describe workflows.frisbee.io tikv-baseline
```

Describe will return a lot of information about the plan. We are interested in the fields `conditions`.

We can use these fields to wait until they become true -- thus saving us from manual inspection.

```
# Wait until the test oracle is triggered.
>> kubectl  wait --for=condition=allActions workflows.frisbee.io/tikv-baseline -n karvdash-fnikol
workflow.frisbee.io/tikv-baseline condition met

```

#### Terminate a test

```bash
kubectl  delete -f tikv.elasticity.yml --cascade=foreground
```

The flag `cascade=foreground` will wait until the experiment is actually deleted. Without this flag, the deletion will
happen in the background.

Use this flag if you want to run multiple tests, without interference.

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

## Deploy a new Version

```bash
>> make release
>> git push --set-upstream origin $(git branch --show-current) && git push --tags
```

1. Go to GitHub and create a pull request

2. Merge pull request

3. Delete branch

4. Go to [GitHub Tags](https://github.com/CARV-ICS-FORTH/frisbee/tags ) and create a new release for the latest tag

# Run Frisbee Locally

## Before Starting

### Install Microk8s

*MicroK8s* is a CNCF certified upstream Kubernetes deployment that runs entirely on your workstation or edge device.

```bash
# Install microk8s
>> sudo snap install microk8s --classic

# Create alias 
>> sudo snap alias microk8s.kubectl kubectl

# Enable Ingress
>> microk8s enable dns ingress ambassador

# Use microk8s config as the default kubernetes config
>> microk8s config > config
```

Verify the installation

```
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

### Install CRDs

```bash
# Fetch Frisbee 
>> git clone git@github.com:CARV-ICS-FORTH/frisbee.git


# Install Frisbee CRDs
>> make install
customresourcedefinition.apiextensions.k8s.io/chaos.frisbee.io configured
customresourcedefinition.apiextensions.k8s.io/clusters.frisbee.io configured
customresourcedefinition.apiextensions.k8s.io/services.frisbee.io configured
customresourcedefinition.apiextensions.k8s.io/templates.frisbee.io configured
customresourcedefinition.apiextensions.k8s.io/workflows.frisbee.io configured

>> kubectl get crds | grep frisbee 
chaos.frisbee.io                        2021-10-07T10:33:04Z
clusters.frisbee.io                     2021-10-07T10:33:04Z
services.frisbee.io                     2021-10-07T10:33:05Z
templates.frisbee.io                    2021-10-07T11:48:00Z
workflows.frisbee.io                    2021-10-07T10:33:05Z

# Remove any previous Chaos-Mesh dependency
>> curl -sSL https://mirrors.chaos-mesh.org/latest/install.sh | bash -s -- --microk8s --template | kubectl delete -f -

# Install a fresh Chaos-Mesh 
>> curl -sSL https://mirrors.chaos-mesh.org/latest/install.sh | bash -s -- --microk8s --template | kubectl create -f -

-- Expected all CRDS to be successfully created --
```

## Deploy a Testplan

#### Run Controller

There are three ways to run the operator:

- As Go program outside a cluster
- As a Deployment inside a Kubernetes cluster
- Managed by
  the [Operator Lifecycle Manager (OLM)](https://sdk.operatorframework.io/docs/olm-integration/tutorial-bundle/#enabling-olm)
  in [bundle](https://sdk.operatorframework.io/docs/olm-integration/quickstart-bundle) format.

```bash
# Run Frisbee controller outside a cluster (from Frisbee directory)
>> make run
```

#### Install Templates

```bash
# Install Frisbee Templates (from Frisbee directory)
>> find examples/templates/ -name "*.yml" -exec kubectl  apply -f {} \;
template.frisbee.io/observability unchanged
configmap/prometheus-config unchanged
configmap/grafana-config unchanged
template.frisbee.io/sysmon unchanged
configmap/sysmon-dashboard unchanged
template.frisbee.io/rediscluster unchanged
template.frisbee.io/redismon unchanged
configmap/redis-dashboard unchanged
template.frisbee.io/redis unchanged
template.frisbee.io/ycsbmon unchanged
configmap/ycsb-dashboard unchanged
```

#### Access Grafana outside the Cluster

Frisbee uses Ingress controller to expose Grafana dashboard externally to the cluster.

We use Ambassador as the default Ingress controller, as shown in  `examples/testplans/validate-local.yml` .



#### Deploy the plan

```bash
# Run a validation experiment (from Frisbee directory)
>> kubectl apply -f examples/testplans/validate-local.yml 
workflow.frisbee.io/validate-local created

# Confirm that Kube API has received the workflow
>> kubectl get workflows.frisbee.io
NAME           AGE
validate-local   47s

# Delete a workflow
>> kubectl delete -f examples/testplans/validate-local.yml --cascade=foreground
workflow.frisbee.io "validate-local" deleted
```

## Observe a Testplan

#### Controller Logs

The logs of the controller are accessible by the terminal on which the controller is running (see make run)

#### Grafana Dashboards & Alerts

```bash
# Access Grafana via your browser
http://grafana.localhost/
```

### Kubernetes Dashboard

Dashboard is a web-based Kubernetes user interface. You can use Dashboard to deploy containerized applications to a
Kubernetes cluster, troubleshoot your containerized application, and manage the cluster resources.

```bash
# Deploy the dashboard
>> kubectl --kubeconfig /home/fnikol/.kube/config.remote  apply -f https://raw.githubusercontent.com/kubernetes/dashboard/v2.3.1/aio/deploy/recommended.yaml

# To access Dashboard from your local workstation you must create a secure channel to your Kubernetes cluster
>> microk8s dashboard-proxy
Checking if Dashboard is running.
Dashboard will be available at https://127.0.0.1:10443

# Now access Dashboard at:
https://127.0.0.1:10443
```

### Chaos Dashboard

The Chaos *Dashboard* is a one-step web UI for managing, designing, and monitoring chaos experiments on *Chaos Mesh*.

The Chaos Dashboard is installed automatically when you deploy the Chaos-Mesh into your Cluster.

```bash
# Forward port
kubectl port-forward -n chaos-testing svc/chaos-dashboard 2333:2333

# Now access Dashboard at:
http://127.0.0.1:2333/dashboard
```

# Run Frisbee on a Remote Kubernetes Cluster

## Before Starting

### Ensure Connectivity

In order to access your Kubernetes cluster, `frisbee` uses kubeconfig to find the information it needs to choose a
cluster and communicate with it.

`kubeconfig` files organize information about clusters, users, namespaces, and authentication mechanisms.

The configuration is the same as `kubectl` and is located at `~/.kube/config`.

Nonetheless, to avoid caching effects we use the configuration directly.

```bash
# Create tunnel for sending requests to Kubernetes controller
>> ssh -L 6443:192.168.1.213:6443 [USER@]SSH_SERVER

# Deploy a hello world
>>  kubectl --kubeconfig ~/.kube/config.remote create deployment hello-node --image=k8s.gcr.io/echoserver:1.4
deployment.apps/hello-node created

# Verify that a hell-node deployment exists
>> kubectl --kubeconfig ~/.kube/config.remote get deployments
NAME         READY   UP-TO-DATE   AVAILABLE   AGE
hello-node   1/1     1            1           36s

# Delete the deployment
>> kubectl --kubeconfig ~/.kube/config.remote delete deployments hello-node
deployment.apps "hello-node" deleted
```

### Install CRDs

```bash
# Fetch Frisbee 
>> git clone git@github.com:CARV-ICS-FORTH/frisbee.git
>> cd frisbee


# Since the webhook requires a TLS certificate that the apiserver is configured to trust, install the cert-manager with the following command:
>> kubectl apply -f https://github.com/jetstack/cert-manager/releases/latest/download/cert-manager.yaml

# Install Frisbee CRDs
>> make install KUBECONFIG="--kubeconfig /home/fnikol/.kube/config.remote"
customresourcedefinition.apiextensions.k8s.io/chaos.frisbee.io configured
customresourcedefinition.apiextensions.k8s.io/clusters.frisbee.io configured
customresourcedefinition.apiextensions.k8s.io/services.frisbee.io configured
customresourcedefinition.apiextensions.k8s.io/templates.frisbee.io configured
customresourcedefinition.apiextensions.k8s.io/workflows.frisbee.io configured

>> kubectl --kubeconfig /home/fnikol/.kube/config.remote get crds | grep frisbee 
chaos.frisbee.io                        2021-10-07T10:33:04Z
clusters.frisbee.io                     2021-10-07T10:33:04Z
services.frisbee.io                     2021-10-07T10:33:05Z
templates.frisbee.io                    2021-10-07T11:48:00Z
workflows.frisbee.io                    2021-10-07T10:33:05Z

# Remove any previous Chaos-Mesh dependency
>> curl -sSL https://mirrors.chaos-mesh.org/latest/install.sh | bash -s -- --template | kubectl  --kubeconfig ~/.kube/config.remote delete --cascade=foreground -f -

# Install a fresh Chaos-Mesh. Beware of the runtime engine. It may be containerd or docker
# https://github.com/chaos-mesh/chaos-mesh/issues/2300
>> curl -sSL https://mirrors.chaos-mesh.org/latest/install.sh | bash -s -- -r d --template | kubectl  --kubeconfig ~/.kube/config.remote create -f -

# Alternatively, you can look at the Chaos-Mesh instructions directly.
# https://chaos-mesh.org/docs/production-installation-using-helm/

-- Expected all CRDS to be successfully created --
```

## Deploy a Testplan

#### Run Controller

There are three ways to run the operator:

- As Go program outside a cluster
- As a Deployment inside a Kubernetes cluster
- Managed by
  the [Operator Lifecycle Manager (OLM)](https://sdk.operatorframework.io/docs/olm-integration/tutorial-bundle/#enabling-olm)
  in [bundle](https://sdk.operatorframework.io/docs/olm-integration/quickstart-bundle) format.

```bash
# Run Frisbee controller outside a cluster (from Frisbee directory)
>> make run
```

#### Install Templates

```bash
# Install Frisbee Templates (from Frisbee directory)
>> find examples/templates/ -name "*.yml" -exec kubectl  --kubeconfig ~/.kube/config.remote apply -f {} \;
template.frisbee.io/observability unchanged
configmap/prometheus-config unchanged
configmap/grafana-config unchanged
template.frisbee.io/sysmon unchanged
configmap/sysmon-dashboard unchanged
template.frisbee.io/rediscluster unchanged
template.frisbee.io/redismon unchanged
configmap/redis-dashboard unchanged
template.frisbee.io/redis unchanged
template.frisbee.io/ycsbmon unchanged
configmap/ycsb-dashboard unchanged
```

#### Access Grafana outside the Cluster

Frisbee uses Ingress controller to expose Grafana dashboard externally to the cluster.

The Ingress controller, however, requires a hostname.

Open `examples/workflows/validate-remote.yml` and change the ingress field to point to your Kubernetes manager.

```bash
ingress:
  host: platform.science-hangar.eu
```

#### Deploy the plan

```bash
# Run a validation experiment (from Frisbee directory)
>> kubectl --kubeconfig ~/.kube/config.remote  apply -f examples/testplans/validate-remote.yml 
workflow.frisbee.io/validate-remote created

# Confirm that Kube API has received the workflow
>> kubectl --kubeconfig ~/.kube/config.remote  get workflows.frisbee.io
NAME           AGE
validate   47s

# Delete a workflow
>> kubectl --kubeconfig ~/.kube/config.remote  delete --cascade=foreground -f examples/testplans/validate-remote.yml 
workflow.frisbee.io "validate-remote" deleted
```

## Observe a Testplan

#### Controller Logs

The logs of the controller are accessible by the terminal on which the controller is running (see make run)

#### Grafana Dashboards & Alerts

```bash
# Access Grafana via your browser
http://grafana.platform.science-hangar.eu 
```

The domain depends on the provided values at the previous step  (see Access Grafana outside the cluster). In this case,
we use provided domain is linked to our testbed "platform.science-hangar.eu".

### Kubernetes Dashboard

Dashboard is a web-based Kubernetes user interface. You can use Dashboard to deploy containerized applications to a
Kubernetes cluster, troubleshoot your containerized application, and manage the cluster resources.

```bash
# Deploy the dashboard
>> kubectl --kubeconfig /home/fnikol/.kube/config.remote  apply -f https://raw.githubusercontent.com/kubernetes/dashboard/v2.3.1/aio/deploy/recommended.yaml

# To access Dashboard from your local workstation you must create a secure channel to your Kubernetes cluster
>> kubectl --kubeconfig /home/fnikol/.kube/config.remote proxy
Starting to serve on 127.0.0.1:8001

# Now access Dashboard at:
http://localhost:8001/api/v1/namespaces/kubernetes-dashboard/services/https:kubernetes-dashboard:/proxy/.
```

### Chaos Dashboard

The Chaos *Dashboard* is a one-step web UI for managing, designing, and monitoring chaos experiments on *Chaos Mesh*.

The Chaos Dashboard is installed automatically when you deploy the Chaos-Mesh into your Cluster.

```bash
# Forward port
kubectl --kubeconfig=/home/fnikol/.kube/config.remote port-forward -n chaos-testing svc/chaos-dashboard 2333:2333

# Now access Dashboard at:
http://127.0.0.1:2333/dashboard
```
