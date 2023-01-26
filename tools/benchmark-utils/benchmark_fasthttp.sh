#!/usr/bin/env bash

# Run this script from the repository root directory
# It requires access to the Makefile targets included by the the Makefile in the repository root directory

target_dir=$(dirname "${BASH_SOURCE[0]}")
source "$target_dir/functions.sh"

VARIANT=BenchmarkHTTP make stack@deploy
make eks@deploy OVERLAY=benchmark
patch_bidder_version fasthttp
kubectl delete configmap bidder-config
# make sure that branch is correct
kubectl create configmap bidder-config --from-env-file=apps/bidder/env/production.env
reload_bidders 32
sleep 60
run_bench 10m 5000 60
collect_stuff bench_bidder_fasthttp

# latest for 2021-01-19 12:00
patch_bidder_version f00a6f5f6535e581bd555fbeaf5dbbef561d33f6
kubectl delete configmap bidder-config
# make sure that branch is correct
kubectl create configmap bidder-config --from-env-file=apps/bidder/env/production.env
reload_bidders 32
sleep 60
run_bench 10m 5000 60
collect_stuff bench_bidder_net_http
