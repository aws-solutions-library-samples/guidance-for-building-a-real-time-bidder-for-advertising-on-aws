#!/usr/bin/env bash

set -e -x

_dir=$(realpath $(dirname "${BASH_SOURCE[0]}"))
_app_dir=$(realpath "$_dir/../../")
_root_dir=$(realpath "$_app_dir/../../")

source "$_app_dir/functions.sh"

MAKE="make -C $_root_dir"

set -a
AWS_REGION=${AWS_REGION:-"us-east-1"}
STACK_NAME=${STACK_NAME:-"AB-429"}
VARIANT="BenchmarkSmallDax"
OVERLAY="benchmark"
VERSION="7d7ed39870a714defc45ee00c7d4e0c08dc476ad"
VALUES="$_dir/test.values.yaml"
set +a

# Prepare the infrastructure
# $MAKE stack@deploy
# $MAKE eks@grant-access
# $MAKE eks@provision

# Deploy the application
# $MAKE eks@use
# $MAKE eks@deploy

# Inspect the configurations
kubectl get services bidder -o json | jq 'del(.metadata.managedFields)'
kubectl get deployments.apps bidder -o json | jq 'del(.metadata.managedFields)'
kubectl get configmaps bidder-config -o json | jq .data

# Wait for bidders to start
target_replicas=$(kubectl get deployments.apps bidder -o json | jq .spec.replicas)
available_replicas=$(kubectl get deployments.apps bidder -o json | jq .status.availableReplicas)
updated_replicas=$(kubectl get deployments.apps bidder -o json | jq .status.updatedReplicas)
while [[ $available_replicas < $target_replicas || $updated_replicas < $target_replicas ]]; do
    sleep 30
    available_replicas=$(kubectl get deployments.apps bidder -o json | jq .status.availableReplicas)
    updated_replicas=$(kubectl get deployments.apps bidder -o json | jq .status.updatedReplicas)
done

# Run benchmark
$MAKE benchmark@cleanup
# VALUES="$_dir/test-3-ramp.values.yaml" $MAKE benchmark@run-simple
VALUES="$_dir/profiler.values.yaml" $MAKE benchmark@run-simple

# Inspect the configuration
kubectl get jobs.batch load-generator -o json | jq 'del(.metadata.managedFields)'

# Wait for benchmark to complete
$MAKE benchmark@wait

# Collect data from load generators
LOAD_GENERATOR_NODE_SELECTOR_POOL=benchmark REPORT_FILE="$_dir/load_generator_summary.md" $MAKE benchmark@report
