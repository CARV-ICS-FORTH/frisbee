Some good scenarios to begin with are:

    Add latency
    Make services and dependencies unavailable
    Throw exceptions randomly
    Cause packet loss across the network
    Fail requests by dropping them or providing failure responses
    Add resource contention such as CPU hogging processes, high network volume, broken sockets, unavailable file descriptors

# Testing Coverage

https://www.confluent.io/blog/measure-go-code-coverage-with-bincover/

# Benchmark Evaluation

https://plugins.jenkins.io/benchmark-evaluator/#documentation

# Monitoring

https://www.replex.io/blog/kubernetes-in-production-the-ultimate-guide-to-monitoring-resource-metrics

https://play.grafana.org/d/vmie2cmWz/bar-gauge?orgId=1

( Prometheus - native )
https://tikv.org/docs/3.0/tasks/monitor/tikv-cluster/

# Multi-cluster Kubernetes deployments

https://www.cockroachlabs.com/docs/stable/orchestrate-cockroachdb-with-kubernetes-multi-cluster.html

https://docs.mongodb.com/manual/core/replica-set-architecture-geographically-distributed/

# Tasks & Data

## Collocate tasks:  Place multiple tasks on the same pod to minimize communication and throttling overheads

## Coloccate tasks with data: Place tasks on the node as the data (e.g, volume)

## Align tasks with data: Migrate data from one volume to another, in order to have a volume, collocated with a task, to contain all the data required by that task.
