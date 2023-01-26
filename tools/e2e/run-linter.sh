#!/bin/sh

set -e

dir=tools/e2e
command=$1
shift 1

cd "$dir"
export dir
exec poetry run bash -c 'exec '"$command"' "${@#$dir/}"' -- "$@"
