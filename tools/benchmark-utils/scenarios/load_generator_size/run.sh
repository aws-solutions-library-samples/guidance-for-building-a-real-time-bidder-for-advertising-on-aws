#!/usr/bin/env bash

set -e

_dir=$(dirname "${BASH_SOURCE[0]}")
_app_dir="$_dir/../../"
_root_dir="$_app_dir/../../"

source "$_app_dir/functions.sh"

function make_bidder() {
  kubectl delete configmap bidder-config || true
  kubectl create configmap bidder-config --from-env-file=bidder.env

  kubectl patch deployment bidder -p "$(cat bidder_deployment_patch.yaml)"

  reload_bidders 32
}

function make_load_generator() {
  if (($# < 4)); then
    echo "make_load_generator: missing arguments"
    echo "  usage: make_load_generator parallelism cpu workers_per_core rate_per_core [variable=value[ variableN=valueN]"
    exit 1
  fi

  parallelism=${1}
  cpu=${2}
  workers_per_core=${3}
  rate_per_core=${4}
  shift
  shift
  shift
  shift

  min_workers=$cpu
  max_workers=$((workers_per_core * cpu))
  rate=$((rate_per_core * cpu))

  template_with_envs "$_dir/load_generator_job.template.yaml" "$_dir/load_generator_job.yaml" \
    parallelism="$parallelism" cpu="$cpu" rate=$rate min_workers="$min_workers" max_workers=$max_workers "${@}"

  cat "$_dir/load_generator_job.yaml"
  kubectl apply -f "$_dir/load_generator_job.yaml"
}

function clean_load_generator() {
  kubectl delete job load-generator || true

  while (($(kubectl get pods -l app=load-generator -o name | wc -l) > 0)); do
    sleep 1
  done
}

function main() {
  # Start cluster
  # VARIANT=BenchmarkHTTP make -C "$_root_dir" stack@deploy

  parallelism_cases=(1 2 4 15 30 60)
  workers_per_core_cases=(4 8 16 32 64)
  rate_per_core_cases=(5000 10000 20000)

  for parallelism in "${parallelism_cases[@]}"; do
    cpu=$((60 / parallelism))
    for workers_per_core in "${workers_per_core_cases[@]}"; do
      for rate_per_core in "${rate_per_core_cases[@]}"; do
        echo ":: Running scenario (parallelism=$parallelism cpu=$cpu workers_per_core=$workers_per_core rate_per_core=$rate_per_core)"
        clean_load_generator
        make_bidder
        make_load_generator "$parallelism" "$cpu" "$workers_per_core" "$rate_per_core" "duration=3m"
        while (($(kubectl get pods -l app=load-generator -o name --field-selector "status.phase!=Succeeded" | wc -l) > 0)); do
          sleep 1
        done
        collect_stuff "var/benchmark/parallelism_${parallelism}_cpu_${cpu}_workers_per_core_${workers_per_core}_rate_per_core_${rate_per_core}/"
        sleep 30
      done
    done
  done
}

main

set +e
