#!/bin/sh

# Run pytest with the given arguments, then expose over HTTP the test files, healthz (empty file) and a pytest-status
# file with pytest exit status.

set -e

exit_status=0
poetry run pytest "$@" || exit_status=$?

echo "$exit_status" > pytest-status

touch healthz
python -m http.server
