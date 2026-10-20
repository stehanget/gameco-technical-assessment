# ADR-0007: Timestamp and Clock Conventions

- **Status:** Accepted
- **Date:** 2026-10-21
- **Scope:** API timestamp wire format, persistence boundaries, cursor timestamps, and testable server time

## Context

The backend exposes timestamps from database-backed entities and uses time values for authentication, session expiry, rate-limit reset calculations, and cursor pagination. Using `time.Time` directly at every API boundary makes the wire contract implicit and allows timezone offsets, precision, and parsing behavior to drift between modules.

The database already uses PostgreSQL `TIMESTAMPTZ`, but persistence correctness alone does not define a consistent client contract. Server time also needs an injectable abstraction so expiration and time-sensitive tests do not depend on wall-clock timing.

## Options considered

1. Let each module serialize `time.Time` using Go's default JSON behavior and choose its own input parser.
2. Standardize on Unix timestamps for all API input and output.
3. Standardize API timestamps on RFC3339Nano UTC, normalize accepted offsets at the boundary, and provide shared timestamp and clock helpers.
4. Accept timezone-less local timestamps and interpret them using the server's local timezone.

## Decision

Use RFC3339Nano UTC as the canonical API timestamp format and provide shared platform helpers.

## API wire contract

All non-null timestamps returned by the API use RFC3339Nano in UTC, with the `Z` suffix:

```text
2026-10-21T10:15:30.123456Z
```

The API must not return local timezone labels, server-local offsets, date-only values, or Unix timestamps for these fields.

Client-provided timestamps may contain an explicit RFC3339 offset:

```text
2026-10-21T17:15:30.123456+07:00
2026-10-21T10:15:30.123456Z
```

The boundary parser accepts RFC3339/RFC3339Nano values with a required timezone, converts them to UTC, and rejects timezone-less values such as:

```text
2026-10-21T17:15:30
```

Invalid timestamp input uses the centralized validation error contract.

## Persistence boundary

PostgreSQL timestamp columns use `TIMESTAMPTZ`. Persistence and response mapping normalize values to UTC. Database models may continue to use `time.Time` and `*time.Time` because those types belong to the persistence layer; API DTOs use the shared timestamp type when timestamps are exposed over HTTP.

Server-owned lifecycle fields such as `created_at` and `updated_at` are not accepted from client request bodies. They are assigned by persistence/database behavior and returned using the canonical API format.

## Shared helpers

The platform timestamp package owns the wire contract:

```text
internal/platform/timestamp/
```

It provides:

- `timestamp.Parse` for strict RFC3339/RFC3339Nano input with a required timezone;
- `timestamp.New` for UTC-normalized values;
- JSON marshal/unmarshal behavior for canonical API output/input.

The platform clock package owns server-time abstraction:

```text
internal/platform/clock/
```

It provides a `Clock` interface, a UTC system clock, and a fixed clock for deterministic tests. Time-sensitive components should depend on the interface when their behavior needs controlled time.

Catalog cursor timestamps use the shared timestamp formatter/parser instead of defining a second layout.

## Consequences

### Positive

- Frontend and backend share one explicit timestamp contract.
- Offsets from clients are interoperable but become canonical UTC immediately.
- Timezone-less ambiguous input is rejected.
- Cursor timestamp ordering uses the same format as the public API.
- Time-sensitive tests can use fixed time without sleeps or wall-clock assumptions.
- PostgreSQL remains the persistence source of truth with timezone-aware columns.

### Negative

- Existing API DTOs exposing timestamps must migrate to the shared type as their contracts are touched.
- Components that still call `time.Now()` directly remain candidates for clock injection.
- Clients must handle RFC3339 timestamps instead of relying on local display formats.
- Timestamp parsing and serialization require focused contract tests.

## Implementation status

Implemented foundation:

- `internal/platform/timestamp` provides strict parsing, UTC normalization, and JSON behavior.
- `internal/platform/clock` provides system and fixed clocks.
- JWT access-token generation accepts an injectable clock while preserving the existing production constructor.
- Catalog product DTOs and cursor timestamp handling use the shared timestamp helper.

Remaining migration work:

- inject the clock into auth session, rate-limit, and payment/order time-sensitive services as those modules are implemented or refactored;
- migrate additional timestamp-bearing API DTOs when their endpoint contracts are added;
- add end-to-end timestamp contract assertions to future marketplace endpoints.

## Evidence and requirements

- `docs/technical-assessment.md`, section 3.4: validated input and consistent output.
- `docs/technical-assessment.md`, section 4: correctness and maintainable architecture.
- `internal/platform/timestamp/timestamp.go`
- `internal/platform/clock/clock.go`
- `internal/platform/auth/jwt.go`
- `internal/modules/catalog/dto.go`
- `internal/modules/catalog/service.go`
- PostgreSQL migrations using `TIMESTAMPTZ`
