# Frisbee in a Nutshell



## Installation

```bash
>> helm install my-deployment --create-namespace -n frisbee-testing ./
```

By default helm will use the Kubernetes configuration from `~/.kube/config`. 
If you wish to run Frisbee on a remote cluster, use `--kubeconfig=~/.kube/config.remote`



#### Run the controller

At first, we need to run the Frisbee operator.

There are three ways to run the operator:

- As Go program outside a cluster
- As a Deployment inside a Kubernetes cluster
- Managed by  the [Operator Lifecycle Manager (OLM)](https://sdk.operatorframework.io/docs/olm-integration/tutorial-bundle/#enabling-olm)  in [bundle](https://sdk.operatorframework.io/docs/olm-integration/quickstart-bundle) format.


To run the controller outside the cluster.

```bash
# Run Frisbee controller outside a cluster (from Frisbee directory)
>> make run
```





## Run a Test

The easiest way to begin with is by have a look at the examples.  Let's assume you are interested in testing TiKV.

```bash
>> cd examples/tikv/
>> ls
templates  plan.baseline.yml  plan.elasticity.yml  plan.saturation.yml  plan.scaleout.yml
```



You will some `plan.*.yml` files, and  a sub-directory called `templates`.

* **Templates:** are libraries of frequently-used specifications that are reusable throughout the testing plan.
* **Plans:** are lists of actions that define what will happen throughout the test. 



In general, a plan may dependent on numerous templates, and the templates depend on other templates. 

To run the test will all the dependencies satisfied:

```bash
# Create a dedicated Frisbee name
>> kubectl create namespace karvdash-fnikol

# Run the test
>> kubectl -n karvdash-fnikol apply -f ../core/observability/ -f ../core/ycsb/ -f ./templates/ -f templates/telemetry/ -f plan.baseline.yml
```



For TikV, the dependencies are:

* `./templates/` : TiKV servers and clients
* `./templates/telemetry` : Telemetry for TiKV servers (TiKV-specific metrics)

* `examples/templates/core/observability` : Telemetry for TiKV containers (system-wise metrics)
* `examples/templates/core/ycsb` : Telemetry for TiKV clients (YCSB-specific metrics)



> BEWARE: 
>
> 1) flag `-f` does not work recursively. You must explicitly declare the telemetry directory.
> 2) If you modify a template, you must re-apply it





#### Wait for completion

After you have successful ran a test, the next question that comes is about its completion.

```bash
# View deployed plans
>> kubectl -n karvdash-fnikol get workflows.frisbee.io
NAME            AGE
tikv-baseline   1m

# Inspect a plan
>> kubectl -n karvdash-fnikol describe workflows.frisbee.io tikv-baseline
```



Describe will return a lot of information about the plan. 
We are interested in the fields `conditions`. 

We can use these fields to wait until they become true -- thus saving us from manual inspection. 



```
# Wait until the test oracle is triggered.
>> kubectl  wait --for=condition=allActions workflows.frisbee.io/tikv-baseline -n karvdash-fnikol
workflow.frisbee.io/tikv-baseline condition met

```



#### Terminate a test

```bash
kubectl  delete -f tikv.elasticity.yml --cascade=foreground
```

The flag `cascade=foreground` will wait until the experiment is actually deleted.
Without this flag, the deletion will happen in the background. 

Use this flag if you want to run multiple tests, without interference.



### Debug a Test



At this point, the workflow is installed. You can go to the controller's terminal and see some progress.

If anything fails, you will it see it from there.





