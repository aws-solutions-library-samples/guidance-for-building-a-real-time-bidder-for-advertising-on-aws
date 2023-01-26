#!/usr/bin/env bash

# Run this script from the repository root directory
# It requires access to the Makefile targets included by the the Makefile in the repository root directory

target_dir=$(dirname "${BASH_SOURCE[0]}")
source "$target_dir/functions.sh"

VARIANT=BenchmarkHTTP make stack@deploy
make eks@deploy OVERLAY=benchmark
disable_kinesis
reload_bidders 32
run_bench 10m 5000 60
collect_stuff bench_bidder_kinesis_disabled

VARIANT=BenchmarkKinesis make stack@deploy
make eks@deploy OVERLAY=benchmark
enable_kinesis
reload_bidders 32
run_bench 10m 5000 60
collect_stuff bench_bidder_kinesis_enabled

make eks@deploy
make stack@deploy
