#!/usr/bin/env bash

set -e -x

_dir=$(realpath "$(dirname "${BASH_SOURCE[0]}")")
_app_dir=$(realpath "$_dir/../../")
_root_dir=$(realpath "$_app_dir/../../")

source "$_app_dir/functions.sh"

MAKE="make -C $_root_dir"

set -a
AWS_REGION=${AWS_REGION:-"us-east-1"}
STACK_NAME=${STACK_NAME:-"AB-507"}
VARIANT="BenchmarkMediumAerospike"

AEROSPIKE_VARIANT="benchmark"

BIDDER_IMAGE_VERSION="d8cb00de08001c93bb7c759d78656f89d68fa029"
LOAD_GENERATOR_IMAGE_VERSION="ba1348aedaeeafa64e256a441c75d00a158e796e"
DATAGEN_IMAGE_VERSION="340bbd0cd5478d6ebd76e1231040a225bdaafdb4"

# Application config
#OVERLAY="benchmark"
#VALUES="$_dir/application.values.yaml"
set +a

output_dir="$_dir/outputs/4/"
mkdir -p "$output_dir"

########################################
## Step 1. Infrastructure
########################################
## Prepare the infrastructure
#$MAKE stack@deploy
#$MAKE eks@grant-access
#$MAKE eks@provision
#$MAKE aerospike@deploy
#wait_for_aerospike_stateful_set
#$MAKE aerospike@datagen
## Inspect the resources
ls_node_pools | tee "$output_dir/node_pools.txt"
## Optional wait period
#sleep 120

########################################
## Step 2A. Application (Deploy)
########################################
## Deploy the application
#$MAKE eks@use
#kubectl delete deployments.apps bidder || true
#OVERLAY="benchmark" VALUES="$_dir/application-standard.values.yaml" $MAKE eks@deploy
#OVERLAY="benchmark" VALUES="$_dir/application-large.values.yaml" $MAKE eks@deploy
#restart_bidder_deployment --step 10
### Inspect the configurations
kubectl get services bidder -o json | jq 'del(.metadata.managedFields)' | tee "$output_dir/services-bidder.json"
kubectl get deployments.apps bidder -o json | jq 'del(.metadata.managedFields)' | tee "$output_dir/deployments-apps-bidder.json"
kubectl get configmaps bidder-config -o json | jq .data | tee "$output_dir/configmaps-bidder-config-data.json"
## Optional wait period
#sleep 120

########################################
## Step 3. Test
########################################
## Delete jobs.batch resources
kubectl delete jobs.batch load-generator || true
kubectl delete jobs.batch report-aggregator || true
## Run test
TARGET="http://bidder/bidrequest" VALUES="$_dir/load-generator-constant.values.yaml" $MAKE benchmark@run-simple
#TARGET="http://bidder/bidrequest" VALUES="$_dir/load-generator-growing.values.yaml" $MAKE benchmark@run-simple
## Inspect the configuration
kubectl get jobs.batch load-generator -o json | jq 'del(.metadata.managedFields)' | tee "$output_dir/jobs-batch-load-generator.json"
## Wait for benchmark to complete
$MAKE benchmark@wait
## Collect data from load generators
LOAD_GENERATOR_NODE_SELECTOR_POOL=benchmark REPORT_FILE="$output_dir/aggregated-report-load-generator.md" $MAKE benchmark@report
## Collect the application logs stats
collect_application_logs_stats "$output_dir/application_logs_stats.txt"
## Optional wait period
#sleep 120
