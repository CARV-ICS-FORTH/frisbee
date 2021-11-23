# Frisbee
**Kubernetes Native Testbed**

![releaser](https://github.com/carv-ics-forth/frisbee/workflows/helm-release/badge.svg)
[![Go Report Card](https://goreportcard.com/badge/github.com/carv-ics-forth/frisbee)](https://goreportcard.com/report/github.com/carv-ics-forth/frisbee)
![License: Apache-2.0](https://img.shields.io/github/license/carv-ics-forth/frisbee?color=blue)
[![GitHub Repo stars](https://img.shields.io/github/stars/carv-ics-forth/frisbee)](https://github.com/carv-ics-forth/frisbee/stargazers)

<a href="https://www.vectorstock.com/royalty-free-vector/disc-golf-frisbee-eps-vector-25179185"><figure><img src="docs/images/logo.jpg" width="400"></figure></a>


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

👉 For feature requests and bugs, file an [issue](https://github.com/carv-ics-forth/frisbee/issues).

👉 For discussions or questions, contact us [directly](fnikol@ics.forth.gr).

👉 To get updates ⭐️ [star this repository](https://github.com/carv-ics-forth/frisbee/stargazers).



## ➕ Contributing

The original intention of our open source project is to lower the threshold of testing distributed systems, so we highly
value the use of the project in enterprises and in academia.

We welcome also every contribution, even if it is just punctuation. Here are some steps to help get you started:

✔ Read and agree to the [Contribution Guidelines](docs/CONTRIBUTING.md).

✔ Browse through the [GitHub discussions](https://github.com/carv-ics-forth/frisbee/discussions/landing).

✔ Read Frisbee design and development details on the [GitHub Wiki](https://github.com/carv-ics-forth/frisbee/wiki).

✔ Contact us [directly](fnikol@ics.forth.gr)for other ways to get involved.


## TL;DR


[Click Here](http://grafana.localhost/d/R5y4AE8Mz/kubernetes-cluster-monitoring-via-prometheus?orgId=1&amp;from=now-15m&amp;to=now)

If everything went smoothly, you should see a similar dashboard. Through these dashboards humans and controllers can
examine to check things like completion, health, and SLA compliance.

#### Client-View (YCSB-Dashboard)

![image-20211008230432961](docs/images/partitions.png)

#### Client-View (Redis-Dashboard)

![](docs/images/masterdashboard.png)


## Acknowledgements

This project has received funding from the European Union's Horizon 2020 research and innovation programme under grant
agreement No. 894204 (Ether, H2020-MSCA-IF-2019).
