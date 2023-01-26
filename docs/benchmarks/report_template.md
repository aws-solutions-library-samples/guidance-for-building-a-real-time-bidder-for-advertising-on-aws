# Benchmark Report

Date: 2021-01-04

Scenario: Unit benchmark

Duration: 5m

Throughput: 18,000 requests/second (constant)

Number of regions: 1 (us-east-1)

## Setup

### Application

Number of machines: 1

Machine type: m6g.8xlarge

Number of pods: 1

Profiler: disabled

Version: ???

Description:
Does the application use cache for budgets, campaigns, or devices? Does it use message buffering in front of the stream
writer? Does it has any specific improvements?

### Benchmark tool

Number of machines: 1

Machine type: m6g.8xlarge

Number of pods: 1

Version: ???

Tool: Vegeta

Rate: 18,000 requests/second

Number of devices: 1,000,000,000

Number of campaigns: 1,000,000

Description:
Should the reader be aware of any customizations?

### Infrastructure

Kinesis shards: 18

DynamoDB provisioned throughput (dev): 9,000 RCU

DynamoDB DAX: no

Network architecture: NLB, 3 AZs, 1 public subnet per AZ. The application is exposed via Kubernetes service resource in
ClusterIP mode. The application and load generator are located in the same EKS cluster, but on different node groups.
All nodes are in the same availability zone and subnet.

Description:
Should the reader be aware of any customizations?

### Additional description

No additional description.

## Results

_TBD: format_

## Analysis and Conclusions

Here is a few helper questions:

* Did the change provide expected performance?

* Did the change improve / worsen the performance?

* What did we learn about the application / benchmark tool / infrastructure?

* Should we change something to the application / benchmark tool / infrastructure / benchmark procedure?
