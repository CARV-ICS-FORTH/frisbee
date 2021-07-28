

<figure><img src="/docs/images/logo.jpg" width="200"></figure>





# Why Frisbee ?



Frisbee is a next generation testbed tool built for exploring, testing, and benchmarking modern applications. We address the key pain points developers and QA engineers  face when testing cloud-native applications.

We make it possible to:

* **Write tests:**  for exploring complex topologies and dynamic operating conditions.

* **Run tests:**  with seamless scaling from a single workstation to hundreds of machines.

* **Debug tests:**  through extensive monitoring and comprehensive dashboards.

  

In this walk-through, we explain how to install Kubernetes locally and run Frisbee workflows. 

We will discuss later how to use the language to build custom experiments.



## Installation

#### Install Kubernetes

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



#### Step 1. Load Templates

Templates are Frisbee objects that hold templates of reusable services.

```bash
# From Frisbee homefolder
$ cd examples/templates
$ find . -name "*.yml" -exec kubectl -n frisbee apply -f {} \;
```



#### Step 2. Deploy the experiment 

```bash
# Run the experiment
$ kubectl -n frisbee apply -f  ../paper/elasticity.yml

# Dashboard will be available at http://grafana.localhost
```



#### Step 3. Real-time visualization 



Dashboard will be available at http://grafana.localhost



##### Fetch PDF from Grafana

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





#### Step 4. Destroy the experiment 

```bash
$ kubectl -n frisbee delete -f  ../paper/elasticity.yml
```







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

