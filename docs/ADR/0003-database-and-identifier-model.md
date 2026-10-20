# ADR-0003: PostgreSQL Data Model and Public Identifiers

- **Status:** Accepted
- **Date:** 2026-10-20
- **Scope:** Persistence and API resource identity

## Context

The API needs stable internal relationships, efficient joins, safe public resource references, and a schema that can enforce integrity at the database layer. Exposing sequential internal IDs directly makes enumeration easier and couples the public contract to storage details.

## Options considered

1. Use UUIDs for every primary and foreign key.
2. Use auto-increment numeric IDs everywhere, including API responses.
3. Use internal numeric IDs for relational storage and separate UUIDv7 public identifiers for resources exposed by the API.

## Decision

Use PostgreSQL with internal auto-increment `BIGINT` primary/foreign keys and application-generated UUIDv7 `public_id` values for externally exposed resources.

- Internal joins use numeric IDs.
- Public resource DTOs expose `public_id`, not internal `id`.
- JWT `jti` values use UUIDv7 format and identify the auth session.
- Money is stored as integer minor units; floating point is not used for price or total fields.
- Product soft-delete uses nullable `deleted_at`, separate from the normal product lifecycle `status`.
- Current authentication schema is created by Goose migrations.
- Marketplace schema is documented in `docs/schema.dbml` and must be added through migrations before marketplace endpoints are enabled.

## Consequences

### Positive

- Internal joins and foreign-key indexes remain compact and predictable.
- Public IDs are not simple row counters and are sortable by UUIDv7 creation time.
- API contracts are decoupled from internal storage identifiers.
- PostgreSQL constraints, indexes, and foreign keys can enforce core integrity.
- Integer money values avoid floating-point rounding errors.

### Negative

- Models and DTOs must explicitly map internal IDs to public IDs.
- Every public resource needs a UUIDv7 generation and uniqueness constraint.
- Developers must not accidentally return or accept internal IDs in public contracts.
- DBML and migration files must be updated together whenever the physical schema changes.

## Evidence

- `internal/platform/identifier/public_id.go`
- `docs/identifier-conventions.md`
- `migrations/20260701090500_create_users.sql`
- `migrations/20260701091000_create_access_control.sql`
- `migrations/20260723120500_create_auth_sessions.sql`
- `docs/schema.dbml`
