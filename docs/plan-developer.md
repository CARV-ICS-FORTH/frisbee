## Guide for the Frisbee Plan Developers

* Spurious Alert may be risen if the expr evaluation frequency is less than the scheduled interval.
* In this case, Grafana faces an idle period, and raises a NoData Alert.
* The controller ignores such messages.

  # Periodically kill some nodes.
    - action: Cascade name: killer depends: { running: [ clients ] } cascade:
      templateRef: chaos.pod.kill instances: 3 inputs:
      - { target: .cluster.clients.one }

This can be wrong because Frisbee selects a single client -- and will be used 3 times, without error. Instead, we must
use as many inputs as the number of instances -- or omit instances.

In general, because when you use one input for multiple instances.

Macros select only Running objects

### Do not set dependencies on cascades

This is because Kills are always running -- therefore cascades that involve kill actions are always running