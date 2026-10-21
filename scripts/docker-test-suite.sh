#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd -- "${SCRIPT_DIR}/.." && pwd)"
TEST_ENV_FILE="${TEST_ENV_FILE:-.env.test}"
if [[ "${TEST_ENV_FILE}" != /* ]]; then
  TEST_ENV_FILE="${REPO_ROOT}/${TEST_ENV_FILE}"
fi

if [[ ! -f "${TEST_ENV_FILE}" ]]; then
  echo "missing ${TEST_ENV_FILE}; copy .env.test.example to .env.test" >&2
  exit 1
fi

compose() {
  docker compose --env-file "${TEST_ENV_FILE}" -f "${REPO_ROOT}/deployments/compose/docker-compose.test.yml" "$@"
}

suite="${1:-}"
case "${suite}" in
  e2e)
    test_command='go run ./cmd/migrate up && go run ./cmd/seed && go test -v -tags=e2e ./tests/e2e -count=1'
    ;;
  integration)
    test_command='go run ./cmd/migrate up && go run ./cmd/seed && go test -v -tags=integration ./tests/integration -count=1'
    ;;
  runtime)
    test_command='set -e; go run ./cmd/migrate up; go run ./cmd/seed; go build -o /tmp/gameco-api ./cmd/api; APP_PORT="${APP_PORT}" /tmp/gameco-api >/tmp/gameco-api.log 2>&1 & api_pid=$!; trap '\''kill "${api_pid}" 2>/dev/null || true; wait "${api_pid}" 2>/dev/null || true'\'' EXIT; for _ in $(seq 1 60); do if bash -c "</dev/tcp/localhost/${APP_PORT}" >/dev/null 2>&1; then break; fi; if ! kill -0 "${api_pid}" 2>/dev/null; then cat /tmp/gameco-api.log >&2; exit 1; fi; sleep 1; done; bash -c "</dev/tcp/localhost/${APP_PORT}"; RUNTIME_BASE_URL="http://localhost:${APP_PORT}" go test -v -tags=runtime ./tests/runtime -count=1'
    ;;
  *)
    echo "usage: $0 {e2e|integration|runtime}" >&2
    exit 2
    ;;
esac

cleanup() {
  compose down --timeout 10 >/dev/null 2>&1 || true
}
trap cleanup EXIT

compose up -d --wait postgres redis
compose run --rm --no-deps test-runner sh -c "${test_command}"