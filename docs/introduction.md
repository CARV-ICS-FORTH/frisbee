# Frisbee in a Nutshell

This tutorial introduces the basic functionalities of Frisbee:

- **Write tests:**  for stressing complex topologies and dynamic operating conditions.
- **Run tests:**  provides seamless scaling from a single workstation to hundreds of machines.
- **Debug tests:**  through extensive monitoring and comprehensive dashboards.

For the rest of this tutorial we will use the Frisbee package of TiKV key/value store.

#### Frisbee Installation

Before anything else, we need to install the Frisbee platform and the Frisbee packages for testing.

```

```

At this step we will deploy the Frisbee CRDs and its dependencies.

Although HELM packages automate this task, for completeness we show the manual method.

Firstly, we need to down the Frisbee source code.

```bash
>> git clone https://github.com/CARV-ICS-FORTH/frisbee
```

Then you have to go install the Frisbee system.

```bash
>> cd charts/frisbee/charts/frisbee
>> helm install frisbee ./ --dependency-update
```

You will see a YAML output that describe the components to be installed.

#### Run the controller

After the Frisbee CRDs and their dependencies are installed, you can start running the Frisbee controller.

From the project's directory run

```bash
>> make run
```

#### Modify an examples test

Perhaps the best way is to modify an existing test. We use the `iperf` benchmark as a reference.

From another terminal (do not close the controller), go to

```bash
>> cd charts/iperf/
```

You will see the following files:

* **templates**:  libraries of frequently-used specifications that are reusable throughout the testing plan.
* **plans**: lists of actions that define what will happen throughout the test.
* **dashboards**: application-specific dashboards for Grafana

Because templates are used by the plans, we must have them installed before the running the tests.

```bash
>> helm install iperf ./ --dependency-update
```

Then, run the test to become familiar with the procedure.

```bash
>> kubectl apply -f plans/plan.validate-network.yml
```

If everything works fine, you will see the logs flowing in the **controller**.

Then you will get a message like

> Error: found in Chart.yaml, but missing in charts/ directory: chaos-mesh, openebs



This is because the Frisbee installation are not yet downloaded. To get them automatically run the previous command
with`--dependency-update` flag. Also, remove the `--dry-run` run to execute the actual installation.

```bash
>>  helm install frisbee ./ --dependency-update
```

Run the controller

== Operatator-SDK

=== Create a new Controller

* operator-sdk create api --group frisbee --version v1alpha1 --kind MyNewController --resource --controller
* operator-sdk create webhook --group frisbee --version v1alpha1 --kind MyNewController --defaulting
  --programmatic-validation

docker save frisbee:latest -o image.tar