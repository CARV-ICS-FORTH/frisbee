# Iperf3

> [*Iperf3*](https://github.com/esnet/iperf)  A TCP, UDP, and SCTP network bandwidth measurement tool

## TL;DR

Install the platform and dependent charts.

```bash
>> helm repo add frisbee https://carv-ics-forth.github.io/frisbee/charts
>> helm install my-frisbee frisbee/platform
>> helm install my-iperf3 frisbee/iperf3
```

Run any of the testing plans.

```bash
>> kubectl apply -f examples/0.server-client.yml
```

## Introduction

This chart bootstraps an Iperf3 deployment on a [Kubernetes](http://kubernetes.io) cluster using
the [Helm](https://helm.sh) package manager.

## Prerequisites

- Kubernetes 1.19+

- Helm 3.5.0

## Installing the Chart

To install the chart with the release name `my-release`:

```bash
# Install helm repo
>> helm repo add frisbee https://carv-ics-forth.github.io/frisbee/charts
# Install Frisbee platform
>> helm install my-frisbee frisbee/platform
# Install dependent charts
>> helm install my-iperf3 frisbee/iperf3
```

These commands deploy Iperf on the Kubernetes cluster in the default configuration.

The [Parameters](#parameters) section lists the parameters that can be configured during installation.

> **Tip**: List all releases using `helm list`

## Uninstalling the Chart

To uninstall/delete the `my-iperf2` release:

```bash
>> helm delete my-iperf3
```

The command removes all the Kubernetes components associated with the chart and deletes the release. Use the
option `--purge` to delete all history too.

## Parameters
