#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=./test-stack.sh
. "${SCRIPT_DIR}/test-stack.sh"

load_test_env
api_port="${APP_PORT:?APP_PORT must be set in ${TEST_ENV_FILE}}"
: "${APP_API_SIGNING_KEYS:?APP_API_SIGNING_KEYS must be set in ${TEST_ENV_FILE}}"
tmp_root="${TMPDIR:-/tmp}"
runtime_tmp_dir="$(mktemp -d "${tmp_root%/}/gameco-runtime.XXXXXX")"
log_file="${runtime_tmp_dir}/api.log"
api_binary="${runtime_tmp_dir}/gameco-api"
api_pid=""

stop_api() {
  if [[ -z "${api_pid}" ]]; then
    return
  fi
  kill "${api_pid}" 2>/dev/null || true
  for _ in $(seq 1 50); do
    if ! kill -0 "${api_pid}" 2>/dev/null; then
      break
    fi
    sleep 0.1
  done
  if kill -0 "${api_pid}" 2>/dev/null; then
    kill -KILL "${api_pid}" 2>/dev/null || true
  fi
  wait "${api_pid}" 2>/dev/null || true
  api_pid=""
}

cleanup() {
  stop_api
  stop_test_dependencies
  rm -rf "${runtime_tmp_dir}"
}
trap cleanup EXIT

start_api() {
  local signature_enabled="$1"
  : >"${log_file}"
  APP_API_SIGNATURE_ENABLED="${signature_enabled}" \
  APP_PORT="${api_port}" \
  "${api_binary}" >"${log_file}" 2>&1 &
  api_pid=$!

  for _ in $(seq 1 60); do
    if curl -fsS "http://localhost:${api_port}/health" >/dev/null 2>&1; then
      return
    fi
    if ! kill -0 "${api_pid}" 2>/dev/null; then
      cat "${log_file}" >&2 || true
      exit 1
    fi
    sleep 1
  done
  echo "runtime API did not become ready" >&2
  cat "${log_file}" >&2 || true
  exit 1
}

cd "${REPO_ROOT}"

if curl -fsS "http://localhost:${api_port}/health" >/dev/null 2>&1; then
  echo "runtime API port ${api_port} is already in use" >&2
  exit 1
fi

start_test_dependencies
prepare_test_database
go build -o "${api_binary}" ./cmd/api

start_api true
APP_API_SIGNATURE_ENABLED=true RUNTIME_BASE_URL="http://localhost:${api_port}" \
  go test -v -tags=runtime ./tests/runtime -run '^TestAPISignatureRejectsTamperingAndReplay$' -count=1

stop_api
start_api false
APP_API_SIGNATURE_ENABLED=false RUNTIME_BASE_URL="http://localhost:${api_port}" \
  go test -v -tags=runtime ./tests/runtime -skip '^TestAPISignatureRejectsTamperingAndReplay$' -count=1

echo "runtime tests passed"
