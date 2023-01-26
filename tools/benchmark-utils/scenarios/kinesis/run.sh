#!/usr/bin/env bash

set -e

_dir=$(dirname "${BASH_SOURCE[0]}")
_app_dir="$_dir/../../"
_root_dir="$_app_dir/../../"

source "$_app_dir/functions.sh"

function make_bidder() {
  if (($# < 1)); then
    echo "make_load_generator: missing arguments"
    echo "  usage: make_bidder version"
    exit 1
  fi

  kubectl delete configmap bidder-config || true
  kubectl create configmap bidder-config --from-env-file=bidder.env

  local version=${1}
  shift

  template_with_envs "$_dir/bidder_deployment_patch.template.yaml" "$_dir/bidder_deployment_patch.yaml" version="$version"

  kubectl patch deployment bidder -p "$(cat "$_dir/bidder_deployment_patch.yaml")"

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
  shift
  shift

  local rate_per_instance=$((rate / 60))

  template_with_envs "$_dir/load_generator_job.template.yaml" "$_dir/load_generator_job.yaml" rate="$rate_per_instance" duration="$duration" "${@}"

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
  VARIANT=BenchmarkKinesis make -C "$_root_dir" stack@deploy
  datetime=$(date -Iseconds | sed "s/://g")
  duration="15m"
#  version_cases=("eb5881e821d137e62b1b9d339b38d7860add5ecb" "AB-194-kinesis-zstd-v1")
#  throughput_cases=(150000 175000 200000 225000 250000 275000 300000)
#
#  for throughput in "${throughput_cases[@]}"; do
#    for version in "${version_cases[@]}"; do
#      echo ":: Running scenario (throughput=${throughput} version=${version})"
#      clean_load_generator
#      make_bidder "${version}"
#      make_load_generator "$throughput" "$duration"
#      while (($(kubectl get pods -l app=load-generator -o name --field-selector "status.phase!=Succeeded" | wc -l) > 0)); do
#        sleep 1
#      done
#      collect_stuff "var/benchmark/${datetime}/throughput=${throughput} version=${version}/"
#      sleep 30
#      true
#    done
#  done

  profile_throughput_cases=(225000 250000)
  version="AB-194-kinesis-zstd-v1"
  for throughput in "${profile_throughput_cases[@]}"; do
    echo ":: Running profile scenario (throughput=${throughput} version=${version})"
    clean_load_generator
    make_bidder "${version}"
    make_load_generator "$throughput" "$duration" "profiler_output=${datetime}-${version}-${throughput}.pprof"
    while (($(kubectl get pods -l app=load-generator -o name --field-selector "status.phase!=Succeeded" | wc -l) > 0)); do
      sleep 1
    done
    collect_stuff "var/benchmark/${datetime}/profile/throughput=${throughput} version=${version}/"
    sleep 30
    true
  done
}

main

set +e

# http://localhost:8080/d/9BDpvv-Mz/bidder?orgId=1&from=1611653360962&to=1611660460331
