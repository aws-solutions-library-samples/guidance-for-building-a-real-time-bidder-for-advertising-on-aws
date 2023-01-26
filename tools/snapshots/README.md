# Snapshot utils

This directory contains utils for creating and restoring snapshots 
of Kubernetes volumes managed by **PersistentVolumeClaim**.

## Available utils

### Create snapshot 

Command: `make snapshot@create`
Description: Creates and tags EBS volume snapshots from Kubernetes volumes managed by PVC.

Example:

```
make snapshot@create SNAPSHOT_APP=aerospike SNAPSHOT_PVC_PREFIX=data-aerospike SNAPSHOT_NAME=demo
```

### Restore snapshot 

Command: `make snapshot@restore`
Description: Restores PVC pointing to previously created EBS snapshots. 

Example:

```
make snapshot@restore SNAPSHOT_APP=aerospike SNAPSHOT_PVC_PREFIX=data-aerospike SNAPSHOT_NAME=demo
```

### Delete snapshot 

Command: `make snapshot@delete`
Description: Permanently deletes EBS snapshots.

Example:

```
make snapshot@delete SNAPSHOT_APP=aerospike SNAPSHOT_NAME=demo
```

## Common arguments

* `SNAPSHOT_APP` - application name
* `SNAPSHOT_NAME` - snapshot name
* `SNAPSHOT_PVC_PREFIX` - PVC name prefix (base name without the index and last dash)

To determine PVC name prefix:

1. list available PVCs with `kubectl get pvc` and locate desired PVC

  ```
  $ kubectl get pvc
  
  NAME              STATUS  VOLUME                                     CAPACITY   ACCESS MODES   STORAGECLASS   AGE
  data-aerospike-0  Bound   pvc-6840fc69-8aad-474b-a623-c43eeab18cd0   20Gi       RWO            csi-aws-ebs    19m
  data-aerospike-1  Bound   pvc-78abe3b2-d085-4b87-a7e5-ac33c01073c1   20Gi       RWO            csi-aws-ebs    19m
  ```

2. From PVC name (eg. `data-aerospike-0`) remove the part starting 
   from the last dash to get PVC prefix (eg. `data-aerospike`). 
