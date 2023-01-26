# Infrastructure

Here's the complete AWS Stateless Bidder infrastructure guide. We are using AWS resources and "Infrastructure as a Code" approach. Every AWS resource is reflected in CloudFormation manifest. We are also using Kubernetes cluster to serve applications. Changes 'by hand' are not recommended, because they are not reflected in CloudFormation manifests.

Applications deployment to development environment is fully automatic. After merge into **master** branch, CI/CD pipeline (AWS CodePipeline) builds required artifacts, pushes them into AWS ECR repository and finally launches new version. To work with AWS Graviton2 processors (ARM) and x86 architecture, ECR images are built with both architectures.

## Tools required to manage infrastructure

### Accessing AWS resources

To access AWS resources, proper AWS keys should be configured with permissions and **aws cli** should be installed. Version 2 is recommended.

[https://docs.aws.amazon.com/cli/latest/userguide/install-cliv2.html](https://docs.aws.amazon.com/cli/latest/userguide/install-cliv2.html)

### Accessing Kubernetes resources

To access Kubernetes cluster, we must have **kubectl** tool installed. More on that here: [https://kubernetes.io/docs/tasks/tools/install-kubectl/](https://kubernetes.io/docs/tasks/tools/install-kubectl/)

[https://docs.aws.amazon.com/eks/latest/userguide/getting-started.html](https://docs.aws.amazon.com/eks/latest/userguide/getting-started.html)

### Configuring access to Kubernetes cluster

Here's the example of getting access to Kubernetes cluster:

```bash
make eks@use
```

To get access for another non-main cluster use:

```bash
make eks@use STACK_NAME=XYZ
```

Commands above creates **~/.kube/config** configuration file with all required credentials.

## Kubernetes cluster

We're using AWS EKS managed cluster along with managed NodeGroups. Right now we have 4 node groups:

- basic-arm - Graviton2 spot instances (m6g.medium or m6g.large), for app development and simple benchmarking
- basic-x86 - x86-backed instances for workloads other then ARM (kube-state-metrics, Grafana image renderer)
- application, benchmark - Graviton2 spot instances (m6g.16xlarge) for benchmarks

More on managed node groups you can find here:

[https://docs.aws.amazon.com/eks/latest/userguide/managed-node-groups.html](https://docs.aws.amazon.com/eks/latest/userguide/managed-node-groups.html)

## Accessing Prometheus and Grafana

To have basic observability and metrics, we're using Prometheus and Grafana stack.

For now, these services are accessed only from **kubectl port-forward** command. In future load balancer should be deployed to access metric stack from outside of Kubernetes cluster.

### Access Prometheus

```bash
kubectl port-forward svc/prometheus-operated 9090:9090
```

Now access [http://localhost:9090](http://localhost:9090) locally from your browser.

### Access Grafana

```bash
kubectl port-forward svc/prom-grafana 8080:80
```

Now access [http://localhost:8080](http://localhost:8080) locally from your browser.

Login with credentials:

- username: **admin**
- password: **prom-operator**


# AWS CodeBuild / CI
We use AWS CodeBuild in order to run our CI tasks. Their definitions are stored in deployment/infrastructure/ci directory.
Every CodeBuild project requires two files:
* `*-stack.yaml` - a CloudFormation template that keeps information about the resources.
* `*-buildspec.yaml` - definitions of the build phases of the project (shell scripts etc.).

When you need to change the build project and deploy the change on CI, just run:

        make ci@deploy PROJECT_NAME=ci-bidder-unit

When you change code of a BuildSpec, it is applied after the changes are merged to the main branch of the repository.

**IMPORTANT**
Remember that CI is used by the other people in the team, and plan your changes/adjustments avoiding CI distruptions.

# Security

Everytime a pull-request is pushed, we run security scans.
We validate the security of the package using the tools suggested by the AWS security team:
* viperlight
* cfn_nag

To run them locally:

    make cfn_nag@run
    make viperlight@run

To run security linters without worrying about the list of currently required linters:

    make security@lint
