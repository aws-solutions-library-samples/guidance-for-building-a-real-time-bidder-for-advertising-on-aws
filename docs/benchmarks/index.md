# Benchmarks

We run benchmarks to analyse the performance of the system, or its parts,
in various scenarios.

## Procedure

The steps described bellow are automated, please check the [guide how to run a benchmark](guide.md).

### Phase 1) Define benchmark objectives and background.

Define the benchmark duration,
the volume of requests _(targeted throughput)_ to send,
and how does the volume change over time _(growing, constant)_.

Define the application setup:

* number and type of machines,
* number of deployed pods, and its configuration,
* version of application, and its configuration,
* note whether a profiler was enabled.

Define the benchmark tool setup:

* number and type of machines,
* number of deployed pods, and its configuration,
* version and name of used tool, and its configuration,
* number of devices and cmapaigns used.

Define the remaingin infrastructure setup:

* number of Kinesis shards,
* provisioned throughput for the DynamoDB tables,
* DynamoDB DAX configuration,
* network architecture.

### Phase 2) Prepare the environment.

1. Shutdown all resources that generate requests to the bidder.

2. Create a new node group that will handle the workload.

3. Patch the bidder deployment that the application will deploy on the new node group.
   Apply changes to the application and Kubernetes configuration.

4. Deploy new Kinesis Data Stream cluster.
   Adjust the number of shards accordingly to the targeted throughput.

5. Configure the provisioned throuput for the DynamoDB tables accordingly to the targeted throughput.

6. Create a new node group that will run the benchmark tool.

### Phase 3) Run the benchmark.

The current benchmark tool does not support a distributed load testing.
The distribution is achieved by running multiple instances of the tool in parallel.
For this purpose we'll use [Kubernetes job controller](https://kubernetes.io/docs/concepts/workloads/controllers/job/).

### Phase 4) Collect the metrics and document the benchmark.

The telemetry data is collected in a Prometheus instance.

The benchmark report includes following metrics:

* load balancer throughput (number of request per second),

* load balancer response latencies at .9, .95, .99 percentiles,

* bidder application throughput,

* bidder application HTTP request handling duration at .9, .95, .99 percentiles,

* bidder application auction duration at .9, .95, .99 percentiles,

* total number of requests to the DynamoDB tables (cummulative for all tables) over time,

* DynamoDB request latencies measure within the bidder application
  (includes DynamoDB latency and network latency),

* total number of writes to the Kinesis data streams (cummulative for all streams) over time,

* latency for Kinesis writes measure within the bidder application,

* number of messages buffered to write to the Kinesis data streams over time,

* CPU consumption per node over time,

* memory consumption per node over time.

Use the queries for retrieving the benchmark results to gather metrics.

Use the [benchmark report template](report_template.md) to describe the benchmark setup and results.

### Phase 5) Tear down

1. Remove node groups used for running workload and benchmark tools.

2. Deploy the bidder application on the basic cluster.

3. Run the resources that generate the requests in a regular deployment.

## Scenarios

_Benchmark setup: Run as many instances as needed to reach targeted throughput._


### Unit

The unit benchmark allows to analyse performance of a single instance of the application.

It is deployed as a single pod on a single machine.

It let us observe the performance and behavior of a single instance,
it helps estimate the resource allocation for a distributed environment.

### Pod per node

A single instance of application is deployed per node.

It should allow us to observe whether the application can scale linearly.

### Random allocation

It leaves the allocation of resources to the Kubernetes.

It is the most realistic scenario.

## References

* [Kubernetes job controller](https://kubernetes.io/docs/concepts/workloads/controllers/job/)
