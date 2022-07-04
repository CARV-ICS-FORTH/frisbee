# Flower

> [*Flower*](https://github.com/adap/flower)  A Friendly Federated Learning Framework



## Install the Chart

Firstly, install the platform and dependent charts.

```bash
# Download the source code
>> git clone git@github.com:CARV-ICS-FORTH/frisbee.git

# Move to the Frisbee project
>> cd frisbee

# Install the package
>> helm upgrade --install my-frisbee frisbee/platform --debug
>> helm upgrade --install --wait my-system ./charts/system --debug -n mytest --create-namespace
>> helm upgrade --install my-flower ./charts/flower/ --debug -n mytest
```



Now you are ready to run the Flower scenarios.

```bash
>>  ls -1a ./charts/flower/examples/
10.scheduled-join.yml
11.scaled-baseline.yml
1.baseline.yml
2.throttle-server.yml
3.throttle-client.yml
4a.partition-server-init.yml
4b.partition-server-progress.yml
5.partition-client.yml
6.partition-p2p.yml
7.network-loss.yml
8.network-duplicates.yml
9.network-delay.yml
n.scaled-baseline.yml
```



## Run a Scenario

Let's choose the 3rd scenario.

```bash
>> kubectl -f ./charts/flower/examples/3.throttle-client.yml apply
persistentvolumeclaim/dataset created
scenario.frisbee.dev/throttle-client created
```



Once it's running, you can access the real-time dashboard. To do so, you can get the endpoints by inspecting the scenario.

```bash
>> kubectl describe scenario.frisbee.dev/throttle-client
...
  Grafana Endpoint:     grafana-karvdash-fnikol.platform.science-hangar.eu
  Message:              running jobs: [clients server throttled-client]
  Phase:                Running
  Prometheus Endpoint:  prometheus-karvdash-fnikol.platform.science-hangar.eu
```

Notice the ".platform.science-hangar.eu". This is the cluster's dns, and we have it defined in the `charts/platform/values.yaml`.


If you are too lazy, you can open it with the following one-liner:

`firefox $(kubectl describe scenario.frisbee.dev/throttle-client | grep Grafana | cut -d ":" -f 2)`



## Uninstall the Chart

Firstly, terminate the experiment.

```bash
>> kubectl -f ./charts/flower/examples/3.throttle-client.yml delete --cascade=foreground
```



Then delete the installed charts. 

```bash
>> helm delete my-flower my-system my-frisbee
```

The command removes all the Kubernetes components associated with the chart and deletes the release. Use the
option `--purge` to delete all history too.

## Parameters
