#!/usr/bin/env bash

set -e -x

_dir=$(realpath $(dirname "${BASH_SOURCE[0]}"))
_app_dir=$(realpath "$_dir/../../")
_root_dir=$(realpath "$_app_dir/../../")

source "$_app_dir/functions.sh"

MAKE="make -C $_root_dir"

set -a
AWS_REGION=${AWS_REGION:-"us-east-1"}
STACK_NAME=${STACK_NAME:-"AB-430"}
VARIANT="BenchmarkMediumDax"
OVERLAY="benchmark"
VERSION="7d7ed39870a714defc45ee00c7d4e0c08dc476ad"
VALUES="$_dir/test.values.yaml"
set +a

output_dir="$_dir/outputs/test-workers-32"
mkdir -p "$output_dir"

########################################
## Step 1. Infrastructure
########################################
## Prepare the infrastructure
$MAKE stack@deploy
$MAKE eks@grant-access
$MAKE eks@provision
## Inspect the resources
kubectl get nodes -l pool=basic-arm -o json | jq 'del(.metadata.managedFields)' | tee "$output_dir/nodes-basic-arm.json"
kubectl get nodes -l pool=application -o json | jq 'del(.metadata.managedFields)' | tee "$output_dir/nodes-application.json"
kubectl get nodes -l pool=benchmark -o json | jq 'del(.metadata.managedFields)' | tee "$output_dir/nodes-benchmark.json"
## Optional wait period
sleep 120

########################################
## Step 2A. Application (Deploy)
########################################
## Deploy the application
$MAKE eks@use
$MAKE eks@deploy
## Wait for bidders to start
target_replicas=$(kubectl get deployments.apps bidder -o json | jq '.spec.replicas // 0')
available_replicas=$(kubectl get deployments.apps bidder -o json | jq '.status.availableReplicas // 0')
updated_replicas=$(kubectl get deployments.apps bidder -o json | jq '.status.updatedReplicas // 0')
while [[ $available_replicas -lt $target_replicas || $updated_replicas -lt $target_replicas ]]; do
    sleep 30
    available_replicas=$(kubectl get deployments.apps bidder -o json | jq '.status.availableReplicas // 0')
    updated_replicas=$(kubectl get deployments.apps bidder -o json | jq '.status.updatedReplicas // 0')
done
## Inspect the configurations
kubectl get services bidder -o json | jq 'del(.metadata.managedFields)' | tee "$output_dir/services-bidder.json"
kubectl get deployments.apps bidder -o json | jq 'del(.metadata.managedFields)' | tee "$output_dir/deployments-apps-bidder.json"
kubectl get configmaps bidder-config -o json | jq .data | tee "$output_dir/configmaps-bidder-config-data.json"
## Optional wait period
sleep 120

#########################################
### Step 2B. Application (Reload)
#########################################
### Get the current replica count
#replicas=$(kubectl get deployments.apps bidder -o json | jq '.spec.replicas // 0')
### Scale in the replicas
#kubectl scale deployments.apps bidder --replicas=0
### Wait for replicas to scale in
#available_replicas=$(kubectl get deployments.apps bidder -o json | jq '.status.availableReplicas // 0')
#while [[ $available_replicas -gt 0 ]]; do
#    sleep 30
#    available_replicas=$(kubectl get deployments.apps bidder -o json | jq '.status.availableReplicas // 0')
#done
### Scale out the replicas
#kubectl scale deployments.apps bidder --replicas="$replicas"
### Wait for replicase to scale out
#target_replicas=$(kubectl get deployments.apps bidder -o json | jq '.spec.replicas // 0')
#available_replicas=$(kubectl get deployments.apps bidder -o json | jq '.status.availableReplicas // 0')
#updated_replicas=$(kubectl get deployments.apps bidder -o json | jq '.status.updatedReplicas // 0')
#while [[ $available_replicas -lt $target_replicas || $updated_replicas -lt $target_replicas ]]; do
#    sleep 30
#    available_replicas=$(kubectl get deployments.apps bidder -o json | jq '.status.availableReplicas // 0')
#    updated_replicas=$(kubectl get deployments.apps bidder -o json | jq '.status.updatedReplicas // 0')
#done
### Inspect the configurations
#kubectl get services bidder -o json | jq 'del(.metadata.managedFields)' | tee "$output_dir/services-bidder.json"
#kubectl get deployments.apps bidder -o json | jq 'del(.metadata.managedFields)' | tee "$output_dir/deployments-apps-bidder.json"
#kubectl get configmaps bidder-config -o json | jq .data | tee "$output_dir/configmaps-bidder-config-data.json"
### Optional wait period
#sleep 120

########################################
## Step 3. Warmup
########################################
## Delete jobs.batch resources
kubectl delete jobs.batch load-generator || true
kubectl delete jobs.batch report-aggregator || true
## Run test
VALUES="$_dir/warmup.values.yaml" $MAKE benchmark@run-simple
## Inspect the configuration
kubectl get jobs.batch load-generator -o json | jq 'del(.metadata.managedFields)' | tee "$output_dir/jobs-batch-load-generator-warmup.json"
## Wait for benchmark to complete
$MAKE benchmark@wait
## Collect data from load generators
LOAD_GENERATOR_NODE_SELECTOR_POOL=benchmark REPORT_FILE="$output_dir/aggregated-report-load-generator-warmup.md" $MAKE benchmark@report
## Optional wait period
sleep 120

########################################
## Step 4. Test
########################################
## Delete jobs.batch resources
kubectl delete jobs.batch load-generator || true
kubectl delete jobs.batch report-aggregator || true
## Run test
VALUES="$_dir/ramp.values.yaml" $MAKE benchmark@run-simple
## Inspect the configuration
kubectl get jobs.batch load-generator -o json | jq 'del(.metadata.managedFields)' | tee "$output_dir/jobs-batch-load-generator.json"
## Wait for benchmark to complete
$MAKE benchmark@wait
## Collect data from load generators
LOAD_GENERATOR_NODE_SELECTOR_POOL=benchmark REPORT_FILE="$output_dir/aggregated-report-load-generator.md" $MAKE benchmark@report

########################################
## Step 5. Collect application logs
########################################
tmp_file=$(mktemp)
for pod in $(kubectl get pods -l app=bidder -o name); do
    kubectl logs "$pod" | sed -E 's/[0-9]+/#/g' | sort | uniq >> "$tmp_file"
done
sort "$tmp_file" | uniq > "$output_dir/application-unique-logs.txt"
