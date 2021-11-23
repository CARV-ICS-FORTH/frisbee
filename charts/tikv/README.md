# TiKV

TiKV is a highly scalable, low latency, and easy to use key-value database.

## TL;DR

```bash
# Add repo
>> helm repo add frisbee https://carv-ics-forth.github.io/frisbee/charts
# Install dependencies
>> helm install my-frisbee frisbee/platform
>> helm install my-observability frisbee/observability
>> helm install my-sysmon frisbee/sysmon
>> helm install my-ycsbmon frisbee/ycsbmon
# Install TiKV
>> helm install my-tikv frisbee/tikv
# Run Frisbee controller
>> make run
# Deploy the testing plan
>> kubectl apply -f plans/plan.baseline.yml 
```

## Introduction

This chart bootstraps a TiKV deployment on a [Kubernetes](http://kubernetes.io) cluster using
the [Helm](https://helm.sh) package manager.

## Prerequisites

- Kubernetes 1.19+

- Helm 3.5.0

## Installing the Chart

To install the chart with the release name `my-release`:

```console
$ helm repo add bitnami https://charts.bitnami.com/bitnami
$ helm install my-release bitnami/influxdb
```

These commands deploy influxdb on the Kubernetes cluster in the default configuration. The [Parameters](#parameters)
section lists the parameters that can be configured during installation.

> **Tip**: List all releases using `helm list`

## Uninstalling the Chart

To uninstall/delete the `my-tikv` release:

```bash
>> helm delete my-tikv
```

The command removes all the Kubernetes components associated with the chart and deletes the release. Use the
option `--purge` to delete all history too.
