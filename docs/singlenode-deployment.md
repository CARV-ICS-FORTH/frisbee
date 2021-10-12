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

```bash
ingress:
  host: localhost
   useAmbassador: true
```

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
>> kubectl delete -f examples/testplans/validate-local.yml 
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
