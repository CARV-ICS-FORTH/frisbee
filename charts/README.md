# The Frisbee Library 

Popular applications, provided by Frisbee, ready to launch on Kubernetes using [Kubernetes Helm](https://github.com/helm/helm).

## TL;DR

```bash
$ helm repo add frisbee https://carv-ics-forth.github.io/frisbee/charts
$ helm search repo frisbee
$ helm install my-release frisbee/<chart>
```



## Before you begin



### Prerequisites

* Kubernetes 1.19+
* Helm 3.6.0+



### Setup a Kubernetes Cluster

The quickest way to setup a Kubernetes cluster to install Frisbee Charts is following the "Frisbee Get Started" guides for the different services:

- [Get Started with Frisbee Charts using MicroK8s](docs/get-started-microk8s/)



For setting up Kubernetes on other cloud platforms or bare-metal servers refer to the Kubernetes [getting started guide](https://kubernetes.io/docs/getting-started-guides/).



### Install Helm

Helm is a tool for managing Kubernetes charts. Charts are packages of pre-configured Kubernetes resources.

To install Helm, refer to the [Helm install guide](https://github.com/helm/helm#install) and ensure that the `helm` binary is in the `PATH` of your shell.



### Add Repo

The following command allows you to download and install all the charts from this repository:

```bash
>> helm repo add frisbee https://carv-ics-forth.github.io/frisbee/charts
```



### Using Helm



Once you have installed the Helm client, you can deploy a Frisbee Helm Chart into a Kubernetes cluster.

Please refer to the [Quick Start guide](https://helm.sh/docs/intro/quickstart/) if you wish to get running in just a few commands, otherwise the [Using Helm Guide](https://helm.sh/docs/intro/using_helm/) provides detailed instructions on how to use the Helm client to manage packages on your Kubernetes cluster.

Useful Helm Client Commands:
* View available charts: `helm search repo`
* Install a chart: `helm install my-release frisbee/<package-name>`
* Upgrade your application: `helm upgrade`



## License

Copyright &copy; 2022 Frisbee

Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the License. You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific language governing permissions and limitations under the License.

