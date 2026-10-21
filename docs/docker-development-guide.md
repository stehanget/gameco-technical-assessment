# Docker Development Guide

This guide explains the Docker-first workflow for this boilerplate after initial setup.

If you are new to the repo and just want the first successful boot, read:
- `docs/getting-started.md`

If your team uses mixed operating systems, this is the safest default path.

---

## 1. Why Docker-first?

Docker-first development helps because:
- everyone runs the same runtime shape
- fewer host OS differences leak into the project
- infra setup is simpler
- Windows users are less likely to suffer from shell/path mismatch

In short:
- local Go workflow is still supported
- Docker-first is the recommended cross-platform workflow

---

## 2. What Docker currently runs

Using Compose, this repo can run:
- `postgres`
- `redis`
- `api`
- `worker`
- `migrate` as a one-shot job

In the dev overlay, `api` and `worker` run with **Air** for live reload.

The worker is also where Redis-backed background jobs are processed.

---

## 3. Main Docker commands

### Start only infra for a host-running API/worker

```bash
make docker-infra-up
```

Starts:
- postgres
- redis

### Start the complete base stack

```bash
make docker-up
```

Starts the base Compose services, including the migration service, API, and worker using their configured images.

### Start full dev stack

```bash
make docker-dev-up
```

Starts:
- postgres
- redis
- api
- worker

### Stop dev stack

```bash
make docker-dev-down
```

### Reset dev stack and volumes

```bash
make docker-dev-reset
```

Use this if you want a clean local database/Redis volume state.

### Show running containers

```bash
make docker-dev-ps
```

Container names are prefixed by the Compose project name, which defaults to `APP_NAME` from `.env` or `.env.example`.
So with `APP_NAME=gameco-technical-assessment`, you should see names like `gameco-technical-assessment-api-1` instead of a generic `compose-api-1`.

### Build containers

```bash
make docker-build
```

### Run migrations

```bash
make docker-migrate-up
```

### Seed production bootstrap data

```bash
make docker-seed
```

This target runs the base image's production-safe seed mode after migrations. It seeds authorization reference data and catalog categories only; it does not create the demo user or demo products.

### Seed optional demo data in development

```bash
make docker-dev-seed
```

Use this when you want the predictable `user@mail.com` bootstrap user with buyer and seller roles.

### Roll back one migration

```bash
make docker-migrate-down
```

### Generate Swagger in Docker

```bash
make docker-swagger
```

### Run fast Go tests in Docker

```bash
make docker-test
```

### Run PostgreSQL E2E, integration, and runtime tests fully in Docker

These targets start a disposable PostgreSQL and Redis test stack, run migrations and seed data in a Go test container, then clean up the test stack automatically. They require `.env.test` but do not require Go installed on the host.

```bash
make docker-test-e2e
make docker-test-integration
make docker-test-runtime
```

### Run formatting in Docker

```bash
make docker-fmt
```

### Tail logs

```bash
make docker-dev-logs
```

---

## 4. Live reload behavior

The dev overlay uses Air.

That means:
- when Go files change
- the dev container rebuilds the binary
- the process restarts automatically

This currently covers:
- API
- worker

It is a practical dev workflow, not a full platform abstraction.

---

## 5. Docker-first vs local-Go workflow

### Docker-first
Good when:
- team uses mixed OSes
- you want fewer setup differences
- you want onboarding simpler

### Local Go workflow
Good when:
- you want fastest inner loop
- your host setup is already stable
- you are okay with some OS-specific tooling differences

This boilerplate supports both, but Docker-first is the safer shared recommendation.

---

## 6. Suggested daily loop

Typical day-to-day flow:

```bash
make docker-dev-up
make docker-migrate-up   # only when schema changes
make docker-dev-logs         # when you want to watch runtime output
make docker-test         # before opening PR / merging
```

Optional shared quality checks:

```bash
make docker-fmt
make docker-lint
make test-e2e
```

---

## 7. Manual Docker commands if you do not use Make

### Start full dev stack

```bash
docker compose -f deployments/compose/docker-compose.yml -f deployments/compose/docker-compose.dev.yml up --build -d postgres redis api worker
```

### Run migrations

```bash
docker compose -f deployments/compose/docker-compose.yml run --rm migrate up
```

### Stop stack

```bash
docker compose -f deployments/compose/docker-compose.yml -f deployments/compose/docker-compose.dev.yml down
```

### Reset stack + volumes

```bash
docker compose -f deployments/compose/docker-compose.yml -f deployments/compose/docker-compose.dev.yml down -v
```

### Generate Swagger inside a temporary Go container

```bash
docker run --rm -v "$PWD:/app" -w /app golang:1.25 sh -lc 'go run github.com/swaggo/swag/cmd/swag@latest init -g cmd/api/main.go -o docs/swagger'
```

---

## 8. Test notes

The repo also supports host-side tests, but for mixed-OS teams this is a nice shared fallback:

```bash
make docker-test
make test-e2e
```

---

## 9. Good team recommendation

If you want one simple policy for a mixed-OS team, use this:
- macOS/Linux: Docker-first preferred
- Windows: WSL2 + Docker-first strongly recommended

That keeps behavior much more predictable.
