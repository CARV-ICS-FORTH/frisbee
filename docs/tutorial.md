# Tutorial

This tutorial describes how to deploy Frisbee and start running tests.

## Run a test

#### Step 1:  Install Dependencies

Make sure that [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl-linux/)
and  [Helm](https://helm.sh/docs/intro/install/) are installed on your system, and that you have access to a Kubernetes
installation.

* **Local Installation** If you want a local installation you can use [Microk8s](https://microk8s.io/docs) that runs
  entirely on your workstation or edge device.

```bash
# Install microk8s
>> sudo snap install microk8s --classic

# Create alias 
>> sudo snap alias microk8s.kubectl kubectl

# Enable Dependencies
>> microk8s enable dns ingress ambassador

# Use microk8s config as the default kubernetes config
>> microk8s config > config
```

* **Remote Installation**: Set  `~/.kube/config` appropriately, and create tunnel for sending requests to Kubernetes
  API.

```bash
# Create tunnel for sending requests to Kubernetes API.
>> ssh -L 6443:192.168.1.213:6443 [USER@]SSH_SERVER
```

And then validate that everything works.

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

#### Step 2: Update Helm repo

```bash
# Install Helm3
>> curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash

# Update Helm repo
>> helm repo add frisbee https://carv-ics-forth.github.io/frisbee/charts
```

#### Step 3: Install Frisbee platform

```bash
# Install the platform with local ingress
>> helm upgrade --install --wait my-frisbee frisbee/platform
```

This step will install the following components:

* Frisbee CRDS
* Frisbee Controller
* Frisbee Dependency stack (e.g, Chaos toolkits, dynamic volume provisioning, observability stack)
* Ingress for making the observability stack accessible from outside the Kubernetes

By default the platform sets the Ingress to `localhost`.

If you use a non-local cluster, you can these the ingress via the  `globa.ingress` flag.

```bash
# Install the platform with non-local ingress
>> helm upgrade --install --wait my-frisbee frisbee/platform --set global.ingress=platform.science-hangar.eu 
```

#### Step 4:  Install the testing components

```bash
# Install the package for monitoring YCSB output
>> helm upgrade --install --wait my-ycsb frisbee/ycsb
# Install TiKV store
>> helm upgrade --install --wait my-tikv frisbee/tikv
```

#### Step 5: Run the testing workflow

This url points
to : https://raw.githubusercontent.com/CARV-ICS-FORTH/frisbee/main/charts/tikv/examples/plan.baseline.yml

```bash
# Create a plan
>> curl -sSL https://tinyurl.com/t3xrtmny | kubectl -f - apply
```

#### Step 6: Wait for completion

After you have successfully run a test, the next question that comes is about its completion.

```bash
# View deployed plans
>> kubectl get workflows.frisbee.io
NAME            AGE
tikv-baseline   1m

# Inspect a plan
>> kubectl describe workflows.frisbee.io tikv-baseline
```

Describe will return a lot of information about the plan. We are interested in the fields `conditions`.

We can use these fields to wait until they become true -- thus saving us from manual inspection.

```bash
# Wait until the test oracle is triggered.
>> kubectl  wait --for=condition=allActions workflows.frisbee.io/tikv-baseline 
workflow.frisbee.io/tikv-baseline condition met
```

####     

#### Step 7: Destroy the testing workflow

This url points
to : https://raw.githubusercontent.com/CARV-ICS-FORTH/frisbee/main/charts/tikv/examples/plan.baseline.yml

```bash
# Destroy a plan
>> curl -sSL https://tinyurl.com/t3xrtmny | kubectl -f - delete --cascade=foreground
```

The flag `cascade=foreground` will wait until the experiment is actually deleted. Without this flag, the deletion will
happen in the background. Use this flag if you want to run multiple tests, without interference.

# Observe a Testplan

### Kubernetes Dashboard

Dashboard is a web-based Kubernetes user interface. You can use Dashboard to deploy containerized applications to a
Kubernetes cluster, troubleshoot your containerized application, and manage the cluster resources.

```bash
# Deploy the dashboard
>>  curl -sSL https://raw.githubusercontent.com/kubernetes/dashboard/v2.3.1/aio/deploy/recommended.yaml | kubectl -f - apply

# To access Dashboard from your local workstation you must create a secure channel to your Kubernetes cluster
>> kubectl proxy
Starting to serve on 127.0.0.1:8001

# Now access Dashboard at:
http://localhost:8001/api/v1/namespaces/kubernetes-dashboard/services/https:kubernetes-dashboard:/proxy/.
```

###     

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

### Chaos Dashboard

The Chaos *Dashboard* is a one-step web UI for managing, designing, and monitoring chaos experiments on *Chaos Mesh*.

The Chaos Dashboard is installed automatically when you deploy the Chaos-Mesh into your Cluster.

```bash
# Forward port
>> kubectl port-forward svc/chaos-dashboard 2333:2333

# Now access Dashboard at:
http://localhost:2333/dashboard
```

### Grafana Dashboard & Alerts

Grafana is a multi-platform open source analytics and interactive visualization web application.

To access it, use the format `http://grafana.${INGRESS}` where `Ingress` is the value you defined in step 3.

For example,

```bash
# Access Grafana via your browser
http://grafana.platform.science-hangar.eu 
```
