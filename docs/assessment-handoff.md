# Assessment Handoff

## Repository status

- Branch: `main`
- Previous pushed commit: `a9e87076feb8d459d3f9e944d56f91f06779f4ab`
- Working tree: contains the current seller-order, anti-enumeration, audit, health, generated Swagger, and test changes; it is not clean until this slice is committed.
- No secrets, credentials, tokens, or connection strings are included in this handoff.

## Delivered scope

### Authentication and authorization

- PostgreSQL-backed authentication sessions and refresh-token rotation.
- Logout and session revocation.
- JWT identity bound to the authoritative database session owner.
- Actor and permission-aware authorization.
- Seller ownership predicates on catalog mutations and reads.
- Login failure lockout plus unknown-email dummy bcrypt verification to keep the stable invalid-credentials response and reduce timing-based enumeration.
- Append-only audit producers for failed login attempts, seller price changes, and payment-driven order status transitions. Audit writes are sanitized and use the domain transaction for product/payment mutations.
- Liveness/readiness routes: `/livez` is process-only, `/readyz` probes PostgreSQL and Redis, and `/health` remains a backward-compatible liveness alias.
- Redis-backed rate limiting: auth and payment webhooks use dedicated policies, while opted-in catalog, cart, and checkout/order routes share global anonymous/authenticated policies.
- CORS restricted to the configured localhost allowlist with credentials disabled.
- API response security headers: `nosniff`, frame denial, strict referrer policy, and disabled camera/microphone/geolocation permissions.

### Catalog, cart, and checkout

- Public catalog listing/detail and canonical category options.
- Seller product create, update, list, and ownership enforcement.
- Seller order listing at `GET /api/v1/seller/orders`, scoped by `order_items.seller_id`, with seller-only item snapshots, seller subtotal, payment/order status, and bounded page pagination.
- Buyer cart lifecycle with ownership and availability validation.
- Checkout from the authenticated active cart.
- `Idempotency-Key` replay and request-hash conflict handling.
- Atomic PostgreSQL stock reservation and deterministic product locking.
- Buyer order detail reads.

### Payment and maintenance

- Signed payment webhook verification with timestamp freshness checks.
- Provider event idempotency and payload conflict detection.
- Paid, failed, and expired payment transitions.
- Exactly-once stock restoration for expired payments.
- Retention cleanup for processed webhook events.
- Retention cleanup for completed checkout idempotency rows while pending rows are preserved.
- Production-safe seed mode containing reference authorization data and categories only.

### Database and runtime parity

- PostgreSQL is the source of truth for production, development, E2E, integration, and runtime tests.
- Catalog focused tests use PostgreSQL-dialect GORM with `go-sqlmock`.
- The former alternate database drivers and source usage have been removed.
- Checkout locking is unconditional; no database-dialect branch remains.
- Migration loading requires PostgreSQL configuration only and does not require application secrets.

### API request signing

- API request signing is a separate mechanism from payment webhook signing.
- Payment webhook headers and signing behavior remain unchanged.
- Trusted integration requests can opt into `middleware.RequireAPISignature(...)` at route/group scope; current auth, catalog, cart, and checkout route groups use it when enabled, while it is not installed globally.
- API signing uses `X-API-Key`, `X-API-Timestamp`, `X-API-Nonce`, and `X-API-Signature: v1=<hex>`.
- The canonical fields are method, escaped path plus query, timestamp, nonce, and SHA-256 of the raw body, joined with `|`.
- The canonical field order rotates according to `timestamp mod APP_API_SIGNATURE_ROTATION_COUNT`; the default is four rotations, and supported values are 1 through 5.
- Runtime tests automatically restart the API between the signature-enabled phase and the ordinary unsigned phase; `APP_API_SIGNATURE_ENABLED` is not required in `.env.test` or `.env.test.example`.
- Redis stores accepted key/nonce pairs for the signature window to prevent replay.
- API request signing is intended for trusted server-to-server integrations; public browser/mobile clients continue using JWT/session authorization.

## Verification evidence

The following verification was recorded before the current documentation/logger follow-up edits:

```bash
make check
go test ./... -count=1
make docker-test
make docker-test-e2e
make docker-test-integration
make docker-test-runtime
git diff --check
```

The catalog logger follow-up was subsequently verified with:

```bash
make test
git diff --check
```

Compose configuration validation passed for:

```bash
docker compose --env-file .env -f deployments/compose/docker-compose.yml config --quiet
docker compose --env-file .env -f deployments/compose/docker-compose.yml \
  -f deployments/compose/docker-compose.dev.yml config --quiet
docker compose --env-file .env.test \
  -f deployments/compose/docker-compose.test.yml config --quiet
```

The current slice has been additionally verified with:

```bash
make docker-test-e2e
make swagger
go test ./...
git diff --check
```

The Docker E2E suite passes seller-order isolation, unknown-email login anti-enumeration, failed-login audit persistence, price-change audit persistence, and payment order-status audit persistence. The liveness/readiness package tests pass in `go test ./...`.


The canonical Docker test targets are:

```bash
make docker-test
make docker-test-e2e
make docker-test-integration
make docker-test-runtime
```

The E2E, integration, and runtime targets provision disposable PostgreSQL and Redis services and run Go inside the test container. Go is not required on the host for these targets.

## Local development flow

```bash
cp .env.example .env
make docker-dev-up
make docker-migrate-up
make docker-dev-seed
make docker-dev-logs
```

For the base/production Compose stack:

```bash
make docker-up
make docker-seed
make docker-ps
make docker-logs
```

Production seed mode intentionally does not create demo users, demo passwords, demo role assignments, or sample products.

## API and contract notes

- Health endpoints: `/health` is a backward-compatible liveness alias; `/livez` is process liveness; `/readyz` checks PostgreSQL and Redis and returns `503` when dependencies are unavailable.
- Versioned API routes: `/api/v1/...`.
- Swagger UI is available at `/swagger/index.html` when enabled by the runtime configuration.
- Internal database identifiers are `BIGINT`; public resource identifiers are UUIDv7.
- Error responses use the centralized safe application error envelope.
- Raw internal database errors are not returned to clients.

## Operational follow-up

- Add a GitHub Actions workflow for pull-request fast checks and full PostgreSQL/Docker verification on `main`.
- Configure image publishing with immutable release tags instead of relying only on `latest`.
- Decide whether the assessment requires an explicit CI status badge and release/tag strategy.
- Keep `.env` and `.env.test` local-only; do not commit credentials or secrets.
