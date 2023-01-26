---
title: "Guide: How to run a benchmark?"
---

This guide explains step by step how to run a benchmark. 

## Authenticate with Kubernetes cluster

Make sure that you are authenticated your aws-cli with correct profile.
Then, update kubernetes config to connect to bidder's Kubernetes cluster

```shell
make eks@use
```

or (if you using [independent stack](../infrastructure/deployment.md) `XYZ`): 

```shell
make eks@use STACK_NAME=XYZ
```

## Bootstrap the infrastructure

The infrastructure is configured in a number of variants for the application stack and DynamoDB.
Depending on what is the goal of the benchmark we can use one of predefined variants,
or create a new one.

The variant configurations are available in `deployment/infrastructure/application.yaml`. 

```shell
make stack@deploy VARIANT=BenchmarkHTTP STACK_NAME=XYZ
```

### Scaling DynamoDB

The DynamoDB can be scaled independently of the application stack.
Depending on what is expected usage of the service **by all of the deployed stacks**
we can use one of predefined variants, or create a new one.

Preconfigured variants for DynamoDB:

* `Basic` - all DynamoDB tables configured to use 25 RCU and 25 WCU
  
* `BenchmarkDynamoDB`:
  * 40000 RCU and 25 WCU for `dev` table (mapping devices to audiences)
  * 5000 RCU and 25 WCU for `audience_campaigns` table
  * 5000 RCU and 25 WCU for `campaign_budget` table
  * 100000 RCU and 25 WCU for `budget` table

* `BenchmarkAutoscale`:
  * 40000-120000 RCU and 25 WCU for `dev` table
  * 5000-20000 RCU and 25 WCU for `audience_campaigns` table
  * 5000-20000 RCU and 25 WCU for `campaign_budget` table
  * 100000-400000 RCU and 25 WCU for `budget` table

The variant configurations are available in `deployment/infrastructure/dynamodb.yaml`.

```shell
make dynamodb@update VARIANT=BenchmarkDynamoDB
```

In some rare situation, e.g. when switching from variant with autoscaling to variant without autoscaling
and with the same minimum capacity as previous one, the capacity can be not updated automatically.
In that case, the above command will produce list of drifted resources that need to be updated manually in AWS Console.

### Get application versions

All deployment scripts default to using the latest versions of the bidder and tools. When new versions are built, these
affect benchmarks and require different configuration values, so we recommend passing specific commit hashes or git
tags of versions to deploy.

To get the tags corresponding to the `latest` tag (of the current master branch as of the last change affecting the
application):
```shell
make ecr@get-latest-tags
```
Alternatively, push tags of the commit to use: all applications will be built and tagged so.

Then set the environment variables with commit hashes or tags:
```shell
export BIDDER_IMAGE_VERSION=  # bidder
export LOAD_GENERATOR_IMAGE_VERSION=  # load-generator
export REPORT_AGGREGATOR_IMAGE_VERSION=  # report-aggregator
```
or adjust your `VALUES` files to set `image.tag` (all other ways of passing this value get overridden by the
environment variables if invoking Helm via `make` targets).

### Patch the application configuration

Changes that are applied:

* node selector matching the newly created larger nodes,
* increase the number of replicas to 32,
* apply resource limitations.

The configuration is available in `deployment/infrastructure/bidder-benchmark.yaml`.

```shell
make eks@deploy OVERLAY=benchmark
```

For different configurations, either create a custom file named `deployment/infrastructure/bidder-ABC.yaml` and deploy via `make
eks@deploy OVERLAY=ABC` or store the file elsewhere and deploy via `make eks@deploy VALUES=path/to/values.yaml`. See
`deployment/infrastructure/charts/bidder/values.yaml` for all settings and their default values.

To scale the application to desired number of instances use `kubectl scale` command.

#### Load balancing

By default, the bidder is deployed without AWS load balancer and uses only Kubernetes load balancing.

If you want to use AWS load balancer, pass additional `OVARLAY` to `make eks@deploy`:

* public AWS NLB: `make eks@deploy OVERLAY=benchmark,public-nlb`
* internal AWS NLB: `make eks@deploy OVERLAY=benchmark,internal-nlb`

...or override settings in your `values.yaml`. Look for examples in files:

* `deployment/infrastructure/bidder-public-nlb.yaml`
* `deployment/infrastructure/bidder-internal-nlb.yaml`

You'll also need to provide load balancer's ssl certificate uuid. You'll find/create one in certificate manager in 
AWS Console for the region you're deploying. Copy it's identifier and add 
`CERTIFICATE_UUID=certificate-uuid` parameter to the command 
ie. `make eks@deploy OVERLAY=benchmark,internal-nlb CERTIFICATE_UUID=certificate-uuid`.

### Start load generator

The load generator runs as a Kubernetes job.
The load generator container has a limit of two CPUs, and is run with exact number of 16 workers.
These setting allow it to easily generate 5,000 requests per second.
To achieve higher throughput you must run more instances. 

To configure the load generator use the following environment variables.

```shell
export TIMEOUT=100ms        # Request timeout (default 100ms)
export DURATION=500s        # duration of the load generation
export RATE_PER_JOB=5000    # target request rate for the load generator
export NUMBER_OF_DEVICES=10 # number of device IFAs to use in bid request
export NUMBER_OF_JOBS=1     # number of parallel instances of the load generator
export SLOPE=0              # slope of requests per second increase (zero for a constant rate; see <https://en.wikipedia.org/wiki/Slope>)
export ENABLE_PROFILER=1    # used to start profiling session, leave unset to disable
```

To use internal Kubernetes load balancing:

```shell
export TARGET=http://bidder/bidrequest
```

To target AWS load balancer:

```shell
# autogenerate the target
export STACK_NAME=XYZ

# ... or specify the target manually
export TARGET=https://XYZ.us-east-1.ab.clearcode.cc/bidrequest
```

To target multiple AWS load balancers:

```shell
export STACK_NAME=XYZ
export SERVICE_COUNT=1 # pass the same number as you have passed to `make eks@deploy`
```

#### Run the benchmark

```shell
make benchmark@run
```

To wait for the benchmark to complete use:

```shell
make benchmark@wait
```

## Collect load generator report

The load generator writes its results to the container's standard output.
To collect of the results and prepare an aggregated report run

```shell
make benchmark@report
```

You can also pass to above command `REPORT_FILE=report.txt` to save the report to `report.txt` file.

## View charts in Grafana

Create tunnel to the Grafana instance.

```shell
kubectl port-forward svc/prom-grafana 8080:80
```

[Visit the Grafana.](http://localhost:8080/)

## Delete leftovers after running load generator

```shell
make benchmark@cleanup
```

## Delete the infrastructure

Deploy the `Basic` variant of the infrastructure.

```shell
make eks@deploy
make stack@deploy STACK_NAME=XYZ
make dynamodb@update
```
