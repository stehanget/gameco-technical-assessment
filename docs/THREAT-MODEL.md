# Threat Model

## Scope and status

This document covers the current backend foundation and the agreed marketplace design in `docs/schema.dbml`.

The repository currently has implemented authentication, database-backed sessions, role/permission authorization, CORS, Redis-backed rate limiting, migrations, seeders, public catalog reads, seller product management, and buyer cart item management. Cart ownership, current-stock validation, cart-specific error contracts, and self-purchase prevention are implemented. Checkout, order, payment, stock-ledger, and audit runtime flows remain implementation scope; they must not be treated as implemented merely because they appear in the DBML or ADRs.

## Assets

- User credentials and password hashes.
- Access tokens, refresh tokens, and database session records.
- Role and permission assignments.
- User email and request metadata such as IP address and user agent.
- Product price, stock, and seller ownership.
- Cart and order data, including immutable order-item price snapshots.
- Payment references, webhook payloads, and payment state.
- Idempotency keys and request hashes.
- Audit events and stock movement ledger records.
- JWT signing secret, database credentials, Redis credentials, and webhook signing secret.

## Trust boundaries

1. **Public clients → HTTP API**
   Web SPA and mobile clients are untrusted public clients. They cannot keep a confidential secret.

2. **Partner server → partner API**
   A future partner API will be a separate confidential-client boundary. It is not implemented in the current repository and must not reuse a first-party public-client assumption without a separate ADR.

3. **API → PostgreSQL**
   PostgreSQL is the source of truth for users, access-control records, and authentication sessions.

4. **API → Redis**
   Redis provides shared rate-limit state and operational support. Redis must not become the only source of truth for session revocation.

5. **Payment provider → webhook endpoint**
   Payment callbacks are untrusted input. The future webhook handler must verify HMAC signatures, timestamp freshness, and event idempotency before changing payment or order state.

6. **API → worker/queue**
   Queued jobs are untrusted at the boundary and must be validated and authorized before performing mutations.

## Current mitigations and evidence

| Area | Current mitigation | Evidence |
| --- | --- | --- |
| Password storage | bcrypt hashing with the library default cost; plaintext passwords are not persisted | `internal/platform/auth/password.go` |
| JWT algorithm confusion | Parser accepts only `HS256` | `internal/platform/auth/jwt.go` |
| Session revocation | JWT `jti` is resolved against an active PostgreSQL session; a short-lived Redis cache is guarded by a revoke marker checked before positive cache entries | `internal/platform/middleware/auth.go`, `internal/modules/auth/repository.go`, `internal/modules/auth/service.go` |
| Refresh-token storage | Only a hash is stored in `auth_sessions`; refresh rotation revokes the previous session | `internal/modules/auth/service.go`, `migrations/20260723120500_create_auth_sessions.sql` |
| Authorization | Permission middleware defaults to deny when auth context is missing or malformed | `internal/platform/middleware/auth.go` |
| Rate limiting | Redis Lua script performs atomic increment and expiry; global traffic selects either an IP bucket or a user bucket, and limits are shared across API instances | `internal/platform/middleware/rate_limit.go`, `internal/config/rate_limit.go`, `internal/app/router.go` |
| CORS | Explicit localhost allowlist; wildcard origins and credentials are disabled | `internal/config/cors.go`, `internal/app/cors_test.go` |
| Request tracing | Request ID middleware exposes `X-Request-ID` | `internal/platform/middleware/request_id.go` |
| Schema integrity | Internal numeric IDs, unique public IDs, join-table keys, and declared foreign keys are documented/migrated | `migrations/`, `docs/schema.dbml` |
| Regression coverage | Register → login → refresh rotation → `/auth/me`, including refresh-token reuse handling, is covered by a black-box E2E test | `tests/e2e/auth_test.go` |
| Error responses | Shared error envelope, definition-based typed errors containing code/kind/message, status-based safe mapper, request ID propagation, structured logging, panic recovery, and handler migration away from raw error responses | `internal/platform/httpx/response.go`, `internal/platform/httpx/error_test.go`, `internal/platform/faults/error.go`, `internal/constants/error/definitions.go`, `internal/platform/middleware/request_id.go`, `internal/platform/middleware/logging.go`, `internal/platform/middleware/recovery.go`, `docs/ADR/0005-centralized-error-handling.md` |

## OWASP API Security Top 10 (2023) mapping

### API1:2023 Broken Object Level Authorization

**Risk:** A buyer or seller accesses another user's order or another seller's product by changing an identifier.

**Mitigation required:** Every marketplace read and mutation must scope the query by the authenticated actor and resource ownership, not by identifier alone. Seller product writes must include seller ownership in the database query. Buyer order reads must include buyer ownership. Public catalog reads must exclude soft-deleted products.

**Current status:** Public catalog reads and seller product handlers are implemented with seller-scoped queries and route-level authentication, role, permission, and ownership checks. Buyer order ownership and transaction authorization remain pending with the future order/checkout flows.

### API2:2023 Broken Authentication

**Risk:** Stolen, replayed, or forged access/refresh credentials create unauthorized sessions.

**Mitigations:** bcrypt password hashing, strict `HS256` verification, 15-minute access tokens, database-backed active-session lookup, short-lived session caching with revoke markers, atomic refresh rotation, refresh hash persistence, reuse detection, 24-hour refresh tokens, and server-side logout.

**Gaps/residual risk:** Login failure lockout state (`user_login_security_states`) is wired into the login service with a five-attempt, bounded lockout and reset-after-success behavior. MFA, manual unlock/audit workflow, and device/session management are not implemented. JWT rotation supports a current secret plus a bounded previous-secret list; secure secret-manager storage and rotation orchestration remain deployment responsibilities.

### API3:2023 Broken Object Property Level Authorization

**Risk:** Mass assignment lets a client modify protected fields such as role, seller ownership, price, stock, or order status.

**Mitigation required:** Use request DTOs with explicit allow-lists. Never bind request bodies directly into persistence models. Server-owned fields must be assigned by the service layer.

**Current status:** Auth, seller product, and cart DTOs are explicit allow-lists, including partial-update null rejection. Future checkout, payment, and order handlers must preserve this rule.

### API4:2023 Unrestricted Resource Consumption

**Risk:** Login, register, refresh, checkout, webhook, list, or search requests consume excessive CPU, database, Redis, or external-provider capacity.

**Mitigations:** Redis-backed shared rate limiting, request body limits, bounded query filters, cursor page-size limits, indexed catalog search, timeout-aware external calls, and endpoint-specific policies.

**Current status:** Global IP and authenticated-user rate limiting, plus auth-module policies, are implemented through the shared `middleware.RateLimit` middleware. Public catalog query bounds, opaque cursors, allow-listed sorting, and PostgreSQL trigram search indexes are implemented. Body-size limits, checkout limits, and payment-provider circuit breaking are still required for the future transaction scope.

### Public catalog query safety

**Risk:** A guest abuses broad search, deep pagination, or dynamic sorting to consume database CPU and expose hidden catalog rows.

**Mitigation required:** Keep the public query surface read-only; expose only allow-listed `sort_by` and `sort_order` values; use parameterized filters; cap `per_page`; use opaque cursor pagination with a deterministic tie-breaker; search only active, non-deleted products; and add PostgreSQL `pg_trgm` GIN indexes for the name and description search path. Do not expose internal IDs in cursors or responses as a public contract.

**Current status:** The catalog repository and handlers enforce active/non-deleted visibility, parameterized filters, allow-listed sorting, bounded page size, opaque cursor validation, and PostgreSQL trigram indexes. E2E, integration, and runtime coverage exists for the main list/detail paths; representative load testing remains pending.

### API5:2023 Broken Function Level Authorization

**Risk:** A buyer invokes seller/admin operations, or a normal user invokes audit or administrative actions.

**Mitigation:** Permission catalog constants and `RequirePermissions` middleware with default deny.

**Current status:** Auth permission enforcement and seller product route guards are implemented. Admin operations and buyer/order permissions remain pending with the future administrative and transaction surfaces.

### API6:2023 Unrestricted Access to Sensitive Business Flows

**Risk:** Automated checkout, payment retry, product scraping, or stock manipulation causes financial or inventory abuse.

**Mitigations required:** Idempotency keys, atomic stock mutation, order/payment state transitions, webhook replay protection, per-user and per-IP limits, and audit/stock ledgers.

**Current status:** The agreed tables are documented in `docs/schema.dbml`; the transaction implementation and critical tests are pending.

### API7:2023 Server Side Request Forgery

**Risk:** A future product-image URL or partner integration causes the API to fetch internal addresses or cloud metadata endpoints.

**Mitigation required:** Do not fetch client-provided URLs directly. If remote media is added, enforce an allow-list, resolve DNS safely, reject private/link-local ranges after resolution, limit redirects, and apply strict timeouts.

**Current status:** No remote URL fetch feature is implemented.

### API8:2023 Security Misconfiguration

**Risk:** Wildcard CORS, leaked secrets, verbose errors, unsafe Swagger exposure, or incorrect production defaults.

**Mitigations:** Explicit CORS allowlist, credentials disabled, environment-based configuration, recovery middleware, generated Swagger, and `.env.example` without real secrets.

**Residual risk:** Swagger is disabled in `production`. Production Compose still requires a deliberate manual hardening pass before deployment: explicit credentials, pinned application images, and internal-only PostgreSQL/Redis networking.

### API9:2023 Improper Inventory Management

**Risk:** Retired routes, legacy fields, or undocumented endpoints remain reachable and create an unexpected attack surface.

**Mitigation:** Route registration is centralized in `internal/app/router.go`; the example module is retained as scaffolding but is not registered. Documentation and generated Swagger are regenerated after route changes.

**Residual risk:** The marketplace surface is still under construction. Before release, audit route registration against the intended endpoint inventory and remove unused production handlers.

### API10:2023 Unsafe Consumption of APIs

**Risk:** Payment-provider responses or webhook payloads are trusted without signature, freshness, schema, or state validation.

**Mitigation required:** Verify HMAC with constant-time comparison, reject stale timestamps, deduplicate provider event IDs, validate amounts/references against local state, and apply explicit state transitions.

**Current status:** Payment tables are documented but the webhook implementation is pending.

## Concrete attack scenarios

### 1. Seller changes another seller's product

- **Attack:** Seller A submits Seller B's product public ID to the update endpoint.
- **Impact:** Unauthorized price, stock, or catalog mutation.
- **Mitigation:** Update query must constrain both `product.public_id` and `product.seller_id = authenticated_user_id`; protected fields must come from explicit DTOs.
- **Residual risk:** Seller product ownership is mitigated by seller-scoped queries and route/service regression coverage. Equivalent buyer/order ownership coverage remains pending until order endpoints exist.

### 2. Two checkout requests consume the last stock

- **Attack:** Send concurrent checkout requests for the same product with quantity equal to the remaining stock.
- **Impact:** Overselling and inconsistent order/payment state.
- **Mitigation:** Lock or conditionally update the product row inside the checkout transaction, insert `stock_movements` in the same transaction, and test concurrent requests.
- **Residual risk:** The design is recorded in the ADR/schema, but no runtime checkout implementation currently proves it.

### 3. Reuse a stolen refresh token

- **Attack:** Replay a refresh token after the legitimate client has rotated it.
- **Impact:** Unauthorized session continuation.
- **Mitigation:** Store only the hash, atomically revoke the old session during rotation, reject a missing active hash, detect reuse, and revoke all active sessions for the affected user. The refresh-token TTL is 24 hours.
- **Residual risk:** A token may still be used until rotation or revocation if stolen before the legitimate client rotates it. The shorter TTL and reuse response reduce but do not eliminate that window.

### 4. Replay or forge a payment webhook

- **Attack:** Send a fake payment-success callback or replay a valid callback after the order has changed state.
- **Impact:** Unauthorized order payment or duplicate fulfillment.
- **Mitigation:** HMAC verification, constant-time comparison, timestamp window, unique provider event ID, local payment-reference validation, and idempotent state transitions.
- **Residual risk:** Not mitigated until the payment webhook endpoint is implemented.

### 5. Brute-force a user account

- **Attack:** Send repeated login attempts from distributed IP addresses.
- **Impact:** Account compromise or user lockout denial of service.
- **Mitigation:** Shared IP rate limiting, progressive per-user login security state, generic invalid-credential responses, and bounded lockout behavior.
- **Residual risk:** Redis rate limiting and per-user security state are implemented, but the current lockout is time-bounded rather than manually unlockable. A future manual-unlock/audit flow must prevent attackers from indefinitely locking a victim account.

### 6. Abuse a client-provided image URL

- **Attack:** Submit a URL targeting localhost, private network services, or cloud metadata.
- **Impact:** SSRF and internal data disclosure.
- **Mitigation:** Do not fetch arbitrary URLs; use allow-listed remote media with DNS/IP/redirect validation if this feature is introduced.
- **Residual risk:** No image-fetching feature exists today.

### 7. Abuse public catalog search and pagination

- **Attack:** Send repeated broad substring searches, malformed cursors, or large page-size values to force expensive scans or repeatedly traverse the catalog.
- **Impact:** Database resource exhaustion, elevated latency, or accidental exposure of inactive/soft-deleted products.
- **Mitigation:** Apply the shared rate limiter, validate and cap `per_page`, reject invalid cursor/filter combinations, restrict sorting to an allow-list, use parameterized queries, add `pg_trgm` GIN indexes for name and description search, and enforce `active` plus `deleted_at IS NULL` visibility predicates in the repository query.
- **Residual risk:** The catalog runtime implementation and focused regression coverage are present. Index selectivity and search cost still need measurement against representative production-sized data.

### 8. Purchase a product owned by the authenticated buyer

- **Attack:** A seller uses the buyer capability of the same account to add their own product to a cart and later attempts to create an artificial sale.
- **Impact:** Manipulated sales, revenue, promotion, or review metrics.
- **Mitigation:** Cart item mutation compares the locked product `seller_id` with the authenticated actor ID and rejects a match with `commerce.self_purchase_not_allowed`. Checkout must repeat the same validation inside its transaction rather than trusting cart state.
- **Residual risk:** Cart-time prevention is implemented. Checkout-time defense-in-depth remains pending until the checkout slice exists.

## Logging and privacy rules

- Never log passwords, access tokens, refresh tokens, JWT signing secrets, webhook secrets, or payment-card data.
- Redact credentials in test fixtures and documentation.
- Include request IDs in logs and responses.
- Avoid logging full webhook payloads unless sensitive fields are redacted and retention is controlled.
- Treat email, IP address, and user agent as sensitive operational data.

## What is not safe yet

1. Cart lifecycle, cart ownership, current-stock validation, self-purchase prevention, and cart HTTP/integration/runtime coverage are implemented. Checkout, order, payment, stock-ledger, audit, idempotency, webhook verification, and critical transaction tests are not implemented yet.
2. `user_login_security_states` is wired into login failure tracking, five-attempt lockout, reset-after-success behavior, and PostgreSQL/integration/runtime coverage.
3. JWT key rotation is implemented through `APP_JWT_SECRET` plus a bounded `APP_JWT_PREVIOUS_SECRETS` verification list; secret-manager integration and automated rotation orchestration are not implemented.
4. Production Compose still requires a manual hardening pass in `deployments/compose/docker-compose.yml`: explicit deployment credentials, pinned application images, and internal-only PostgreSQL/Redis networking.
5. The current rate limiter uses global anonymous-IP and authenticated-user policies; additional business-flow policies remain to be designed for checkout, payment, and webhook endpoints.
6. Registration/login still need a dedicated anti-enumeration and timing-hardening review; lockout and generic invalid-credentials responses are implemented.
7. Error response foundations, definition-based error metadata, marketplace catalog errors, and cart-specific errors exist; stable domain codes and field-level validation details need to expand with the future transaction handlers.

These are intentionally documented as residual risks rather than hidden behind claims that the assessment scope is already complete.
