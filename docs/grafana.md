# Grafana API



Start by setting the grafana url

```bash
export grafana=http://grafana-skata-612.localhost
```



Assuming that the folder is `alarms`



#### Get Alarm Rules

```bash
curl ${grafana}/api/ruler/grafana/api/v1/rules/alarms
```





#### Set Alarms Rules

```
curl -H 'Content-Type: application/json' -d "@request.json" ${grafana}/api/ruler/grafana/api/v1/rules/alarms?subtype=cortex
```



the body of request.json:

```json
{
  "name": "malakies",
  "interval": "1m",
  "rules": [
    {
      "grafana_alert": {
        "title": "Correlated Outbound Network Throughput",
        "condition": "B",
        "no_data_state": "NoData",
        "exec_err_state": "Error",
        "data": [
          {
            "refId": "Throughput",
            "queryType": "",
            "relativeTimeRange": {
              "from": 8603,
              "to": 7468
            },
            "datasourceUid": "PBFA97CFB590B2093",
            "model": {
              "datasource": {
                "type": "prometheus",
                "uid": "PBFA97CFB590B2093"
              },
              "exemplar": true,
              "expr": "avg(rate(container_network_transmit_bytes_total{id=\"/\"}[$__rate_interval]))  != 0",
              "hide": false,
              "interval": "",
              "legendFormat": "Packets",
              "refId": "Throughput",
              "intervalMs": 15000
            }
          },
          {
            "refId": "Packets",
            "queryType": "",
            "relativeTimeRange": {
              "from": 8603,
              "to": 7468
            },
            "datasourceUid": "PBFA97CFB590B2093",
            "model": {
              "datasource": {
                "type": "prometheus",
                "uid": "PBFA97CFB590B2093"
              },
              "exemplar": true,
              "expr": "avg(rate(container_network_transmit_packets_total{id=\"/\"}[$__rate_interval]))  != 0",
              "hide": false,
              "interval": "",
              "legendFormat": "Throughput",
              "refId": "Packets",
              "intervalMs": 15000
            }
          },
          {
            "refId": "A",
            "datasourceUid": "-100",
            "queryType": "",
            "model": {
              "refId": "A",
              "hide": false,
              "type": "reduce",
              "datasource": {
                "uid": "-100",
                "type": "__expr__"
              },
              "conditions": [
                {
                  "type": "query",
                  "evaluator": {
                    "params": [],
                    "type": "gt"
                  },
                  "operator": {
                    "type": "and"
                  },
                  "query": {
                    "params": [
                      "A"
                    ]
                  },
                  "reducer": {
                    "params": [],
                    "type": "last"
                  }
                }
              ],
              "reducer": "last",
              "expression": "Throughput"
            }
          },
          {
            "refId": "B",
            "datasourceUid": "-100",
            "queryType": "",
            "model": {
              "refId": "B",
              "hide": false,
              "type": "threshold",
              "datasource": {
                "uid": "-100",
                "type": "__expr__"
              },
              "conditions": [
                {
                  "type": "query",
                  "evaluator": {
                    "params": [
                      0
                    ],
                    "type": "gt"
                  },
                  "operator": {
                    "type": "and"
                  },
                  "query": {
                    "params": [
                      "B"
                    ]
                  },
                  "reducer": {
                    "params": [],
                    "type": "last"
                  }
                }
              ],
              "expression": "A"
            }
          }
        ]
      },
      "for": "5m",
      "annotations": {
        "__dashboardUid__": "summary",
        "__panelId__": "158"
      },
      "labels": {}
    }
  ]
}

```





