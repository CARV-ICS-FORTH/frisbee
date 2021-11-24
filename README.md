# Frisbee
**Kubernetes Native Testbed**

![releaser](https://github.com/carv-ics-forth/frisbee/workflows/helm-release/badge.svg)
[![Go Report Card](https://goreportcard.com/badge/github.com/carv-ics-forth/frisbee)](https://goreportcard.com/report/github.com/carv-ics-forth/frisbee)
![License: Apache-2.0](https://img.shields.io/github/license/carv-ics-forth/frisbee?color=blue)
[![GitHub Repo stars](https://img.shields.io/github/stars/carv-ics-forth/frisbee)](https://github.com/carv-ics-forth/frisbee/stargazers)

<a href="https://www.vectorstock.com/royalty-free-vector/disc-golf-frisbee-eps-vector-25179185">
  <img src="docs/images/logo.jpg" width="400">
</a>

Frisbee is a next generation platform designed to unify chaos testing and perfomance benchmarking.

We address the key pain points developers and QA engineers face when testing cloud-native applications in the earlier
stages of the software lifecycle.

We make it possible to:

* **Write tests:**  for stressing complex topologies and dynamic operating conditions.
* **Run tests:**  provides seamless scaling from a single workstation to hundreds of machines.
* **Debug tests:**  through extensive monitoring and comprehensive dashboards

Our platform consists of a set of Kubernetes controller designed to run performance benchmarks and introduce failure
conditions into a running system, monitor site-wide health metrics, and notify systems with status updates during the
testing procedure.

Frisbee provides a flexible, YAML-based configuration syntax and is trivially extensible with additional functionality.



## 📙 Documentation

Frisbee installation and reference documents are available at:

👉 **[Quick Start](docs/introduction.md)**

👉 **[Installation](docs/installation.md)**

👉 **[Test Plans](charts)**


## 🙋‍♂️ Getting Help

We are here to help!

👉 If you face troubles, check the [Frequently Asked Questions](docs/faq.md).

👉 For feature requests and bugs, file an [issue](https://github.com/carv-ics-forth/frisbee/issues).

👉 For great new ideas, browse through the [GitHub discussions](https://github.com/carv-ics-forth/frisbee/discussions).

👉 To get updates ⭐️ [star this repository](https://github.com/carv-ics-forth/frisbee/stargazers).



## ➕ Contributing

The original intention of our open source project is to lower the threshold of testing distributed systems, so we highly
value the use of the project in enterprises and in academia.

We welcome also every contribution, even if it is just punctuation. Here are some steps to help get you started:

✔ Read and agree to the [Contribution Guidelines](docs/CONTRIBUTING.md).

✔ Read Frisbee design and development details on the [GitHub Wiki](https://github.com/carv-ics-forth/frisbee/wiki).

✔ Contact us [directly](fnikol@ics.forth.gr) for other ways to get involved.

## TL;DR

Make sure that [Microk8s](https://microk8s.io/docs) and  [Helm](https://helm.sh/docs/intro/install/) are installed on your system, then install the Frisbee dependencies:

```bash
# Clone Frisbee repository
>> git clone https://github.com/CARV-ICS-FORTH/frisbee.git
# Install TiKV dependencies
>> helm install my-frisbee charts/platform --dependency-update
>> helm install my-observability charts/observability --dependency-update
>> helm install my-sysmon charts/sysmon --dependency-update
>> helm install my-ycsbmon charts/ycsbmon --dependency-update
# Install TiKV store
>> helm install my-tikv charts/tikv --dependency-update
```

Then run:

```bash
# Run Frisbee controller
>> make run

# Deploy the testing plan
>> kubectl apply -f charts/tikv/plans/plan.localhost.yml 
```



If everything went smoothly, you should see a similar [Grafana Dashboard](http://grafana.localhost/d/R5y4AE8Mz/kubernetes-cluster-monitoring-via-prometheus?orgId=1&amp;from=now-15m&amp;to=now). 

Through these dashboards humans and controllers can examine to check things like completion, health, and SLA compliance.

#### Client-View (YCSB-Dashboard)

![image-20211008230432961](docs/images/partitions.png)

#### Client-View (Redis-Dashboard)

![](docs/images/masterdashboard.png)


## Acknowledgements

This project has received funding from the European Union's Horizon 2020 research and innovation programme under grant
agreement No. 894204 (Ether, H2020-MSCA-IF-2019).
