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

For PostgreSQL/Redis integration and runtime tests, create the ignored test environment file:

```bash
cp .env.test.example .env.test
```

`.env.test` is local-only and must not be committed.

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
make docker-dev-ps
make docker-migrate-up
make docker-migrate-down
make docker-dev-seed
make docker-swagger
make docker-fmt
make docker-lint
make docker-test
make docker-test-e2e
make docker-test-integration
make docker-test-runtime
make docker-dev-logs
```

For the base/production Compose stack:

```bash
make docker-up
make docker-seed
make docker-ps
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
make test-integration
make test-runtime
make verify
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

### Catalog and seller products
- `GET /api/v1/products`
- `GET /api/v1/products/:product_public_id`
- `GET /api/v1/categories/options`
- `GET /api/v1/seller/products`
- `GET /api/v1/seller/products/:product_public_id`
- `POST /api/v1/seller/products`
- `PATCH /api/v1/seller/products/:product_public_id`

### Cart, checkout, and orders
- `GET /api/v1/cart`
- `PUT /api/v1/cart/items/:product_public_id`
- `DELETE /api/v1/cart/items/:product_public_id`
- `POST /api/v1/checkout`
- `GET /api/v1/orders`
- `GET /api/v1/orders/:order_public_id`
- `GET /api/v1/seller/orders`
- `GET /api/v1/seller/orders/:order_public_id`
- `POST /api/v1/webhooks/payment`

### Development demo tools
- `POST /api/v1/dev/signatures/api` — only when `APP_ENV=development`
- `POST /api/v1/dev/signatures/webhook` — only when `APP_ENV=development`

### Health
- `GET /health` — backward-compatible liveness alias
- `GET /livez` — process liveness only; does not probe dependencies
- `GET /readyz` — PostgreSQL and Redis readiness probe; returns `503` while dependencies are unavailable

### Security and audit
- Login for an unknown email performs dummy bcrypt verification and returns the same stable error contract as a known email with a wrong password.
- Failed login attempts, seller price changes, and payment-driven order status changes are written to append-only `audit_events` records without raw credentials or passwords.

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
    catalog/                 # catalog reads and seller product management
    cart/                    # buyer cart lifecycle
    checkout/                # checkout, order detail, and payment webhook
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
  e2e/                       # PostgreSQL + Redis HTTP tests
  integration/               # real PostgreSQL/Redis in-process HTTP tests
  runtime/                   # real API binary black-box HTTP tests
  support/                   # shared real-dependency test harness
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
- [Coding conventions](docs/CODING-CONVENTIONS.md) — cross-cutting rules for Go, API contracts, SQL, timestamps, security, migrations, and tests
- [Identifier conventions](docs/identifier-conventions.md) — internal IDs, public UUIDv7 identifiers, JWT JTI, and refresh-token storage
- [Database schema](docs/schema.dbml) — current PostgreSQL auth and marketplace schema
- [Threat model](docs/THREAT-MODEL.md) — OWASP API risks, attack scenarios, mitigations, and residual risks
- [Architecture decisions](docs/ADR/README.md) — accepted session, data model, stock concurrency, rate-limit, and CORS decisions
- [API signature guide](docs/api-signature-guide.md) — trusted server-to-server request signing, canonicalization, replay protection, and integration examples
- [Assessment handoff](docs/assessment-handoff.md) — delivered scope, verification evidence, and operational follow-up
- [Technical assessment](docs/technical-assessment.md) — original marketplace technical-assessment requirements

---

## Notes

- The Go module path is intentionally local-first: `gameco-technical-assessment`.
- Change the module path before publishing the repository.
- `internal/modules/example` is a reference module you can copy when creating a new feature.
