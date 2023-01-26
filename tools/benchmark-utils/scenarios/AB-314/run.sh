#!/usr/bin/env bash

set -ex

dir=$(dirname "${BASH_SOURCE[0]}")
app_dir="$dir/../../"
root_dir="$app_dir/../../"

source "$app_dir/functions.sh"

# Vars
stack_name="mk"
benchmark_datetime=$(date -Idate)
image_version="AB-314"

enable_profiler=0
#rate_cases=(20000 40000 60000 80000 100000 120000)
rate_cases=(120000)
duration="5m"

function call_make() {
    make -C "$root_dir" $@
}

function make_bidder() {
    if (($# < 3)); then
        echo "make_load_generator: missing arguments"
        echo "  usage: make_bidder stack_name variant_name config_name"
        exit 1
    fi

    local stack_name=${1}
    local variant_name=${2}
    local config_name=${3}

    local config_overrides_file="config.patch.tpl.env"
    shift 3

    local config_file=$(mktemp)

    template_with_envs "${dir}/${config_overrides_file}" "${config_file}" stack_name="${stack_name}" "${@}"
    call_make eks@deploy \
        VERSION="${image_version}" \
        STACK_NAME="${stack_name}" \
        VARIANT="${variant_name}" \
        BIDDER_CONFIG_OVERRIDES="${config_file}"

    kubectl patch deployment bidder -p "$(cat "$dir/${config_name}.deployment.patch.yaml")"
    reload_bidders
}

function make_load_generator() {
    if (($# < 2)); then
        echo "make_load_generator: missing arguments"
        echo "  usage: make_load_generator rate duration profiler_output"
        exit 1
    fi

    local rate=${1}
    local duration=${2}
    local profiler_output=${3}
    shift 3

    local target="https://${stack_name}.us-east-1.ab.clearcode.cc/bidrequest"
    local jobs=60
    local rate_per_job=$((rate / jobs))

    call_make benchmark@run \
        LOAD_GENERATOR_IMAGE_VERSION="${image_version}" \
        VARIANT=Benchmark \
        DURATION="${duration}" \
        NUMBER_OF_JOBS="${jobs}" \
        RATE_PER_JOB="${rate_per_job}" \
        ENABLE_PROFILER="${profiler_output}" \
        PROFILER_OUTPUT="${profiler_output}" \
        NUMBER_OF_DEVICES="1000000000" \
        TARGET="${target}"

    start_time=0
    while (($(kubectl get pods -l app=load-generator -o name --field-selector "status.phase!=Running" | wc -l) > 0)); do
        sleep 1
        start_time=$((start_time + 1))
    done

    echo ":: Load generators started in ${start_time} seconds"
}

function clean_load_generator() {
    kubectl delete job load-generator || true
    kubectl delete job report-aggregator || true
    kubectl wait --for=delete job load-generator || true
    kubectl wait --for=delete job report-aggregator || true
}

function get_stack_variant() {
    aws cloudformation describe-stacks \
        --stack-name "${stack_name}" \
        --output=text --query="Stacks[0].Parameters[?ParameterKey=='Variant'].ParameterValue" 2>/dev/null || echo ""
}

function make_stack() {
    if (($# < 1)); then
        echo "make_stack: missing arguments"
        echo "  usage: make_stack variant_name"
        exit 1
    fi

    local variant_name=${1}
    local current_variant=$(get_stack_variant)

    if [[ "${current_variant}" == "${variant_name}" ]]; then
        call_make eks@use STACK_NAME="${stack_name}"
    else
        call_make stack@deploy STACK_NAME="${stack_name}" VARIANT="${variant_name}"
        call_make eks@grant-access STACK_NAME="${stack_name}"
        call_make eks@provision || true
        call_make eks@deploy STACK_NAME="${stack_name}" VARIANT="${variant_name}"
    fi
}

function run_benchmark() {
    if (($# < 3)); then
        echo "run_benchmark: missing arguments"
        echo "  usage: run_benchmark variant_name scenario_name config_name"
        exit 1
    fi

    local variant_name=${1}
    local scenario_name=${2}
    local config_name=${3}

    local report_dir="${root_dir}/docs/benchmarks/results/${scenario_name}/"

    mkdir -p "${report_dir}"

    for rate in "${rate_cases[@]}"; do
        _log "running scenario ( variant=${variant_name} rate=${rate} config=${config_name})"

        case_report_dir="${report_dir}/rate-${rate}/"
        mkdir -p "${case_report_dir}"

        if [[ $enable_profiler -ne 0 ]]; then
            profiler_output="${scenario_name}/pprof-{{.Endpoint}}-{{.Hostname}}"
        fi

        make_bidder "$stack_name" "${variant_name}" "${config_name}"
        local start=$(date +%s000)

        clean_load_generator
        make_load_generator "$rate" "$duration" "${profiler_output}"
        wait_for_load_generators_to_complete 180

        collect_stuff "${case_report_dir}"
        local end=$(date +%s000)

        collect_grafana "${case_report_dir}/bidder.png" "${start}" "${end}"
        collect_grafana_node "${case_report_dir}/node.png" "${start}" "${end}"
    done
}

make_stack "Benchmark"
run_benchmark "Benchmark" "${benchmark_datetime}-default-cpu-policy-burstable" "burstable"
run_benchmark "Benchmark" "${benchmark_datetime}-default-cpu-policy-guaranteed" "guaranteed"

# cannot update CPU manager policy on existing node group
# need to remove node group for benchmark and recreate it with new policy
make_stack "Basic"

make_stack "BenchmarkStaticCPU"
run_benchmark "BenchmarkStaticCPU" "${benchmark_datetime}-static-cpu-policy-guaranteed" "guaranteed"
