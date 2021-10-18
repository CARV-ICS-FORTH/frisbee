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
