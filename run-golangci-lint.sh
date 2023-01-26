#!/bin/bash
set -xe

dir=$1
config=$2

cd "$dir"
exec golangci-lint run -c $config --allow-parallel-runners ./...
