## Frisbee - A Test Automation Framework For Kubernetes

<p align="center">
    <a href="https://www.vectorstock.com/royalty-free-vector/disc-golf-frisbee-eps-vector-25179185">
        <img src="docs/images/logo.jpg" width="400">
    </a>
</p>


<p align="center">
    <a href="https://frisbee.dev/">Website</a> |
    <a href="https://frisbee.dev/blog">Blog</a> |
    <a href="https://frisbee.dev/docs/">Docs</a> |
    <a href="mailto: fnikol@ics.forth.gr">Contact</a>
    <br /><br />
</p>

<p align="center">
    <a href="https://github.com/carv-ics-forth/frisbee/blob/main/LICENSE">
        <img src="https://img.shields.io/github/license/carv-ics-forth/frisbee">
    </a>    
    <a href="https://github.com/carv-ics-forth/frisbee/blob/main/CONTRIBUTING.md">
        <img src="https://img.shields.io/badge/PRs-welcome-brightgreen.svg">
    </a>    
    <a Code build and checks>
        <img src="https://github.com/CARV-ICS-FORTH/frisbee/actions/workflows/test-unit.yml/badge.svg">
    </a>        
</p>    

## What is Frisbee ?

Frisbee is a workflow-based engine that lowers the threshold of testing containerized applications on Kubernetes. 

Frisbee is implemented as a set of Kubernetes CRD (Custom Resource Definition) that:

* Setup initial application stack – easily!

* Test against actual, close to production software - no mocks!

* Replay complex workloads written in an intuitive language!

* Combine Chaos Engineering with large-scale performance testing!

* Assert actual program behavior and side effects.

  

To learn more about Frisbee, check the **[QuickStart](https://frisbee.dev/docs/quick-start/)** tutorial or visit our **[Website](https://frisbee.dev)**.

## Use-Cases and Testing Patterns 

👉 [Databases](charts/databases)

👉 [Federated Learning](charts/federated-learning)

👉 [Filesystems](charts/filesystems)

👉 [HPC](charts/hpc)

👉 [Networking](charts/networking)



## Getting Started

Before starting, Make sure that [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl-linux/) and  [Helm](https://helm.sh/docs/intro/install/) are installed on your system.

If you use `microk8s`, make sure you have created aliases to the commands.

```shell
alias kubectl='microk8s kubectl'
alias helm='microk8s helm3'
```



### Install Frisbee CLI and Frisbee Platform

Then, run the `install.sh`, that will deploy the **Frisbee Terminal**  as an extension to  `kubectl`.

```shell
curl -sSLf https://raw.githubusercontent.com/CARV-ICS-FORTH/frisbee/main/install.sh | bash
```

Through **Frisbee Terminal** we can easily install the **Frisbee Platform**.

```shell
kubectl frisbee install production
```

Finally, download the Frisbee repo from GitHub.

```shell
git clone git@github.com:CARV-ICS-FORTH/frisbee.git
```

This step is not really needed for the installation.
We use it to get local access in `examples` and `charts` directories.

* [examples](examples): contains a list of test-cases.
* [charts](charts): contains Helm charts that provide templates used in the test-cases.


### Submit a Testing Job.

Let's start by running the  hello-world.

```shell
kubectl frisbee submit test demo- ./examles/1.hello-world.yml
```



<p align="center">
    <img src="docs/readme.assets/submit.png" width="400">
</p>



The name can be explicit (e.g, my-demo), or autogenerated given a prefix followed by a `-` (e.g, demo-).

In any case, `submit` returns the `test id` (`| awk /test:/'{print $4}`) so that it can be used to downstream commands.


### Inspect Submitted Jobs. 

To get a list of submitted tests, use:

```shell
kubectl frisbee get tests
```

<p align="center">
    <img src="docs/readme.assets/list.png" width="400">
</p>


Note that every test-case runs on a dedicated namespace (named after the test). To further dive into execution details use: 

```shell
kubectl frisbee inspect tests demo-482
```

<p align="center">
    <img src="docs/readme.assets/inspect.png" width="900">
</p>



### Live Progress Monitoring

Let's try something more complex to demonstrate the integration of Frisbee with Prometheus/Grafana.

```shell
kubectl frisbee submit test demo- examples/15.performance-monitoring.yml charts/system charts/networking/iperf2
```

Notice that we modified the command to include `dependencies` required for the execution of the scenario.
* **charts/systems**: provides the templates for the telemetry stack.
* **charts/networking/iperf**: provides the templates for iperf benchmark.

<p align="center">
    <img src="docs/readme.assets/inspect-dashboards.png" width="900">
</p>



All that it takes now is to open the URLs of section `Visualization Dashboards` in your browser.
You can it either manually or via the one-liner:

```shell
firefox $(kubectl frisbee inspect tests demo-710 | grep grafana- | awk '{print $3'})
```

<p align="center">
    <img src="docs/readme.assets/grafana.png" width="900">
</p>


In contrast to the vanilla Grafana which plots only the performance metrics, Frisbee provides `Contextualized Visualizations` that contain information for:
* Joining nodes (blue vertical lines)
* Exiting nodes (orange vertical lines)
* Fault-Injection (red ranges)

Information like that helps in `root-cause analysis`, as it makes it easy to correlate an `observed behavior back to a testing event`. 

For example, in the next figure, it fairly easy to understand that `INSERT_ERROR` messages (`yellow line`) are triggered by a `fault-injection event`.

<p align="center">
    <img src="docs/readme.assets/contextualized-visualization.png" width="900">
</p>



## Features

👉 Workflow templating to store commonly used workflows in the cluster.

👉 DAG based declaration of testing workflows.

👉 Step level input & outputs (template parameterization).

👉 Conditional Execution (Time-Driven, Status-Driven, Performance-Driven).

👉 Live Progress monitoring via Prometheus/Grafana.

👉 Assertions and alerting of SLA violations.

👉 Placement Policies (affinity/tolerations/node selectors).

👉 Archiving Test results after executing for later access.

👉 On-Demand reliable container attached storage.

👉 Garbage collection of completed resources.

👉 Chaos-Engineering and Fault-Injection via Chaos-Mesh.

👉 On-Demand reliable container attached storage.

👉 CLI applications to test management and test inspection. 


To learn how to use these features, check the **[Walkthrough](https://frisbee.dev/docs/walkthrough)**. 


## Citation

If you publish work that uses Frisbee, please cite Frisbee as follows:

```bibtex
@article{nikolaidis2021frisbee,
title={Frisbee: automated testing of Cloud-native applications in Kubernetes},
author={Nikolaidis, Fotis and Chazapis, Antony and Marazakis, Manolis and Bilas, Angelos},
journal={arXiv preprint arXiv:2109.10727},
year={2021}
}
```



## Contributing to Frisbee

We welcome contributions. Please see [CONTRIBUTING.md](CONTRIBUTING.md) to get
started!




## Acknowledgements

This project has received funding from the European Union's Horizon 2020 research and innovation programme under grant
agreement No. 894204 (Ether, H2020-MSCA-IF-2019).
