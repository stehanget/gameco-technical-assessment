# gameco-technical-assessment

A generic Go backend boilerplate for building SaaS-style APIs with:

- Gin
- GORM
- Goose
- JWT access token + Redis refresh session
- User roles with domain-specific permission capabilities
- Swagger / OpenAPI docs
- Docker-first local development
- Modular monolith structure

This repo is meant to be a **starter that people can actually understand**, not a fancy architecture demo.

---

## What you get out of the box

### Runtime and infrastructure
- HTTP API with Gin
- PostgreSQL for transactional data
- Redis for refresh sessions
- Goose SQL migrations
- Structured logging with `slog`
- Swagger docs route
- Dockerfiles for API, worker, migrate, and dev runtime
- Docker Compose for infra and live-reload development

### Product foundation
- User registration and login
- JWT access token
- Refresh token rotation
- Logout / refresh session revoke
- Seeded `user@mail.com` with buyer + seller roles
- Default-deny permission middleware
- Redis-backed background job starter with dedicated worker process

### Developer workflow
- Docker-first dev flow
- Air-based live reload for API and worker
- `gofumpt` + `goimports` formatting workflow
- `golangci-lint` lint workflow
- optional Git pre-commit hook
- worker-ready async example route backed by Redis queue

---

## Recommended development style

This repo now prefers a **Docker-first, cross-platform** workflow.

Best shared team default:
- Docker Desktop / Docker Engine
- Docker Compose
- WSL2 on Windows
- local Go only if you want faster host-side development

If you want the least platform friction, use Docker for:
- app runtime
- worker runtime
- infra
- migrations
- optional Swagger generation
- optional formatting, linting, and tests

---

## Quick start

### 1. Copy env file

```bash
cp .env.example .env
```

### 2. Start full dev stack with live reload

```bash
make docker-dev-up
```

### 3. Run migrations

```bash
make docker-migrate-up
```

### 4. Generate Swagger docs

If you have local Go installed:

```bash
make swagger
```

If you want Docker-only generation:

```bash
make docker-swagger
```

### 5. Open Swagger UI

```text
http://localhost:8080/swagger/index.html
```

---

## Common commands

### Docker-first workflow

```bash
make docker-dev-up
make docker-dev-down
make docker-dev-reset
make docker-ps
make docker-migrate-up
make docker-migrate-down
make docker-seed
make docker-swagger
make docker-fmt
make docker-lint
make docker-test
make docker-test-e2e
make docker-logs
```

### Local Go workflow

```bash
make run
make worker
make seed
make migrate-up
make migrate-down
make fmt
make fmt-check
make lint
make build
make test
make test-e2e
make swagger
make install-hooks
```

---

## Main API areas

### Auth
- `POST /api/v1/auth/register`
- `POST /api/v1/auth/login`
- `POST /api/v1/auth/refresh`
- `POST /api/v1/auth/logout`
- `GET /api/v1/auth/me`

### Example
- `GET /api/v1/examples`
- `POST /api/v1/examples`
- `POST /api/v1/examples/async`

## Project structure

```text
cmd/                    # executable entrypoints
  api/
  worker/
  migrate/
internal/
  app/                  # bootstrap, router, wiring
  config/               # env/config loading and validation
  modules/              # product modules
    auth/
    health/
    example/
  platform/             # infra integrations (db, auth, cache, middleware)
  workers/              # background job code
migrations/             # Goose SQL migrations
deployments/            # Docker and runtime assets
docs/                   # human-facing project documentation
```

---

## Documentation map

- `docs/getting-started.md` — first successful local boot for beginners
- `docs/windows-guide.md` — recommended Windows + WSL2 setup path
- `docs/docker-development-guide.md` — day-to-day Docker-first development reference
- `docs/editor-setup.md` — VS Code / IDE setup for format-on-save and linting
- `docs/pre-commit-guide.md` — how to install and use the Git pre-commit hook
- `docs/module-template.md` — single guide for adding a new module/feature properly
- `docs/migrations-guide.md` — how to create, write, run, and review migrations
- `docs/testing-guide.md` — how to run and add tests in this repo

---

## Notes

- The Go module path is intentionally local-first: `gameco-technical-assessment`.
- Change the module path before publishing the repository.
- `internal/modules/example` is a reference module you can copy when creating a new feature.
