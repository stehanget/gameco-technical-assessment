# Coding Conventions

This document is the repository's source of truth for writing new Go code and reviewing changes. It is intentionally pragmatic: conventions that improve contract clarity, security, testability, and maintainability are mandatory; formatting preferences that do not affect behavior should stay lightweight.

## 1. Scope and quality gates

Before opening or merging a change, run the narrowest useful checks first, then the complete repository gates:

```text
make fmt
make fmt-check
make lint
go test ./...
go build ./...
go vet ./...
git diff --check
```

The optional pre-commit hook runs `make check` (format check, lint, verbose tests, vet, and diff check). Generated Swagger must be regenerated whenever route annotations or API DTO contracts change:

```text
make swagger
```

Do not claim a check passed unless the command actually ran successfully. Keep unrelated working-tree changes out of the change being reviewed.

## 2. Package and module boundaries

Use the existing modular-monolith layout:

```text
internal/app/        bootstrap, dependency wiring, router registration
internal/config/     environment-backed configuration and validation
internal/constants/  reusable role, permission, status, and error catalogs
internal/modules/    feature-owned model, DTO, repository, service, handler, route
internal/platform/   cross-cutting infrastructure and adapters
internal/seeds/      idempotent bootstrap seeders
internal/workers/    background job and worker code
```

Keep dependency direction explicit:

```text
handler -> service -> repository -> database
handler -> platform/httpx and platform/faults
service -> constants and platform abstractions
```

Feature-specific rules belong in the feature module. Reusable cross-cutting rules belong under `internal/platform` or `internal/constants`; do not create a generic helper merely to avoid a small amount of local code.

Each module owns its route registration. Register shared repository/service instances from app wiring when multiple handlers need the same dependency rather than silently constructing parallel stateful instances.

## 3. Naming and public identifiers

- Use Go naming conventions: `MixedCaps` for exported identifiers and concise local names.
- Name path parameters after the resource they identify:
  - `:product_public_id`, not `:public_id`;
  - `:comment_public_id` for a comment;
  - nested paths must remain unambiguous: `/products/{product_public_id}/comments/{comment_public_id}`.
- Keep route registration, `c.Param(...)`, Swagger `@Param`, Swagger `@Router`, and tests synchronized when a path parameter is renamed.
- Internal primary and foreign keys remain numeric `BIGINT`/Go integer persistence fields.
- Public-facing identifiers use UUIDv7 and are exposed as `public_id`; internal IDs must not be returned in API DTOs or cursors.
- Public ID lookup methods should be named explicitly, for example `FindByPublicID`.
- Do not use generic `id` when the value is a public identifier in an API contract.

## 4. API DTOs and HTTP contracts

Separate API DTOs from persistence models. Do not serialize GORM models directly from handlers.

### Timestamp contract

- API response timestamps use `timestamp.Timestamp`.
- Nullable API timestamps use `*timestamp.Timestamp`.
- Persistence models may use `time.Time` and `*time.Time`.
- PostgreSQL timestamp columns use `TIMESTAMPTZ`.
- API output is RFC3339Nano in UTC with a `Z` suffix.
- API input requires RFC3339/RFC3339Nano with an explicit timezone; offsets such as `+07:00` are accepted and normalized to UTC.
- Timezone-less, date-only, and Unix timestamp input are not part of the API contract.
- Do not format timestamps manually in handlers or DTO mappers with `time.Format(...)`.
- Use `timestamp.New(...)` when mapping persistence values into API DTOs.
- Use `timestamp.Parse(...)` for cursor or API timestamp parsing; do not define a second layout.
- Use injectable `clock.Clock` for time-sensitive logic such as expiry, rotation, rate windows, deadlines, and retry behavior.
- Do not force database-owned `created_at`/`updated_at` values through a service clock when the database is the source of truth.

Swagger timestamp fields should be documented as `string`, `format: date-time`, with a canonical UTC example. If a custom Go type is rendered as an object by Swagger, fix the source DTO annotation and regenerate Swagger; do not hand-edit generated artifacts.

### Response presence, requiredness, and nullability

Client-facing response DTOs must keep a stable JSON shape. Do not use `omitempty` merely to hide a nil or zero value when the contract says the field exists but may be null. Return explicit `null` instead:

```go
type ProductResponse struct {
	Name        string         `json:"name"`
	Description *string        `json:"description"`
	Category    *CategoryDTO   `json:"category"`
	Alternatives []ProductDTO  `json:"alternatives"`
}
```

The intended contract is:

- a field guaranteed by the backend is marked/documented as **required**;
- a field that may legitimately have no value uses a nullable pointer and is serialized as `null` when absent;
- an array that is part of the response contract returns `[]` when empty, not an omitted field or an accidental `null`, unless the API explicitly defines the array itself as nullable;
- `omitempty` is reserved for fields whose absence is intentionally part of the contract, not as a general response cleanup mechanism;
- request DTO optionality and response DTO presence are separate decisions; do not infer response behavior from request binding tags.

Keep required/nullable declarations synchronized across the DTO, handler behavior, tests, and generated Swagger. Verify the generated schema's `required` list and nullable representation instead of assuming a struct tag was interpreted correctly.

For nullable scalar fields, explicit nullable metadata may be used when it produces the intended OpenAPI contract. For nullable objects or arrays of objects, avoid adding `x-nullable` when the Swagger generator renders the example as a string instead of an object or array. In that case, preserve the correct Go pointer/slice shape, use a source-level schema/example annotation that keeps the generated example structured, and verify the browser-rendered Swagger example after regeneration. Never hand-edit generated Swagger as the primary fix.

### Request binding and validation

- Use explicit `json`, `form`, `uri`, and `binding` tags.
- `binding:"required"` is for required request input; a struct field without it is optional.
- Use pointers when absence differs from an explicit zero value, for example `*int64` for optional price bounds.
- Validate at the HTTP boundary, then apply business invariants in the service layer.
- Reject unknown or malformed public identifiers at the boundary.
- Server-owned lifecycle fields must not be accepted from client request bodies.
- Keep response envelopes consistent with `internal/platform/httpx`.
- Do not expose raw database IDs, password hashes, refresh-token hashes, internal errors, or secret material.

## 5. Routes, handlers, and errors

Handlers should remain thin:

1. bind and validate request input;
2. call the application service with `c.Request.Context()`;
3. map the service result to the API response;
4. route all failures through `httpx.HandleError(c, err)`.

Use the centralized error flow:

```text
faults.New(...) -> httpx.HandleError(...)
```

- Create typed application errors with `faults.New(errorconst.SomeDefinition, cause)`.
- Define each public error once as a `faults.Definition` in `internal/constants/error/definitions.go`; the definition owns its code, kind, and safe default message.
- Do not add domain-error mappings to `internal/platform/httpx/response.go`; that layer only maps generic fault kinds to HTTP statuses and renders the definition metadata.
- Never send `err.Error()` or database/driver details directly to clients.
- Preserve request IDs in success metadata and error responses where middleware has created one.
- Keep HTTP status mapping centralized; handlers should not duplicate error-to-status logic.
- Keep Swagger annotations next to the handler that owns the endpoint.
- Use `/health` only for the unversioned operational health route; versioned business APIs use `/api/v1/...`.

## 6. SQL and repository conventions

### Readability

Use ordinary quoted strings for short one-line SQL fragments:

```go
Where("p.status = ?", catalog.ProductStatusActive)
Joins("JOIN categories AS c ON c.id = p.category_id")
```

Use raw strings/backticks when the SQL is genuinely multiline or becomes easier to inspect:

```go
Select(`
    p.public_id,
    p.name,
    p.created_at
`)
```

Use backticks for readability, not automatically for every SQL expression. Do not put backticks around a one-line query merely because it is SQL.

### Safety

- Always use parameter binding for values.
- Never concatenate user input into SQL.
- Dynamic SQL fragments such as sort columns and directions must come only from an explicit allow-list.
- `fmt.Sprintf` may assemble a SQL fragment only after every dynamic fragment has been selected from a trusted allow-list; values still use placeholders.
- Escape `LIKE` wildcards when literal search semantics are intended, while retaining parameter binding.
- Keep query context-aware with `WithContext(ctx)`.
- Select only the columns required by the read model; do not accidentally expose internal fields.
- Use deterministic ordering for pagination. Include a stable public-ID tie-breaker.
- Cursor payloads are opaque to clients and must validate their filter/sort context before use.
- Translate `gorm.ErrRecordNotFound` and other storage errors at the service/application boundary; do not leak ORM errors through HTTP.
- Keep transaction boundaries in the service/application layer when a business operation spans multiple repository writes.

## 7. Pagination, search, and sorting

- Prefer opaque cursor pagination for public catalog/list APIs.
- Keep `per_page` bounded with a documented default and maximum.
- Cursor fields must use public identifiers and shared timestamp formatting; never store internal database IDs in a public cursor.
- Search input must be normalized consistently and remain parameterized.
- Search over multiple text fields must have an intentional index strategy; catalog name/description search uses the repository's trigram indexes.
- Sort fields and directions must be allow-listed. Never pass raw query parameters directly to `ORDER BY`.
- Preserve the same filter and sort combination when generating and consuming a cursor.

## 8. Authentication, authorization, and secrets

- JWT access tokens carry minimal identity/session claims; roles and permissions are resolved from the authorization source rather than trusted as a stale token snapshot.
- Session source of truth is PostgreSQL. Redis is a cache/operational accelerator and must not silently become the only session authority.
- Refresh-token values are stored only as hashes; never log or persist raw refresh tokens.
- Passwords, bearer tokens, signing secrets, API keys, connection strings, CVCs, and one-time codes must never appear in source, logs, tests, checkpoints, or documentation. Use `[REDACTED]` in evidence.
- Use constants/catalogs for role, permission, status, and error codes. Do not scatter magic strings across services or seeders.
- Authorization must fail closed. Missing authentication context or permission is an explicit error, not an implicit full-access path.
- Public routes and authenticated routes must use the route-level middleware policy defined in app wiring.
- CORS allowlists must be explicit; do not use wildcard origins with credentials.

## 9. Time, concurrency, and idempotency

- Use `context.Context` for cancellation and request-scoped work.
- Never create unbounded goroutines from a request handler.
- Use injectable clocks for expiry and deadline decisions; avoid `time.Now()` in business logic.
- Make retryable mutations idempotent with a stable key or uniqueness constraint where the domain requires it.
- Define transaction ownership before adding a repository method that writes multiple tables.
- For stock, payment, session, and order state transitions, enforce invariants in both application logic and database constraints where possible.
- Treat cache invalidation and queue publication as part of the consistency design, not as incidental side effects.

## 10. Logging and observability

- Use structured `slog` fields, not ad-hoc concatenated log strings.
- Include request ID, route template, HTTP status, duration, and safe error metadata where applicable.
- Log route templates rather than raw paths when raw paths may contain identifiers.
- Redact authorization headers, tokens, passwords, secrets, API keys, connection details, and sensitive request fields.
- Do not log full request/response bodies by default.
- Use status-aware log levels and preserve panic diagnostics on the server side without exposing stack traces to clients.
- Validate or safely generate request IDs; do not trust arbitrary malformed IDs as internal correlation identifiers.

## 11. Configuration and environment

- Keep environment loading and validation in `internal/config`.
- Keep deployment-tunable values in config/env when they are genuinely operational settings.
- Keep fixed security policy owned by the middleware or module enforcing it; do not create one environment variable per endpoint policy.
- Keep common Compose environment in the base file and environment-specific differences in the dev overlay.
- Never commit real `.env` values. Keep `.env.example` sanitized and complete enough to explain required configuration.
- When adding a config field, update loader, validation, example environment, Compose wiring, and relevant docs together.

## 12. Migrations, schema, and seeders

- Treat migrations and `docs/schema.dbml` as one contract. Keep tables, indexes, constraints, enums, and foreign keys synchronized.
- Prefer reset-safe canonical migrations while the project is pre-deployment; do not add compatibility layers without a concrete upgrade requirement.
- Use PostgreSQL-native types and constraints for closed status vocabularies when the domain requires them.
- Give indexes names that reveal their table and purpose.
- Keep migration ownership aligned with the domain that owns the table.
- Seeders must be modular, idempotent, deterministic, and safe to run repeatedly.
- Resolve bootstrap relationships using stable public/business identifiers such as slug or demo email, not guessed internal IDs.
- Keep reusable status constants in `internal/constants`; keep seeder-only bootstrap identifiers in `internal/seeds/bootstrap_identifiers.go`.
- Do not seed operational carts, orders, payments, audit events, or stock movements unless the test scenario explicitly requires them.

## 13. Tests

- Put unit/package tests next to the package they cover.
- Put end-to-end HTTP tests under `tests/e2e`.
- Use a separate test database/resource per test when shared state can cause order dependence.
- Prefer fixed clocks over sleeps for time-sensitive tests.
- Test both contract and behavior: HTTP status, response envelope, public fields, error code, request ID behavior, and persistence effects.
- Add regression tests for every security or contract bug.
- For cursor APIs, test first page, next page, tie-break ordering, invalid cursor, filter mismatch, and page-size limits.
- For seeders, run the seeder twice and verify counts/uniqueness remain correct.
- For migrations, verify fresh up, down-to-zero, fresh reset up, validation, indexes, and foreign keys.
- Use sanitized fixtures; never place real credentials or tokens in test data.

## 14. Documentation and generated artifacts

- Source annotations and runtime routes are the source of truth; generated Swagger files are artifacts.
- Update source route, handler parameter lookup, Swagger annotations, tests, and generated docs together.
- Keep ADRs aligned with actual runtime behavior. Mark planned or unverified behavior explicitly instead of documenting it as complete.
- Use exact paths, commands, public codes, timestamp formats, and response envelopes in docs.
- Do not paste secrets, raw tokens, or sensitive logs into ADRs, evidence reports, or examples.

This document describes the intended standard. Passing formatting and linting alone does not prove that a change satisfies the API, security, migration, or consistency rules above.
