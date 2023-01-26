#!/usr/bin/env bash

set -e -x

_dir=$(realpath "$(dirname "${BASH_SOURCE[0]}")")
_app_dir=$(realpath "$_dir/../../")
_root_dir=$(realpath "$_app_dir/../../")

source "$_app_dir/functions.sh"

MAKE="make -C $_root_dir"

set -a
AWS_REGION=${AWS_REGION:-"us-east-1"}
STACK_NAME=${STACK_NAME:-"AB-629"}
VARIANT="BenchmarkAB-629"

BIDDER_IMAGE_VERSION="AB-629-Full-system-benchmark-with-DynamoDB-v2"
LOAD_GENERATOR_IMAGE_VERSION="AB-629-Full-system-benchmark-with-DynamoDB-v2"
set +a

output_dir="$_dir/outputs/1/"
mkdir -p "$output_dir"

########################################
## Step 1. Infrastructure
########################################
## Prepare the infrastructure
$MAKE stack@deploy
$MAKE eks@grant-access
$MAKE eks@provision
## Inspect the resources
ls_node_pools | tee "$output_dir/node_pools.txt"

########################################
## Step 2. Application (Deploy)
########################################
## Deploy the application
$MAKE eks@use
$MAKE eks@cleanup || true
OVERLAY="benchmark" VALUES="$_dir/application.values.yaml" $MAKE eks@deploy
### Inspect the configurations
kubectl get services bidder -o json | jq 'del(.metadata.managedFields)' | tee "$output_dir/services-bidder.json"
kubectl get deployments.apps bidder -o json | jq 'del(.metadata.managedFields)' | tee "$output_dir/deployments-apps-bidder.json"
kubectl get configmaps bidder-config -o json | jq .data | tee "$output_dir/configmaps-bidder-config-data.json"

########################################
## Step 3. Test
########################################
## Delete jobs.batch resources
$MAKE benchmark@cleanup || true
## Run test
TARGET="http://bidder/bidrequest" VALUES="$_dir/load-generator.values.yaml" $MAKE benchmark@run
## Inspect the configuration
kubectl get jobs.batch load-generator -o json | jq 'del(.metadata.managedFields)' | tee "$output_dir/jobs-batch-load-generator.json"
## Wait for benchmark to complete
$MAKE benchmark@wait
## Collect data from load generators
LOAD_GENERATOR_NODE_SELECTOR_POOL=benchmark REPORT_FILE="$output_dir/aggregated-report-load-generator.md" $MAKE benchmark@report
## Collect the application logs stats
collect_application_logs_stats "$output_dir/application_logs_stats.txt"
