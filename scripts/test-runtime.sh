#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=./test-stack.sh
. "${SCRIPT_DIR}/test-stack.sh"

load_test_env
api_port="${APP_PORT:?APP_PORT must be set in ${TEST_ENV_FILE}}"
tmp_root="${TMPDIR:-/tmp}"
runtime_tmp_dir="$(mktemp -d "${tmp_root%/}/gameco-runtime.XXXXXX")"
log_file="${runtime_tmp_dir}/api.log"
api_binary="${runtime_tmp_dir}/gameco-api"
api_pid=""

cleanup() {
  if [[ -n "${api_pid}" ]]; then
    kill "${api_pid}" 2>/dev/null || true
    wait "${api_pid}" 2>/dev/null || true
  fi
  stop_test_dependencies
  rm -rf "${runtime_tmp_dir}"
}
trap cleanup EXIT

cd "${REPO_ROOT}"

if curl -fsS "http://localhost:${api_port}/health" >/dev/null 2>&1; then
  echo "runtime API port ${api_port} is already in use" >&2
  exit 1
fi

start_test_dependencies
prepare_test_database

go build -o "${api_binary}" ./cmd/api

APP_PORT="${api_port}" \
"${api_binary}" >"${log_file}" 2>&1 &
api_pid=$!

for _ in $(seq 1 60); do
  if curl -fsS "http://localhost:${api_port}/health" >/dev/null 2>&1; then
    break
  fi
  if ! kill -0 "${api_pid}" 2>/dev/null; then
    cat "${log_file}" >&2 || true
    exit 1
  fi
  sleep 1
done

curl -fsS "http://localhost:${api_port}/health" >/dev/null
RUNTIME_BASE_URL="http://localhost:${api_port}" go test -v -tags=runtime ./tests/runtime -count=1

echo "runtime tests passed"
