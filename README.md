# gameco-technical-assessment

A generic Go backend boilerplate for building SaaS-style APIs with:

- Gin
- GORM
- Goose
- JWT access token + PostgreSQL-backed authentication sessions
- Redis-backed rate limiting and background-job support
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
- Redis for rate limiting, cache/support operations, and background jobs
- Goose SQL migrations
- Structured logging with `slog`
- Swagger docs route
- Dockerfiles for API, worker, migrate, and dev runtime
- Docker Compose for infra and live-reload development

### Product foundation
- User registration and login
- JWT access token
- Refresh token rotation with database-backed session revocation
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
- worker-ready async job foundation backed by Redis queue

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

### Health
- `GET /api/v1/health`

The `internal/modules/example` package remains as reference scaffolding, but it is not registered in the application router.

## Project structure

```text
cmd/                         # executable entrypoints
  api/                       # HTTP API server and Swagger metadata
  migrate/                   # Goose migration runner
  seed/                      # modular database seeder runner
  worker/                    # background worker process
internal/
  app/                       # bootstrap, router, dependency wiring
  config/                    # environment/config loading and validation
  constants/                 # shared role and permission catalogs
    permission/
    role/
  modules/                   # feature modules and route ownership
    auth/
    health/
    example/                 # reference scaffolding; not registered in routes
  platform/                  # infrastructure integrations
    auth/                    # JWT, password hashing, refresh tokens
    cache/                   # Redis client
    database/                # PostgreSQL/GORM and Goose helpers
    httpx/                   # shared HTTP response helpers
    identifier/              # UUIDv7 public identifier generation
    logger/                  # structured logging
    middleware/              # auth, rate limiting, request ID, recovery
  seeds/                     # modular, idempotent seeders
  workers/                   # background jobs and queue processing
migrations/                  # Goose SQL migrations
tests/
  e2e/                       # cross-module HTTP end-to-end tests
deployments/
  compose/                   # production/development Compose files
  docker/                    # API, worker, migrate, and dev Dockerfiles
docs/
  swagger/                   # generated OpenAPI artifacts
  schema.dbml                # database schema documentation
  *.md                       # human-facing project documentation
scripts/                     # repository helper scripts
```

---

## Documentation map

- [Getting started](docs/getting-started.md) — first successful local boot for beginners
- [Windows guide](docs/windows-guide.md) — recommended Windows + WSL2 setup path
- [Docker development guide](docs/docker-development-guide.md) — day-to-day Docker-first development reference
- [Editor setup](docs/editor-setup.md) — VS Code / IDE setup for format-on-save and linting
- [Pre-commit guide](docs/pre-commit-guide.md) — how to install and use the Git pre-commit hook
- [Module template](docs/module-template.md) — guide for adding a new module/feature properly
- [Migrations guide](docs/migrations-guide.md) — how to create, write, run, and review migrations
- [Testing guide](docs/testing-guide.md) — how to run and add tests in this repo
- [Identifier conventions](docs/identifier-conventions.md) — internal IDs, public UUIDv7 identifiers, JWT JTI, and refresh-token storage
- [Database schema](docs/schema.dbml) — current auth schema and planned marketplace domain model
- [Threat model](docs/THREAT-MODEL.md) — OWASP API risks, attack scenarios, mitigations, and residual risks
- [Architecture decisions](docs/ADR/README.md) — accepted session, data model, stock concurrency, rate-limit, and CORS decisions
- [Technical assessment](docs/technical-assessment.md) — original marketplace technical-assessment requirements

---

## Notes

- The Go module path is intentionally local-first: `gameco-technical-assessment`.
- Change the module path before publishing the repository.
- `internal/modules/example` is a reference module you can copy when creating a new feature.
