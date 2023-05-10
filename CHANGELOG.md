# Frisbee Changelog

## Changes Since Last Release

### Changed defaults / behaviours

- Changed default distribution to Constant
- ...

### New Features & Functionality

- Templated federated learning clients can connect to multiple servers
- Added a constant distribution with all elements having a probability of 1.
- ...

## Bug Fixes

- Add warning message on `kubectl-frisbee` when no config is found.
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