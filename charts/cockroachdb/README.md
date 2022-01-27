# CockroachDB

> [*CockroachDB*](https://github.com/cockroachdb/cockroach) is a distributed database with standard SQL for cloud applications.

## TL;DR

Install the platform and dependent charts.

```bash
>> helm repo add frisbee https://carv-ics-forth.github.io/frisbee/charts
>> helm install my-frisbee frisbee/platform
>> helm install my-ycsb frisbee/ycsb
>> helm install my-cockroach frisbee/cockroachdb
```

Run any of the testing plans.

```bash
>> kubectl apply -f examples/plan.baseline.yml 
```

## Introduction

This chart bootstraps a CockroachDB deployment on a [Kubernetes](http://kubernetes.io) cluster using
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
>> helm install my-ycsb frisbee/ycsb
>> helm install my-cockroach frisbee/cockroachdb
```

These commands deploy CockroachDB on the Kubernetes cluster in the default configuration.

The [Parameters](#parameters) section lists the parameters that can be configured during installation.

> **Tip**: List all releases using `helm list`

## Uninstalling the Chart

To uninstall/delete the `my-cockroach` release:

```bash
>> helm delete my-cockroach
```

The command removes all the Kubernetes components associated with the chart and deletes the release. Use the
option `--purge` to delete all history too.

## Parameters
