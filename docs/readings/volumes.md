# Persistent Volumes

A PersistentVolume (PV) is a piece of networked storage in the cluster that has been provisioned by an administrator. It
is a resource in the cluster just like a node is a cluster resource. PVs are volume plugins like Volumes, but have a
lifecycle independent of any individual pod that uses the PV.

This API object captures the details of the implementation of the storage, be that NFS, iSCSI, or a
cloud-provider-specific storage system.

A PV can be statically provisioned by an administrator or dynamically provisioned using Storage Classes.

## Static Provisioning

Sometimes, you don’t have block volumes provisioning available on your Kubernetes cluster. In particular when your
cluster is running on premise / outside the clouds providers.

In this case, the cluster administrator has to manually create a number of PVs. They carry the details of the real
storage, which is available for use by cluster user.

In this Chapter, we analyze how to provision automatically local PVs with
the [local static provisionner](https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner) provided by the
Kubernetes [SIG](https://github.com/kubernetes-sigs).

#### What is a Local Volume ?

A local volume is a mounted storage device like a disk, a partition or even a simple directory.

* **HostPath:** The volume itself does not contain scheduling information. If you want to fix each pod on a node, you
  need to configure scheduling information, such as nodeSelector, for the pod. **HostPath incures data loss is the pod
  is moved to a different node by the scheduler**


* **LocalVolume:** The volume itself contains scheduling information, and the pods using this volume will be fixed on a
  specific node, which can ensure data continuity. Local volumes can only be used as a statically created
  PersistentVolume. Dynamic provisioning is not supported. **LocalVolumes ensures that a pod with persistent data is
  always scheduled to the same worker node.**

You must set a PersistentVolume `nodeAffinity` when using `local` volumes. The Kubernetes scheduler uses the
PersistentVolume `nodeAffinity` to schedule these Pods to the correct node.

PersistentVolume `volumeMode` can be set to "Block" (instead of the default value "Filesystem") to expose the local
volume as a raw block device.

#### Local static Provisioner

An external static provisioner can be run separately for improved management of the local volume lifecycle. Note that
this provisioner does not support dynamic provisioning yet.

For an example on how to run an external local provisioner, see
the [local volume provisioner user guide](https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner).

The local volume static provisioner manages the PersistentVolume lifecycle for pre-allocated disks by detecting and
creating PVs for each local disk on the host, and cleaning up the disks when released. It does not support dynamic
provisioning.

#### Create Persistent Volume Claim

A PersistentVolumeClaim (PVC) is a request for storage by a user. It is similar to a pod. Pods consume node resources
and PVCs consume PV resources. Pods can request specific levels of resources (CPU and Memory). Claims can request
specific size and access modes (e.g., can be mounted once read/write or many times read-only).

**Binding:**

A control loop in the master watches for new PVCs, finds a matching PV  (if possible), and binds them together.

If a PV was dynamically provisioned for a new PVC, the loop will always bind that PV to the PVC.

Otherwise, the user will always get at least what they asked for, but the volume may be in excess of what was requested.

Claims will remain unbound indefinitely if a matching volume does not exist

When using local volumes, it is recommended to create a StorageClass with `volumeBindingMode` set
to `WaitForFirstConsumer`. For more details, see the
local [StorageClass](https://kubernetes.io/docs/concepts/storage/storage-classes/#local) example.

Delaying volume binding ensures that the PersistentVolumeClaim binding decision will also be evaluated with any other
node constraints the Pod may have, such as node resource requirements, node selectors, Pod affinity, and Pod
anti-affinity.

**Using:**

Pods access storage by using the claim as a volume.

Claims must exist in the same namespace as the Pod using the claim.

The cluster inspects the claim to find the bound volume and mounts that volume for a Pod.

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: mypod
spec:
  containers:
    - name: myfrontend
      image: nginx
      volumeMounts:
        - mountPath: "/var/www/html"
          name: mypd
  volumes:
    - name: mypd
      persistentVolumeClaim:
        claimName: myclaim
```

## Storage Classes (Dynamic Provisioning)

If none of the static persistent volumes match the user’s PVC request, the cluster may attempt to dynamically create a
PV that matches the PVC request based on storage class.

This provisioning is based on StorageClasses, the PVC must request a storage class and the administrator must have
created and configured that class for dynamic provisioning to occur.


> This PVC results in an PersistentVolume being automatically provisioned.
>
> When the claim is deleted, the volume is destroyed.



If you set up a Kubernetes cluster on GCP, AWS, Azure, or any other cloud platform, a default StorageClass creates for
you which uses the standard persistent disk type.

#### StorageClass Configuration

```yaml
---
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: standard
provisioner: kubernetes.io/aws-ebs
reclaimPolicy: Retain
volumeBindingMode: Immediate
```

Fields:

| Field             | Description                                                  |
| ----------------- | ------------------------------------------------------------ |
| provisioner       | determines what volume plugin is used for provisiong PVs.    |
| reclaimPolicy     | determines what happens when a persistent volume claim is deleted |
| volumeBindingMode | controls when volume binding and dynamic provisioning should occur. the default is *
Immediate*. Alternatively it can be *WaitForFirstCustomer* . |

* The classes.yml defines the top-level classes that are commonly used by Frisbee.
* The other files implement it for every different infrastructure.

#### Topology Awareness

In [Multi-Zone](https://kubernetes.io/docs/setup/multiple-zones) clusters, Pods can be spread across Zones in a Region.
Single-Zone storage backends should be provisioned in the Zones where Pods are scheduled. This can be accomplished by
setting the [Volume Binding Mode](https://kubernetes.io/docs/concepts/storage/storage-classes/#volume-binding-mode).

Sources:

https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/blob/master/docs/best-practices.md

https://docs.pingcap.com/tidb-in-kubernetes/dev/configure-storage-class

https://medium.com/avmconsulting-blog/persistent-volumes-pv-and-claims-pvc-in-kubernetes-bd76923a61f6

https://unofficial-kubernetes.readthedocs.io/en/latest/concepts/storage/persistent-volumes/