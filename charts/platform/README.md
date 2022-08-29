## Parameters

### Global parameters

| Name                  | Description                                                               | Value       |
| --------------------- | ------------------------------------------------------------------------- | ----------- |
| `global.domainName`   | DNS name for making Telemetry stack accessible outside the cluster.       | `localhost` |
| `global.ingressClass` | Type of ingres for making Telemetry stack accessible outside the cluster. | `public`    |


### Frisbee Operator parameters

| Name                            | Description                                                                | Value              |
| ------------------------------- | -------------------------------------------------------------------------- | ------------------ |
| `operator.enabled`              | Set it to false for running the controller outside the Kubernetes Cluster. | `true`             |
| `operator.name`                 | Defines the name of the controller.                                        | `frisbee-operator` |
| `operator.advertisedHost`       | Defines the Public IP of the controller, when operator.enabled==false.     | `10.1.96.192`      |
| `operator.webhook.k8s.enabled`  | Enables the Admission webhooks                                             | `true`             |
| `operator.webhook.k8s.port`     | Sets the port for the Admission/Mutation  webhook server.                  | `9443`             |
| `operator.webhook.grafana.port` | Sets the port for the telemetry webhook server.                            | `6666`             |


### Provision of dynamic volumes

| Name                               | Description                                          | Value                                                          |
| ---------------------------------- | ---------------------------------------------------- | -------------------------------------------------------------- |
| `openebs.enabled`                  | Whether to enable OpenEBS                            | `true`                                                         |
| `openebs.storagePath`              | The filesystem dir where volumes will be provisioned | `/mnt/local`                                                   |
| `openebs.nfs-provisioner.enabled`  | Will enable dynamic NFS servers                      | `true`                                                         |
| `openebs.ndm.enabled`              | Whether to enable block device management            | `true`                                                         |
| `openebs.ndm.filters.includePaths` | Include block devices for dynamic block provisioning | `""`                                                           |
| `openebs.ndm.filters.excludePaths` | Exclude block devices for dynamic block provisioning | `/dev/fd0,/dev/sr0,/dev/ram,/dev/dm-,/dev/md,/dev/rbd,/dev/zd` |


### Chaos Injection Parameters

| Name                                        | Description                               | Value                                           |
| ------------------------------------------- | ----------------------------------------- | ----------------------------------------------- |
| `chaos.enabled`                             | Whether to enable the Chaos controllers   | `true`                                          |
| `chaos-mesh.controllerManager.replicaCount` | Number of Chaos-Mesh controller replicas  | `1`                                             |
| `chaos-mesh.chaosDaemon.runtime`            | Specifies which container runtime to use. | `containerd`                                    |
| `chaos-mesh.chaosDaemon.socketPath`         | Specifies the container runtime socket.   | `/var/snap/microk8s/common/run/containerd.sock` |


### General purpose, web-based UI for Kubernetes clusters

| Name                | Description                 | Value  |
| ------------------- | --------------------------- | ------ |
| `dashboard.enabled` | Whether to enable Dashboard | `true` |


