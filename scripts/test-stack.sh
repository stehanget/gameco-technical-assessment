#!/usr/bin/env bash

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd -- "${SCRIPT_DIR}/.." && pwd)"
TEST_COMPOSE_FILE="${REPO_ROOT}/deployments/compose/docker-compose.test.yml"

load_test_env() {
  local requested_env_file="${TEST_ENV_FILE:-.env.test}"
  if [[ "${requested_env_file}" = /* ]]; then
    TEST_ENV_FILE="${requested_env_file}"
  else
    TEST_ENV_FILE="${REPO_ROOT}/${requested_env_file}"
  fi

  if [[ ! -f "${TEST_ENV_FILE}" ]]; then
    echo "missing ${TEST_ENV_FILE}; copy .env.test.example to .env.test" >&2
    return 1
  fi

  set -a
  # shellcheck disable=SC1090
  . "${TEST_ENV_FILE}"
  set +a

  : "${APP_PAYMENT_WEBHOOK_SECRET:?APP_PAYMENT_WEBHOOK_SECRET must be set in ${TEST_ENV_FILE}}"
  : "${APP_PAYMENT_WEBHOOK_EVENT_RETENTION:?APP_PAYMENT_WEBHOOK_EVENT_RETENTION must be set in ${TEST_ENV_FILE}}"
  : "${APP_PAYMENT_WEBHOOK_CLEANUP_INTERVAL:?APP_PAYMENT_WEBHOOK_CLEANUP_INTERVAL must be set in ${TEST_ENV_FILE}}"
  : "${APP_CHECKOUT_IDEMPOTENCY_RETENTION:?APP_CHECKOUT_IDEMPOTENCY_RETENTION must be set in ${TEST_ENV_FILE}}"
  : "${APP_CHECKOUT_IDEMPOTENCY_CLEANUP_INTERVAL:?APP_CHECKOUT_IDEMPOTENCY_CLEANUP_INTERVAL must be set in ${TEST_ENV_FILE}}"

  : "${POSTGRES_USER:?POSTGRES_USER must be set in ${TEST_ENV_FILE}}"
  : "${POSTGRES_PASSWORD:?POSTGRES_PASSWORD must be set in ${TEST_ENV_FILE}}"
  : "${POSTGRES_DB:?POSTGRES_DB must be set in ${TEST_ENV_FILE}}"
  : "${POSTGRES_PORT:?POSTGRES_PORT must be set in ${TEST_ENV_FILE}}"
  : "${REDIS_ADDR:?REDIS_ADDR must be set in ${TEST_ENV_FILE}}"

  REDIS_HOST_PORT="${REDIS_ADDR##*:}"
  : "${REDIS_HOST_PORT:?REDIS_ADDR must include a port in ${TEST_ENV_FILE}}"
  export REDIS_HOST_PORT
}

test_compose() {
  docker compose --env-file "${TEST_ENV_FILE}" -f "${TEST_COMPOSE_FILE}" "$@"
}

port_is_in_use() {
  local port="$1"
  if command -v lsof >/dev/null 2>&1; then
    lsof -nP -iTCP:"${port}" -sTCP:LISTEN >/dev/null 2>&1
    return $?
  fi
  if command -v ss >/dev/null 2>&1; then
    ss -ltn "sport = :${port}" | tail -n +2 | grep -q .
    return $?
  fi
  if command -v nc >/dev/null 2>&1; then
    nc -z 127.0.0.1 "${port}" >/dev/null 2>&1
    return $?
  fi
  return 1
}

assert_test_ports_available() {
  local postgres_port="$1"
  local redis_port="$2"
  if port_is_in_use "${postgres_port}"; then
    echo "test PostgreSQL port ${postgres_port} is already in use" >&2
    return 1
  fi
  if port_is_in_use "${redis_port}"; then
    echo "test Redis port ${redis_port} is already in use" >&2
    return 1
  fi
}

start_test_dependencies() {
  stop_test_dependencies
  assert_test_ports_available "${POSTGRES_PORT}" "${REDIS_HOST_PORT}"
  test_compose up -d
  until test_compose exec -T postgres pg_isready -U "${POSTGRES_USER}" -d "${POSTGRES_DB}" >/dev/null 2>&1; do
    sleep 1
  done
  until test_compose exec -T redis redis-cli ping >/dev/null 2>&1; do
    sleep 1
  done
}

stop_test_dependencies() {
  test_compose down --timeout 10 -v >/dev/null 2>&1 || true
}

prepare_test_database() {
  (
    cd "${REPO_ROOT}"
    go run ./cmd/migrate up
    go run ./cmd/seed
  )
}
