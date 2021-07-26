https://chaos-mesh.org/docs/development_guides/develop_a_new_chaos/

Make uninstall make install

// Will create new CRDS

Make update

Because of this bug, https://github.com/argoproj/argo-cd/issues/820, we cannot use the kubectl apply directly on the
CRDS. For this reason, we must uninstall and install all the crds at once.

make install KUBECONFIG="--kubeconfig /home/fnikol/.kube/config.evolve" NAMESPACE="-n karvdash-fnikol"

RESERVED LABELS:

* discover
* owner


* For mapping local directories to configMap
  https://itnext.io/helm-3-mapping-a-directory-of-files-into-a-container-ed6c54372df8


# Why Frisbee ?





## In a nutshell

Frisbee is a next generation testbed tool built for the  modern application. We address the key pain points developers and QA engineers  face when testing modern applications.

We make it possible to:

- [Set up tests](https://docs.cypress.io/guides/overview/why-cypress#Setting-up-tests)
- [Write tests](https://docs.cypress.io/guides/overview/why-cypress#Writing-tests)
- [Run tests](https://docs.cypress.io/guides/overview/why-cypress#Running-tests)
- [Debug Tests](https://docs.cypress.io/guides/overview/why-cypress#Debugging-tests)

Cypress is most often compared to Selenium; however Cypress is both  fundamentally and architecturally different. Cypress is not constrained  by the same restrictions as Selenium.

This enables you to **write faster**, **easier** and **more reliable** tests.









Frisbee is a platform for exploration, testing, and benchmarking of distributed systems under various operating conditions.



Testing a distributed application before deployment to the cloud requires significant domain knowledge and can be tedious and time-consuming. Following the manual provision of physical nodes, service operators must manually setup, connect, and configure every physical and virtual device in the experimental platform (i.e., the system running the prototype). The configuration and installation process itself needs to ensure that software and logical dependencies are respected, networks are properly instantiated, and the interoperability of system components is thoroughly tested. And then, there is the need to gather the results and analyze them.



Frisbee consists of the language and the runtime. The language provides natural abstractions for the systematic modeling of experiments with complex topologies and dynamic workflows. The runtime engine handles the deployment and runtime management of the modeled environments on top of Kubernetes. In this walk-through, we explain how to install and execute the runtime with given examples. We will discuss later how to use the language to build custom experiments.





### Kubernetes Installation

#### Local setup

*MicroK8s* is a CNCF certified upstream Kubernetes deployment that runs entirely on your workstation or edge device.



```bash
# Install microk8s backend
sudo snap install microk8s --classic

# Create alias 
sudo snap alias microk8s.kubectl kubectl

# Use microk8s config as kubernetes config
microk8s config > config
```



#### Dashboard

Dashboard is a web-based Kubernetes user interface. You can use Dashboard to deploy containerized applications to a Kubernetes cluster, troubleshoot your containerized application, and manage the cluster resources.



```bash
$ microk8s enable dns
$ microk8s enable dashboard
$ microk8s kubectl port-forward -n kube-system service/kubernetes-dashboard 10443:443

Kubectl will make Dashboard available at http://localhost:8001/api/v1/namespaces/kubernetes-dashboard/services/https:kubernetes-dashboard:/proxy/.

```



#### Chaos-Mesh

Chaos-Mesh provides Chaos engineering capabilities to Frisbee

To install it:

```bash
# Install
curl -sSL https://mirrors.chaos-mesh.org/latest/install.sh | bash -s -- --microk8s

# verify
kubectl get pod -n chaos-testing

# Chaos-dashboard
$ kubectl port-forward -n chaos-testing chaos-dashboard-6fdb79c549-vmvtp 8888:2333

```









### Deploy Experiment



#### On local Kubernetes

To deploy an experiment on the default namespace use:

```bash
# Deploy experiment on default namespace
./bin/frisbee apply -f examples/paper/3.failure/failover.yml kubernetes

or

# Deploy experiment on custom namespace
./bin/frisbee apply -f examples/paper/3.failure/failover.yml kubernetes	-n karvdash-fnikol	
```





#### On remote Experiment

In order to access your Kubernetes cluster, `frisbee` uses kubeconfig to to find the information it needs to choose a cluster and communicate with it.

`kubeconfig` files organize information about clusters, users, namespaces, and authentication mechanisms.

The configuration is the same as `kubectl` and is located at `~/.kube/config`.



```bash
# Create tunnel for sending requests to Kubernetes controller
ssh -L 6443:192.168.1.213:6443 thegates

# Deploy experiment
./bin/frisbee apply -f examples/paper/3.failure/failover.yml kubernetes \
-c /home/fnikol/.kube/config.evolve -n karvdash-fnikol -i 139.91.92.156.nip.io
		
```





## Destroy Experiment



#### On local Kubernetes

```bash
# Destroy an experiment through Frisbee
./bin/frisbee destroy  kubernetes

# or 

# Delete the namespace directly directly
kubectl delete ns frisbee

```



#### On remote Kubernetes

```
./bin/frisbee destroy  kubernetes -c /home/fnikol/.kube/config.evolve -n karvdash-fnikol
```



Parameters:

* -c: path to credentials
* -n: namespace
* -i: ingress controller.





### Fetch PDF

To fetch a PDF from Grafana follow the instructions of: https://gist.github.com/svet-b/1ad0656cd3ce0e1a633e16eb20f66425



Briefly,

1. Install Node.js on your local workstation
2. wget https://gist.githubusercontent.com/svet-b/1ad0656cd3ce0e1a633e16eb20f66425/raw/grafana_pdf.js
3. execute the grafana_fs.js over node.ns



> node grafana_pdf.js "http://10.1.128.16:3000/d/A2EjFbsMk/ycsb-services?viewPanel=74" "":"" something.pdf



###### URL

The url is the link to the dashboard you are interested in. In general, the grafana format is:

grafana_ip: reported by the Frisbee controller

grafana_port: 3000 is used by default.

/.../viewPanel=74: is the relative path to a Grafana dashboard

&from=1621856255052&to=1621857758263": the time range we are interested in.



Be sure to wrap the URL around double quotes.



###### Permissions

By default Grafana is configured without any login requirements, so we must leave this field blank

"":"" denotes empty username:password.



###### Output file

* something.pdf: the output file

