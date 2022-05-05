# TiKV

> [TiKV](https://tikv.org/) provides both raw and ACID-compliant transactional key-value API, which is widely used in
> online serving services,
> such as the metadata storage system for object storage service, the storage system for recommendation systems, the
> online feature store, etc.

## TL;DR

Install the platform and dependent charts.

```bash
>> helm repo add frisbee https://carv-ics-forth.github.io/frisbee/charts
>> helm install my-frisbee frisbee/platform
>> helm install my-ycsb frisbee/ycsb
>> helm install my-tikv frisbee/tikv
```

Run any of the testing plans.

```bash
>> kubectl apply -f examples/plan.baseline.yml 
```

## Introduction

This chart bootstraps a TiKV deployment on a [Kubernetes](http://kubernetes.io) cluster using
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
>> helm install my-ycsb frisbee/ycsbmon
>> helm install my-tikv frisbee/tikv
```

These commands deploy TiKV on the Kubernetes cluster in the default configuration.

The [Parameters](#parameters) section lists the parameters that can be configured during installation.

> **Tip**: List all releases using `helm list`

## Uninstalling the Chart

To uninstall/delete the `my-tikv` release:

```bash
>> helm delete my-tikv
```

The command removes all the Kubernetes components associated with the chart and deletes the release. Use the
option `--purge` to delete all history too.

## Sources

https://docs.google.com/spreadsheets/d/1VjzC3IxCiqGQmSUgRxewgExE3c32YiZMUKNsKDuvrPg/edit#gid=1700439087

## Parameters
