# ADR-0005: Centralized HTTP Error Handling

- **Status:** Accepted; marketplace domain-error mapping pending
- **Date:** 2026-10-21
- **Scope:** HTTP transport error responses and request-correlated logging

## Context

The technical assessment requires every error response to use a consistent format and prohibits leaking stack traces, framework versions, SQL statements, or another user's PII. Errors must remain useful to clients while keeping implementation details in server logs.

The backend already has shared response helpers, request IDs, structured request logging, and panic recovery. However, a centralized response envelope alone is not sufficient when handlers pass raw `err.Error()` values into that envelope. Known domain errors need stable transport semantics, while unknown errors need a safe generic response.

## Options considered

1. Let every handler choose its own HTTP status and serialize `err.Error()` directly.
2. Define response helpers only, but keep error classification and message selection inside every handler.
3. Return typed/sentinel application errors from lower layers and map them centrally at the HTTP boundary.
4. Expose a framework error payload and rely on Gin's default recovery behavior.

## Decision

Use centralized HTTP error handling at the transport boundary.

### Error flow

The intended flow is:

```text
repository/service
    → typed domain or infrastructure error
    → centralized HTTP error mapper
    → safe ErrorResponse with request_id
    → structured server log for diagnostic details
```

Handlers may choose a success response and invoke the shared error writer, but they must not expose raw internal error text to clients.

### Error classification

The mapper must preserve the public HTTP contract for known errors and collapse unknown failures into a safe internal error:

| Error class | HTTP status | Client message behavior |
| --- | ---: | --- |
| malformed JSON or invalid boundary input | 400 | safe validation message; no parser internals |
| unauthenticated request | 401 | generic authentication message |
| authenticated but unauthorized request | 403 | generic authorization message |
| resource not found | 404 | stable not-found message; no query details |
| conflict, duplicate, or idempotency mismatch | 409 | stable conflict message |
| rate limit exceeded | 429 | rate-limit message and limiter headers |
| dependency unavailable or timeout | 503 | safe availability message |
| unknown or internal error | 500 | generic internal-server-error message |

The exact domain error registry may grow per module, but the transport mapping and fallback rules must remain centralized and consistent.

### Response contract

All HTTP errors use the shared error envelope:

```json
{
  "success": false,
  "message": "safe client-facing message",
  "request_id": "uuid"
}
```

Future machine-readable error codes or field-level validation details must be added to the shared response type rather than introduced as module-specific top-level shapes.

Stable public error codes and default client-facing messages are maintained together in `internal/constants/error`. Until i18n is implemented, the transport mapper consumes the static message constants. When i18n is introduced, the message source can be replaced without changing error codes or handlers.

### Security and logging rules

- Never return `err.Error()` for unknown or infrastructure errors.
- Never return stack traces, SQL, framework details, credentials, tokens, or unrelated PII.
- Log the original error server-side with the request ID and structured context.
- Preserve the request ID in the response header and error body when available.
- Do not log passwords, access tokens, refresh tokens, signing secrets, webhook secrets, or payment-card data.
- Panic recovery must use the same safe 500 error envelope as ordinary internal failures.

### Layering rule

Domain and repository layers return typed/sentinel errors and do not depend on Gin or HTTP status codes. The HTTP handler or a transport mapper converts those errors into the public response contract. This keeps business logic independent from the framework and prevents transport policy from being duplicated across services.

## Consequences

### Positive

- Every endpoint can expose the same predictable error envelope.
- Internal implementation details remain in server logs instead of client responses.
- Request IDs make client-visible failures traceable to structured logs.
- Domain errors retain meaningful status semantics without coupling services to HTTP.
- Unknown errors have a safe default instead of accidental information disclosure.
- Error behavior can be tested once at the mapper/HTTP boundary and reused by modules.

### Negative

- The mapper and domain-error catalog require ongoing maintenance as marketplace modules are added.
- Handlers must stop passing raw errors directly to the response helper.
- Some errors require explicit classification to distinguish `400`, `404`, `409`, and `503` correctly.
- Generic internal messages can make debugging harder if structured server logging is incomplete.
- HTTP handlers and middleware use `httpx.HandleError` as the single response error entry point.

## Current implementation status

Implemented foundation:

- `internal/platform/httpx/response.go` provides shared success/error envelopes and the single `HandleError` entry point.
- `internal/platform/middleware/request_id.go` generates or propagates request IDs.
- `internal/platform/middleware/logging.go` includes request IDs in structured logs.
- `internal/platform/middleware/recovery.go` converts panics into a safe 500 response through `HandleError`.

Remaining implementation work:

- extend typed domain-error classification to marketplace modules;
- add marketplace-specific error codes and field-level validation details where the frontend contract requires them;
- add handler/router tests proving that unknown errors never leak internal details and known domain errors retain their intended status.

## Evidence and requirements

- `docs/technical-assessment.md`, section 3.4: consistent error formatting and no sensitive implementation leakage.
- `docs/technical-assessment.md`, section 3.6: request ID/correlation ID in logs and response headers.
- `internal/platform/httpx/response.go`
- `internal/platform/middleware/request_id.go`
- `internal/platform/middleware/logging.go`
- `internal/platform/middleware/recovery.go`
- `internal/platform/faults/error.go`
- `internal/constants/error/codes.go`
- `internal/constants/error/messages.go`
- `internal/modules/auth/handler.go`
- `internal/modules/auth/service.go`
- `internal/modules/example/handler.go`
