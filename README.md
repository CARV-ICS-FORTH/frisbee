
# Why Frisbee ?



Frisbee is a next generation testbed tool built for exploring, testing, and benchmarking modern applications. We address the key pain points developers and QA engineers  face when testing cloud-native applications.

We make it possible to:

* **Write tests:**  for stressing complex topologies and dynamic operating conditions.

* **Run tests:**  provides seamless scaling from a single workstation to hundreds of machines.

* **Debug tests:**  through extensive monitoring and comprehensive dashboards

  

In this walk-through, we explain how to install and execute the runtime with given examples. We will discuss later how to use the language to build custom experiments.



## Installation

#### Local Kubernetes Installation

*MicroK8s* is a CNCF certified upstream Kubernetes deployment that runs entirely on your workstation or edge device.

```bash
# Install microk8s
$ sudo snap install microk8s --classic

# Create alias 
$ sudo snap alias microk8s.kubectl kubectl

# Enable features
$ microk8s enable dns ingress ambassador

# Use microk8s config as kubernetes config
$ microk8s config > config
```



#### Install CRDs

CRDs are extensions of the Kubernetes API. 

```bash
# Install Frisbee CRD (from Frisbee homefolder)
$ make install

# Chaos Mesh provides Chaos engineering capabilities to Frisbee
curl -sSL https://mirrors.chaos-mesh.org/latest/install.sh | bash -s -- --microk8s
```



### Run experiments

## 



#### On local installation

```bash
# Run an experiment
$ kubectl -n frisbee apply -f  ../paper/elasticity.yml

# Delete an experiment
$ kubectl -n frisbee delete -f  ../paper/elasticity.yml
```



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

## 

Dashboard is a web-based Kubernetes user interface. You can use Dashboard to deploy containerized applications to a Kubernetes cluster, troubleshoot your containerized application, and manage the cluster resources.



```bash
# Run Kubernetes dashboard
$ microk8s dashboard-proxy
Dashboard will be available at https://127.0.0.1:10443

# Run Chaos dashboard
$ kubectl port-forward -n chaos-testing svc/chaos-dashboard 2333:2333
Dashboard will be available at http://127.0.0.1:2333/dashboard/experiments

# Run Frisbee dashboard (Grafana)
# This parameter is configured using the ingress parameter of the workflow. By default, 
Dashboard will be available at http://grafana.localhost
```





#### Fetch PDF from Grafana

To fetch a PDF from Grafana follow the instructions of: https://gist.github.com/svet-b/1ad0656cd3ce0e1a633e16eb20f66425

Briefly,

1. Install Node.js on your local workstation
2. wget https://gist.githubusercontent.com/svet-b/1ad0656cd3ce0e1a633e16eb20f66425/raw/grafana_pdf.js
3. execute the grafana_fs.js over node.ns



> node grafana_pdf.js "http://grafana.localhost/d/A2EjFbsMk/ycsb-services?viewPanel=74" "":"" output.pdf

###### 

###### Permissions

By default Grafana is configured without any login requirements, so we must leave this field blank

"":"" denotes empty username:password.





## Bugs and Feedback
For bug report, questions and discussions please submit [GitHub Issues](https://github.com/CARV-ICS-FORTH/frisbee/issues). 


You can also contact us via:
* Email: fnikol@ics.forth.gr
* Slack group: ...
* Twitter: ...



## Contributing

We welcome every contribution, even if it is just punctuation. See details of [CONTRIBUTING](docs/CONTRIBUTING.md)



## Business Registration

The original intention of our open source project is to lower the threshold for chaos engineering to be implemented in enterprises, so we highly value the use of the project in enterprises. Welcome everyone here [ISSUE](https://github.com/chaosblade-io/chaosblade/issues/32). After registration, you will be invited to join the corporate mail group to discuss the problems encountered by Chaos Engineering in the landing of the company and share the landing experience.




## License

Frisbee is licensed under the Apache License, Version 2.0. See [LICENSE](http://www.apache.org/licenses/LICENSE-2.0) for the full license text.


