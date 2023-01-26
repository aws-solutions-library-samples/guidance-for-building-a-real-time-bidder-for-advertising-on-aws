---
id: named-stacks
title: Infrastructure deployment
slug: /deployment
---

The infrastructure can be deployed as almost independent CloudFormation stacks.
The only one shared component is DynamoDB that is bound to the main stack called `bidder`.


# Deploying new stack

1. Pick a DNS-safe name of new CloudFormation stack (**`XYZ` is used in guide below**)

2. Export environment variables:

   * `STACK_NAME=XYZ`
   * optionally `VARIANT` described [here](../benchmarks/guide.md)
   * `AWS_REGION` if not working in `us-east-1`

3. Deploy the stack using `make stack@deploy`.
   **This may take up to 30 minutes.**

4. To configure `kubectl` with new cluster and grant access for rest of the team
   run `make eks@grant-access`.

   The team can connect to the cluster using `make eks@use`

5. Provision the new cluster with basic services (Prometheus stack and ExternalDNS) using `make eks@provision`.
   **This may take up to 5 minutes.**

6. If you want to use Aerospike, follow deployment steps described [here](./aerospike.md#deployment).

7. Optionally update application config: add a `values.yaml` with settings changed from
   `deployment/infrastructure/charts/bidder/values.yaml` and use its path in the next step.

   Some settings will be automatically changed to use the correct per stack resources if not overridden. Some other
   settings have defaults for use in benchmark configurations enabled by the `OVERLAY=benchmark` variable.

   There are three common so we have ready config overrides for them controlled by the `OVERLAY` environment variable:

   * `OVERLAY=basic`: the default, not additional configuration, ready for use on small instances with usual DynamoDB
     tables

   * `OVERLAY=benchmark`: run the bidder with static CPU allocation and multiple replicas on the big application pool
     nodes

   * `OVERLAY=tiny-tables`: run the bidder on small instances and access small on demand DynamoDB tables (ten devices,
     audiences, campaigns): use it for small tests that do not require much DynamoDB traffic

   * `OVERLAY=internal-nlb`: run the bidder with internal AWS Network Load Balancer

   * `OVERLAY=public-nlb`: run the bidder with public AWS Network Load Balancer

   For a more complex customization of a non-basic configuration,
   copy the `deployment/infrastructure/deployment/bidder/overlay-$OVERLAY.yaml` file into your custom `values.yaml`.

   You can pass multiple (comma separated) values to `OVERLAY`.

   For example `OVERLAY=tiny-tables,internal-nlb` will run the bidder on small instances with access small on demand
   DynamoDB tables and with internal AWS NLB.

8. Deploy the application using `make eks@deploy`, passing
   `BIDDER_IMAGE_VERSION=ABC` to deploy bidder tag `ABC` instead of `latest`
   or `VALUES=path/to/values.yaml` to override arbitrary settings.

   To consistently deploy the same version that is currently the latest one, run `make ecr@get-latest-tags` to see the
   commit hashes that can be used instead of the `latest` tag for our applications and tools.


# Removing the stack

To completely remove CloudFormation stack `XYZ`:

1. Export `STACK_NAME=XYZ`.

2. Run `make eks@use` to ensure that you are using correct EKS cluster

3. Remove the application from the cluster with `make eks@cleanup`.  
   If this step would be omitted, the CF stack cannot be removed due to dynamically created network load balancer
   that will be blocking the VPC from being deleted.

4. Remove CF stack using `make stack@delete`.  
   **This may take up to 10 minutes.**

   The command may fail in some situations. To check error messages, navigate to CloudFormation in AWS Console.
   Here is the list of possible situations:

   * **CF error:**   Unable to delete Kinesis stack. Failed to delete `[S3Bucket]`  
     **Reason:**     The application has been deployed with enabled Kinesis, bid request logs have been
                     saved to S3 bucket and they are preventing the stack from being deleted.  
     **Resolution:** Delete all objects from the S3 bucket and retry to delete the stack.

   * **CF error:**   Unable to delete VPC stack.  
     **Reason:**     The application has not been removed from the cluster before deleting the stack
                     and load balancer is still deployed to VPC.  
     **Resolution:** In AWS Console, navigate to EC2 -> Load balancers.  
                     Find Load balancer with tag `kubernetes.io/cluster/XYZ: owned` and remove it.
                     Retry to delete the stack.

# Deploying the application in non-default region

To deploy the application outside `us-east-1`:

1. Ensure that CloudFormation stack with IAM roles is deployed to the region.
   If not, deploy it using name `iam-{REGION}` (eg. `iam-us-east-2`)

    ```shell
    aws cloudformation create-stack
        --region eu-west-1 \
        --stack-name iam-eu-west-1 \
        --template-body file://deployment/infrastructure/iam.yaml \
        --capabilities CAPABILITY_NAMED_IAM
    ```

2. Ensure that ACM certificate has been issued in the region.
   If not, request a new wildcard certificate for application domain using AWS Console.
   Use DNS record validation with Route53.

3. Update Kubernetes service annotation `service.beta.kubernetes.io/aws-load-balancer-ssl-cert` with
   the ARN of certificate from step 2.
