---
id: pinning-to-cpu-cores
title: Pinning bidder instances to separate CPU core
slug: /pinning-to-cpu-cores
---

# Pinning bidder instances to separate CPU cores

The bidder instances can be pinned to separate CPU cores on Kubernetes cluster.
To achieve that, all of the following sentences must be true:

* EKS is deployed with parameter `StaticCPUManagerPolicy: 1`
* CPU limit for the bidder is set to an integer (full cores) 
* memory limit is specified
* CPU/memory requests are not set or are set to the same values as the limits

## Confirming pinning to CPU cores is in effect

To confirm that application deployment is configured correctly, run:

```shell
kubectl get pods -o custom-columns=name:.metadata.name,qos:.status.qosClass -l app=bidder
```

Values in `qos` column should be `Guaranteed`.

You can also check CPU affinity directly on the worker node. After establishing connection 
to the worker node, run:

```shell
for pid in $(pgrep bidder); do taskset -cp $pid; done;
```

Sample output if pinning to CPU cores is **enabled**:

```
pid 14742's current affinity list: 3,4
pid 14830's current affinity list: 5,6
pid 14944's current affinity list: 1,2
pid 15069's current affinity list: 9,10
pid 15173's current affinity list: 7,8
...
```

Sample output if pinning to CPU cores is **disabled**:

```
pid 17461's current affinity list: 0-63
pid 17462's current affinity list: 0-63
pid 17489's current affinity list: 0-63
pid 17648's current affinity list: 0-63
pid 17773's current affinity list: 0-63
...
```

# Notes

On `m6g.16xlarge` with 64 vCPUs we were able to allocate 59 cores exclusively.
There rest of cores are used for the OS and daemon sets containers that use shared CPU pool.
