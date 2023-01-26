#!/usr/bin/env bash

set -e -x

_dir=$(dirname "${BASH_SOURCE[0]}")
_app_dir="$_dir/../../"
_root_dir="$_app_dir/../../"

source "$_app_dir/functions.sh"

STACK_NAME=mmi
VARIANT=BenchmarkHTTP
OVERLAY=benchmark
VERSION=17bd1e5da6ae1e6c7a320d26f4784f00698f0863

function make_bidder() {
    VERSION=$VERSION make eks@deploy

    reload_bidders 32
}

function make_load_generator() {
    if (($# < 2)); then
        echo "make_load_generator: missing arguments"
        echo "  usage: make_load_generator rate duration"
        exit 1
    fi

    local rate=${1}
    local duration=${2}

    make benchmark@run DURATION=${duration} TIMEOUT=100ms NUMBER_OF_JOBS=60 RATE_PER_JOB=$((rate / 60)) NUMBER_OF_DEVICES=1000000000 ENABLE_PROFILER=0 NAME="${NAME}" TARGET="https://${STACK_NAME}.us-east-1.ab.clearcode.cc/bidrequest" VARIANT=$VARIANT NAME=$NAME

    start_time=0
    while (($(kubectl get pods -l app=load-generator -o name --field-selector "status.phase!=Running" | wc -l) > 0)); do
        sleep 1
        start_time=$((start_time + 1))
    done
    echo ":: Load generators started in ${start_time} seconds"
}

function clean_load_generator() {
    make benchmark@cleanup || true

    while (($(kubectl get pods -l app=load-generator -o name | wc -l) > 0)); do
        sleep 1
    done
}

function main() {
  # Vars
  benchmark_datetime=$(date -Iseconds)
  benchmark_name="dynamodb"
  # Set to a high enough value that load generator won't complete while we are waiting for it to start: otherwise we
  # hang.
  duration=5m
  configs=(centuria fqdn netdns)
  #rate_steps=(5000 10000 20000 30000 50000 100000)
  rate_steps=(180000)

  for config in "${configs[@]}"; do

    BIDDER_CONFIG_OVERRIDES="${_dir}/configs/$config" make_bidder

    for rate in "${rate_steps[@]}"; do
      echo ":: Running scenario (config=${config}, rate=${rate})"
      scenario_name="scenario_config_${config}_rate_${rate}"
      clean_load_generator
      local start=$(date +%s000)
      NAME=$scenario_name make_load_generator "$rate" "$duration"
      while (($(kubectl get pods -l app=load-generator -o name --field-selector "status.phase!=Succeeded" | wc -l) > 0)); do
        sleep 1
      done
      sleep 1
      local end=$(date +%s000)
      collect_stuff "var/benchmark/${benchmark_datetime}-${benchmark_name}/${scenario_name}/"
      collect_grafana "var/benchmark/${benchmark_datetime}-${benchmark_name}/${scenario_name}/bidder.png" $start $end
    done
  done
}

main

set +e
