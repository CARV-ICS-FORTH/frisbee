#### On remote cluster

In order to access your Kubernetes cluster, `frisbee` uses kubeconfig to to find the information it needs to choose a cluster and communicate with it.

`kubeconfig` files organize information about clusters, users, namespaces, and authentication mechanisms.

The configuration is the same as `kubectl` and is located at `~/.kube/config`.



```bash
# Create tunnel for sending requests to Kubernetes controller
$ ssh -L 6443:192.168.1.213:6443 thegates

# Run an experiment
$ kubectl -kubeConfig /home/fnikol/.kube/config.evolve -n frisbee apply -f  ../paper/elasticity.yml 		

# Delete an experiment
$ kubectl -kubeConfig /home/fnikol/.kube/config.evolve -n frisbee delete -f  ../paper/elasticity.yml
```

### Dashboard

Dashboard is a web-based Kubernetes user interface. You can use Dashboard to deploy containerized applications to a Kubernetes cluster, troubleshoot your containerized application, and manage the cluster resources.



```bash
# Run Kubernetes dashboard
$ microk8s dashboard-proxy
Dashboard will be available at https://127.0.0.1:10443

# Run Chaos dashboard
$ kubectl port-forward -n chaos-testing svc/chaos-dashboard 2333:2333
Dashboard will be available at http://127.0.0.1:2333/dashboard/experiments
```

