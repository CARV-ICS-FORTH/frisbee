# Frisbee Changelog

## Changes Since Last Release

### Changed defaults / behaviours
- Prevent cadvisor from failing when cgroup is not mounted.

### New Features & Functionality
- ...

## Bug Fixes
- ...

## 1.0.43 \[2023-08-18\]

### Changed defaults / behaviours
- Move basic images from /deploy to /images 
- ...

### New Features & Functionality
- Add autocompletion for uninstall command.
- Add validation for missing callable.
- Add flag for disabling chaos controller.
- Add flat on system charts for setting the inotify limits.
- ...

## Bug Fixes
- Remove VirtualObjects and Templates from the list of CRDs whose finalizers can be forcibly deleted (they have no finalizers).
- Fix issues with partial and conditional deployments (Ingress, chaos, ..)
- Fix platform chart to disable kubernetes dashboard according to the chart values.
- Fix the way embedded files are coped to the host filesystem.
- Remove single quote from cadvisor as it may not be supported on various backends, such as HPK.
- ...


## 1.0.42 \[2023-06-27\]

### Changed defaults / behaviours
- Moved charts from `charts/{category}` to `examples/apps`. This allows to have the apps and the test-cases on the same directory. Additionally, that
means that chart releasing is no longer part of the frisbee release -- which shouldn't have been the case in the first place.
- Renamed template to be in the format 'frisbee.system...' and 'frisbee.apps'. This, however, warrants a new release because
the renamed systems templates affect the controller.
- ...

### New Features & Functionality
- ...

## Bug Fixes
- Stalled cached files were used in reporting. Update the cached files every time we run the report command.
- ...


## 1.0.41 \[2023-06-22\]

### Changed defaults / behaviours
- Upgrade to Grafana v9.4.7 in order to avoid issues with locking database (https://github.com/grafana/grafana/issues/60703)
- Change Grafana requirements to 2 CPU and 14 Gi memory.
- Separate the timeouts for Kubernetes API, Grafana, and Interactions (logs, reports).
- Disable Kubernetes dashboard by default.
- ...

### New Features & Functionality
- Upgrade API to Kubernetes v0.27.2. Upgrade all other deps to the latest version.
- Fixed kubectl-frisbee to monitor multiple pods in parallel.
- Add examples for running sequences of YCSB workloads
- Add verification of resources and fix erroneous examples.
- ...

## Bug Fixes
- Added --network host to "make docker-build" to fix the network discovery issues.
- Fix the Constant distribution. It was returning normalized values.
- Fix --deep flag on test inspection.
- ...

## 1.0.40 \[2023-05-23\]


### Changed defaults / behaviours

- Change Grafana background to white so that reports are almost paper-ready.
- Fix pupetteer to v19.11.0 in order to avoid crashing due to erroneously parsing specific dashboards (i.e scenario events).
- Set maximum log streams to 100 in order to monitor up to medium-scale experiments. For more than 100 endpoints use persistent logs.
- Fix groupped actions (Call, Cluster, Cascade) to use the cluster view for decision-making instead of the Action's status.
  This is to avoid double-action causes by re-queued requests that maintain older state.
- Change Annotation priority to: Failed, Chaos, App, Create, Delete.
- Instead of annotating the lifecycle of Pods, we annotate the lifecycle of Service. This is to support capturing Failures.
- If a Pod is deleted, either manually or by a chaos event, it is considered as failed.
- ...

### New Features & Functionality

- Add fl example for injecting partition as specific epoch
- Added an annotator sidecar for pushing annotations to Grafana directly from the app's pod
- Add a new example in the tutorial that combines cluster with fault tolerance, and cascade of killings.
- ...

## Bug Fixes

- --shell flag of kubectl-inspect takes kubeconfig from environment.
- use background=true on pdf-exports to preserve the legend color
- fix YCSB dashboard to show current time, and update the annotations.
- Remove terminating functions from status classified as it complicates the deletion FSM.
- ...

## 1.0.39 \[2023-05-12\]


### Changed defaults / behaviours

- Changed default distribution to Constant
- Update callables to use 'main' entrypoint instead of the old 'app' entrypoint.
- Moved the watchPod implementation to the generic Watchers package.
- Callable does not fail immediately if it does not find a service, but it retries.
- On Grafana annotations print the extracted Kind of the object instead of the reflected type.
- Change Grafana background to white so that reports are almost paper-ready.
- ...

### New Features & Functionality

- Templated federated learning clients can connect to multiple servers
- Added a constant distribution with all elements having a probability of 1.
- Add support to forcibly delete tests (by removing its finalizers)
- Make the Calls asynchronous (otherwise it creates issues on parallel tests)
- Added testing patterns for databases
- ...

## Bug Fixes

- Add warning message on `kubectl-frisbee` when no config is found.
- Fix `kubectl-frisbee uninstall` to avoid blocking when the controller crashes.
- Fix Call to avoid 'metadata.resourceVersion: Invalid value: "": must be specified for an update'
- --wait on `kubectl-frisbee report` was using as test name the scenario name instead of scenario namespace.
- Fix table representation of VirtualObjects
- Updated callables example to start enumeration from idle-1 instead of idle-0
- ...

## 1.0.38 \[2023-05-05\]

### Changed defaults / behaviours

- The `report` command has been changed to download all metrics and annotations from Grafana.


### New Features & Functionality

- Added more scenarios on federating learning
- Add autocompletion on kubectl-frisbee
- Add wait function on report

## Bug Fixes

- Upgrade golang.org/x/net to v0.7.0 to avoid security bugs
- Fix `kubectl-frisbee report` to evaluate dashboard variables (returns `.+` always).

## 1.0.37 \[2023-02-02\]
