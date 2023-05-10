# Frisbee Changelog

## Changes Since Last Release

### Changed defaults / behaviours

- Changed default distribution to Constant
- Update callables to use 'main' entrypoint instead of the old 'app' entrypoint.
- Moved the watchPod implementation to the generic Watchers package.
- Callable does not fail immediately if it does not find a service, but it retries.
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