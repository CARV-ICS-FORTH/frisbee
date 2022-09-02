# Frisbee - The Kubernetes Native Testbed






<div align="center">
    <a href="https://www.vectorstock.com/royalty-free-vector/disc-golf-frisbee-eps-vector-25179185">
        <img src="docs/images/logo.jpg" width="400">
    </a>
</div>



<p align="center">
    <a href="https://frisbee.dev/">Website</a> |
    <a href="https://frisbee.dev/blog">Blog</a> |
    <a href="https://frisbee.dev/docs/">Docs</a> |
    <a href="mailto: fnikol@ics.forth.gr">Contact</a>
    <br /><br />
</p>
​											[![GitHub license](https://img.shields.io/github/license/carv-ics-forth/frisbee)](https://github.com/adap/flower/blob/main/LICENSE) [![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](https://github.com/carv-ics-forth/frisbee/blob/main/CONTRIBUTING.md) ![Code build and checks](https://github.com/CARV-ICS-FORTH/frisbee/actions/workflows/test-unit.yml/badge.svg)

# Why Frisbee ?

The effort being put in automating tests should not take over delivering value to users.
Frisbee lowers the threshold of testing distributed systems by making it possible to:

🎁 Setup initial dependency stack – easily!

🏭 Test against actual, close to production software - no mocks!

⏳ Replay complex workloads written in an intuitive language!

🏗️ Combine Chaos Engineering with large-scale performance testing!

🕹️ Assert actual program behavior and side effects



# Usage

👉 To begin with Frisbee, check the **[QuickStart](https://frisbee.dev/docs/quick-start/).**

👉 To understand the basic features, check the **[Walkthrough](https://frisbee.dev/docs/walkthrough).**



#### Testing Patterns


Among others, you will find scenarios and testing patterns for:

👉 [Databases](charts/databases)

👉 [Federated Learning](charts/federated-learning)

👉 [Filesystems](charts/filesystems)

👉 [HPC](charts/hpc)

👉 [Networking](charts/networking)



# TL;DR

1. Make sure that [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl-linux/)
   and  [Helm](https://helm.sh/docs/intro/install/) are installed on your system.

2. Update Helm repo.

```bash
>> helm repo add frisbee https://carv-ics-forth.github.io/frisbee/charts
```

3. Install Helm Packages.

```bash
# Install the platform
>> helm upgrade --install --wait my-frisbee frisbee/platform
# Install the package for monitoring YCSB output
>> helm upgrade --install --wait my-ycsb frisbee/ycsb
# Install TiKV store
>> helm upgrade --install --wait my-tikv frisbee/tikv
```

4. Create/Destroy the scenario.

```bash
# Create
>> curl -sSL https://raw.githubusercontent.com/CARV-ICS-FORTH/frisbee/main/charts/tikv/examples/scenario.baseline.yml | kubectl -f - apply

# Destroy
>> curl -sSL https://raw.githubusercontent.com/CARV-ICS-FORTH/frisbee/main/charts/tikv/examples/scenario.baseline.yml | kubectl -f - delete --cascade=foreground
```

If everything went smoothly, you should see a
similar [Grafana Dashboard](http://grafana.localhost/d/R5y4AE8Mz/kubernetes-cluster-monitoring-via-prometheus?orgId=1&amp;from=now-15m&amp;to=now)
.

Through these dashboards humans and controllers can examine to check things like completion, health, and SLA compliance.

#### Client-View (YCSB-Dashboard)

![image-20211008230432961](docs/images/partitions.png)

#### Client-View (Redis-Dashboard)

![](docs/images/masterdashboard.png)






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
