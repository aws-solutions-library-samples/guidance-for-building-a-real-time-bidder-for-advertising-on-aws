#!/usr/bin/env bash

set -e

_dir=$(dirname "${BASH_SOURCE[0]}")
_app_dir="$_dir/../../"
_root_dir="$_app_dir/../../"

source "$_app_dir/functions.sh"

function make_bidder() {
    if (($# < 1)); then
        echo "make_load_generator: missing arguments"
        echo "  usage: make_bidder stack_name"
        exit 1
    fi

    stack_name=${1}
    shift

    template_with_envs "$_dir/bidder.template.env" "$_dir/bidder.env" stack_name="${stack_name}" "${@}"

    kubectl delete configmap bidder-config || true
    kubectl create configmap bidder-config --from-env-file="$_dir/bidder.env"

    template_with_envs "$_dir/bidder-service.template.yaml" "$_dir/bidder-service.yaml" stack_name="${stack_name}" "${@}"
    kubectl apply -f "$_dir/bidder-service.yaml"

    kubectl patch deployment bidder -p "$(cat "$_dir/bidder-deployment-patch.yaml")"

    reload_bidders
}

function make_load_generator() {
    if (($# < 3)); then
        echo "make_load_generator: missing arguments"
        echo "  usage: make_load_generator stack_name rate duration"
        exit 1
    fi

    local stack_name=${1}
    local rate=${2}
    local duration=${3}
    shift 3

    local target="https://${stack_name}.us-east-1.ab.clearcode.cc/bidrequest"
    local rate_per_instance=$((rate / 60))

    template_with_envs "$_dir/load-generator-job.template.yaml" "$_dir/load-generator-job.yaml" \
        target="${target}" rate="$rate_per_instance" duration="$duration" "${@}"

    kubectl apply -f "$_root_dir/deployment/infrastructure/deployment/load-generator/load-generator-sa.yaml"
    kubectl apply -f "$_dir/load-generator-job.yaml"

    start_time=0
    while (($(kubectl get pods -l app=load-generator -o name --field-selector "status.phase!=Running" | wc -l) > 0)); do
        sleep 1
        start_time=$((start_time + 1))
    done
    echo ":: Load generators started in ${start_time} seconds"
}

function clean_load_generator() {
    kubectl delete job load-generator || true

    while (($(kubectl get pods -l app=load-generator -o name | wc -l) > 0)); do
        sleep 1
    done
}

function main() {
    # Consts
    stack_variant="Benchmark"
    benchmark_datetime=$(date -Iseconds | sed "s/://g")

    # Params
    stack_name="AB-293"
    benchmark_name="AB-293"
    create_stack=1
    enable_profiler=0
    experiment_compare=1
    experiment_estimate_max_throughput=1

    # Vars
    duration="5m"

    if [[ $create_stack -ne 0 ]]; then
        # Create cluster
        make -C "$_root_dir" stack@deploy STACK_NAME="${stack_name}" VARIANT="${stack_variant}"

        # Configure access to the Kubernetes
        make -C "$_root_dir" eks@grant-access STACK_NAME="${stack_name}"
        make -C "$_root_dir" eks@use STACK_NAME="${stack_name}"

        # Provision with basic services (e.g. Prometheus)
        make -C "$_root_dir" eks@provision || true

        # Provision the base application version
        make -C "$_root_dir" eks@deploy
    else
        make -C "$_root_dir" eks@use STACK_NAME="${stack_name}"
    fi

    if [[ $experiment_compare -ne 0 ]]; then
        rate=40000
        dynamodb_max_idle_conns_per_host_cases=(2 100)

        for dynamodb_max_idle_conns_per_host in "${dynamodb_max_idle_conns_per_host_cases[@]}"; do
            _log "running scenario ( rate=${rate} dynamodb_max_idle_conns_per_host=${dynamodb_max_idle_conns_per_host} )"
            scenario_name="scenario_compare_max_idle_conns_rate_${rate}_dynamodb_max_idle_conns_per_host_${dynamodb_max_idle_conns_per_host}"
            if [[ $enable_profiler -ne 0 ]]; then
                profiler_output="${benchmark_datetime}-${benchmark_name}/${scenario_name}/pprof-{{.Endpoint}}-{{.Hostname}}"
            fi

            make_bidder "$stack_name" dynamodb_max_idle_conns_per_host="$dynamodb_max_idle_conns_per_host"

            clean_load_generator
            make_load_generator "$stack_name" "$rate" "$duration" "profiler_output=${profiler_output}"

            wait_for_load_generators_to_complete 180
            collect_stuff "var/benchmark/${benchmark_datetime}-${benchmark_name}/${scenario_name}/"
        done
    fi

    if [[ $experiment_estimate_max_throughput -ne 0 ]]; then
        dynamodb_max_idle_conns_per_host=100
        rate_cases=(20000 40000 60000 80000 100000 120000 140000 160000 180000)
        for rate in "${rate_cases[@]}"; do
            _log "running scenario ( rate=${rate} dynamodb_max_idle_conns_per_host=${dynamodb_max_idle_conns_per_host} )"
            scenario_name="scenario_estimate_max_throughput_rate_${rate}_dynamodb_max_idle_conns_per_host_${dynamodb_max_idle_conns_per_host}"
            if [[ $enable_profiler -ne 0 ]]; then
                profiler_output="${benchmark_datetime}-${benchmark_name}/${scenario_name}/pprof-{{.Endpoint}}-{{.Hostname}}"
            fi

            make_bidder "$stack_name" dynamodb_max_idle_conns_per_host="$dynamodb_max_idle_conns_per_host"

            clean_load_generator
            make_load_generator "$stack_name" "$rate" "$duration" "profiler_output=${profiler_output}"

            wait_for_load_generators_to_complete 180
            collect_stuff "var/benchmark/${benchmark_datetime}-${benchmark_name}/${scenario_name}/"
        done
    fi
}

main

set +e
