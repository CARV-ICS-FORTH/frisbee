## Parameters

### Telemetry

| Name                                      | Description                                                                      | Value        |
| ----------------------------------------- | -------------------------------------------------------------------------------- | ------------ |
| `telemetry.grafana.port`                  | Listening port for Grafana                                                       | `3000`       |
| `telemetry.prometheus.name`               | The name of the prometheus service                                               | `prometheus` |
| `telemetry.prometheus.port`               | Listening port for Prometheus                                                    | `9090`       |
| `telemetry.prometheus.honorTimestamp`     | Use the timestamps of the metrics exposed by the agent (time-drifts)             | `true`       |
| `telemetry.prometheus.queryLookbackDelta` | The maximum duration for retrieving metrics for considering the source as stale. | `1m`         |
| `telemetry.dataviewer.port`                | Listening port for Dataviewer                                                     | `80`         |


### Chaos

| Name                                        | Description                                                                          | Value       |
| ------------------------------------------- | ------------------------------------------------------------------------------------ | ----------- |
| `chaos.network.generic.source`              | A list of comma separated services to apply the fault                                | `""`        |
| `chaos.network.generic.duration`            | The duration of the fault                                                            | `2m`        |
| `chaos.network.partition.partial.dst`       | A list of comma seperated services that will be partitioned from the source servies. | `""`        |
| `chaos.network.partition.partial.direction` | The direction of the network partition fault                                         | `both`      |
| `chaos.network.duplicate.duplicate`         | Percent of Duplicate packets                                                         | `40`        |
| `chaos.network.duplicate.correlation`       | Affinity to last packet. Emulates packet burst duplicates.                           | `25`        |
| `chaos.network.loss.loss`                   | Percent of Random packet loss                                                        | `25`        |
| `chaos.network.loss.correlation`            | Affinity to last packet. Emulates packet burst losses.                               | `25`        |
| `chaos.network.delay.latency`               | Per-Packet Latency                                                                   | `90ms`      |
| `chaos.network.delay.correlation`           | Affinity to last packet. Emulates packet burst delays.                               | `25`        |
| `chaos.network.delay.jitter`                | Add randomness in the delay                                                          | `90ms`      |
| `chaos.pod.kill.target`                     | Service to kill                                                                      | `localhost` |


