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

start_test_dependencies() {
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
