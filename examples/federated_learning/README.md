# ML Backend
This experiment is designed for the evaluation of various ML frameworks on the client.
For this purpose, we use a single client, and we change its backend.

```shell
kubectl frisbee submit test fedbed- examples/federated_learning/ml-backend.yml  ./charts/system/ charts/federated-learning/fedbed
```

# Resource Distribution
This experiment is designed for the evaluation of resource heterogeneity.
For this purpose, we use multiple clients and assign the total resources to clients according to a  distribution.

```shell
kubectl frisbee submit test fedbed- examples/federated_learning/resource-distribution.yml  ./charts/system/ charts/federated-learning/fedbed
```

# Client Placement
This experiment is designed for the evaluation of client placement across nodes.
In this case, we only use the placement primitives, without any kind of resource throttling

```shell
kubectl frisbee submit test fedbed- examples/federated_learning/client-placement.yml  ./charts/system/ charts/federated-learning/fedbed
```

# Dataset Distribution
This experiment is designed for the evaluation of dataset heterogeneity.
For this purpose, we use multiple clients and split the dataset to clients according to a  distribution.

```shell
kubectl frisbee submit test fedbed- examples/federated_learning/dataset-distribution.yml  ./charts/system/ charts/federated-learning/fedbed
```