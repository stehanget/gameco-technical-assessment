#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=./test-stack.sh
. "${SCRIPT_DIR}/test-stack.sh"

load_test_env
trap stop_test_dependencies EXIT
start_test_dependencies
prepare_test_database

cd "${REPO_ROOT}"
go test -v -tags=e2e ./tests/e2e -count=1
