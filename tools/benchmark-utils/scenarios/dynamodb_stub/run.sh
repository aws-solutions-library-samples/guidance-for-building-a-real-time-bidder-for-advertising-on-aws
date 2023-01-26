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

    template_with_envs "$_dir/bidder-service.template.yaml" "$_dir/bidder-service.yaml" stack_name="${stack_name}"
    kubectl apply -f "$_dir/bidder-service.yaml"

    template_with_envs "$_dir/bidder-deployment-patch.template.yaml" "$_dir/bidder-deployment-patch.yaml" "${@}"
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
    stack_name="dynamodb-stub"
    benchmark_name="dynamodb-stub"
    create_stack=1
    run_benchmark=1
    enable_profiler=0

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

    if [[ $run_benchmark -ne 0 ]]; then

        rate_steps=(1000 4000 8000 12000 16000 20000 24000 28000 32000)
        replicas=32

        for rate in "${rate_steps[@]}"; do
            scenario_name="scenario_test_fixed_replicas_rate_${rate}_replicase_${replicas}"
            if [[ $enable_profiler -ne 0 ]]; then
                profiler_output="${benchmark_datetime}-${benchmark_name}/${scenario_name}/pprof-{{.Endpoint}}-{{.Hostname}}"
            fi

            _log "running scenario ( test=fixed-replicas rate=$rate replicas=$replicas )"

            make_bidder "$stack_name" replicas="$replicas"

            clean_load_generator
            make_load_generator "$stack_name" "$rate" "$duration" "profiler_output=${profiler_output}"

            while (($(kubectl get pods -l app=load-generator -o name --field-selector "status.phase!=Succeeded" | wc -l) > 0)); do
                sleep 1
            done

            collect_stuff "var/benchmark/${benchmark_datetime}-${benchmark_name}/${scenario_name}/"
        done

        replicas_values=(1 4 8 16 20 24 28 32)
        rate=32000

        for replicas in "${replicas_values[@]}"; do
            _log "running scenario ( test=fixed-rate rate=$rate replicas=$replicas )"

            scenario_name="scenario_test_fixed_rate_rate_${rate}_replicase_${replicas}"
            if [[ $enable_profiler -ne 0 ]]; then
                profiler_output="${benchmark_datetime}-${benchmark_name}/${scenario_name}/pprof-{{.Endpoint}}-{{.Hostname}}"
            fi

            make_bidder "$stack_name" replicas="$replicas"

            clean_load_generator
            make_load_generator "$stack_name" "$rate" "$duration" "profiler_output=${profiler_output}"

            while (($(kubectl get pods -l app=load-generator -o name --field-selector "status.phase!=Succeeded" | wc -l) > 0)); do
                sleep 1
            done

            collect_stuff "var/benchmark/${benchmark_datetime}-${benchmark_name}/${scenario_name}/"
        done
    fi
}

main

set +e
