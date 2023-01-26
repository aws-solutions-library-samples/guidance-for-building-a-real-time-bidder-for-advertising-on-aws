---
id: aerospike
title: Aerospike deployment
slug: /aerospike
---

The application can use Aerospike 5.5.0.3 Community Edition database to retrieve data from.

Aerospike can be configured via following files:

* `deployment/infrastructure/deployment/aerospike/values.yaml` - Helm Chart options 
* `deployment/infrastructure/deployment/aerospike/aerospike.template.conf` - Node configuration template

More about the configuration can be found:

* [Aerospike documentation](https://www.aerospike.com/docs/operations/configure/index.html)
* [Helm Chart documentation](https://artifacthub.io/packages/helm/aerospike/aerospike)

## Accessing Aerospike

**Accessing from inside the cluster**

Host: `aerospike`
Port: `3000`
Authorization is disabled.

**Accessing from outside the cluster**

Use port forwarding to connect to the cluster:

```
kubectl port-forward svc/aerospike 3000:3000
```

Host: `localhost`
Port: `3000`
Authorization is disabled.

## Deployment

Aerospike is deployed on a dedicated node pool attached to the application Kubernetes cluster.
To let the CloudFormation to create the Aerospike node pool set the `AerospikeNodeGroupSize` parameter to value greater
than `0`.
Change the EC2 instance type using the `AerospikeInstanceType` parameter.

**Warning** Aerospike does not support ARM architecture.

1. Optionally [restore](#restoring-data) Aerospike data from the snapshot
2. Optionally configure the Aerospike
3. Deploy the database using `make aerospike@deploy AEROSPIKE_VARIANT=benchmark`

Where `AEROSPIKE_VARIANT` is one of:

* `basic` - dedicated for `Basic` infrastructure variant (2GB of space, replication factor = 1)
* `benchmark` - configuration set according to ADR from AB-142 (750GB of space, replication factor = 3)
* `data-load` - configuration for data load and snapshot (750GB of space, replication factor = 1)
                dedicated for `AerospikeDataLoad` infrastructure variant

The deployment takes about 1 minute per one node.

## Running Aerospike CLI tools

To run Aerospike tolls like `aql`, `asadm`, `asinfo`, use the following make target.

```shell
TOOL=asadm make aerospike@tool
```

The `TOOL` variable defaults to `aql`.

## Loading fresh data

Run the data generation tool (needs benchmark pool nodes):

```
make aerospike@datagen
```

Wait an hour (or two if using the replicated benchmark cluster) for the data to be ready.
Use aql `SHOW SETS;` command to verify that all data is loaded.

## Snapshots

### Creating data snapshot

If Aerospike is deployed, it should be removed (data will be preserved)
from the cluster with:

```
make aerospike@cleanup
```

You can create a snapshot from running instances, but it's not recommended.
"Hot" copies may contain corrupted data if some operation is executed by Aerospike 
(eg. data rebalancing or defragmentation) during the snapshot creation.

To create data snapshot named `demo`:

```
make aerospike@snapshot-create SNAPSHOT_NAME=demo
```

This takes about four hours for a snapshot with the complete data including the billion of device records.

### Restoring data

To be able to use data from the snapshot, you need at least same number of nodes 
in dedicated node group as have been available during snapshot creation.

If you would have more nodes, the data will be rebalanced in the cluster.

If you would have less nodes, the data may not be complete (it depends on replication 
factor of Aerospike namespace). 

If you have already deployed Aerospike in the cluster, first you need to delete the cluster and cleanup
the storage that have been used with:

```
`make aerospike@cleanup-storage`
```

To restore data from the snapshot named `demo`:

```
make aerospike@snapshot-restore SNAPSHOT_NAME=demo
```

It's not known how long a restore takes. The cluster might need 20 hours to copy data from the snapshot to its instance
storage and it will be interrupted (as the pods are not ready yet) once every 10 minutes.
