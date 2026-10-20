# ADR-0004: Shared Rate Limiting and Explicit CORS

- **Status:** Accepted
- **Date:** 2026-10-20
- **Scope:** Public HTTP boundary

## Context

The API can run in multiple instances. An in-memory limiter would allow an attacker to bypass limits by distributing requests across instances. Browser clients also need an explicit cross-origin policy; wildcard CORS is unsafe for an authenticated API and does not express the intended client boundary.

The API also needs one global abuse-control middleware that can distinguish anonymous traffic from authenticated traffic without introducing a session-specific bucket for every access token.

## Options considered

1. In-memory rate limiting per process.
2. Redis-backed shared rate limiting with a global anonymous policy, a global authenticated-user policy, and module-owned special policies.
3. API gateway-only rate limiting with no application enforcement.

For CORS:

1. Wildcard origin.
2. Reflected request origin.
3. Explicit allowlist with credentials disabled.

## Decision

Use Redis-backed rate limiting with an atomic Lua increment/expiry script.

### Global rate-limit middleware

- The global policy is source-controlled in `internal/config/rate_limit.go` and wired in `internal/app/router.go`.
- The public middleware API is `middleware.RateLimit`.
- `RateLimitPolicySet` supports optional `Anonymous` and `Authenticated` policies; either policy may be omitted.
- A request selects exactly one policy:
  - no valid Bearer token → anonymous policy keyed by remote IP;
  - valid Bearer token → authenticated policy keyed by user ID;
  - omitted selected policy → the middleware continues without limiting that request class.
- The global defaults are 60 requests per minute for anonymous traffic and 300 requests per minute for authenticated users.
- The middleware parses a Bearer token only to select the rate-limit bucket. Full signature, expiry, active-session, revoke, and permission checks remain the responsibility of the authentication middleware.
- A session-specific rate-limit policy is intentionally not used.

### Module-specific policies

Module routes can call the same `middleware.RateLimit` function with only the policy class they need. Current auth policies are:

- public auth routes: 10 requests per minute per IP;
- protected `/auth/me`: 60 requests per minute per user.

These policies are source-controlled beside the owning routes and are not exposed as per-endpoint environment variables.

### Failure and response behavior

- Redis errors fail closed for the request with `503 Service Unavailable` rather than silently disabling protection.
- Exceeded requests receive `429 Too Many Requests` and rate-limit headers.
- Redis keys include the policy name and selected identity, preventing anonymous and authenticated counters from sharing state.

Use explicit CORS configuration:

- allowed origins: `http://localhost:3000` and `http://localhost:3001`;
- explicit methods and request headers;
- `AllowCredentials: false`;
- no wildcard origin;
- request ID and content-disposition are exposed headers.

## Consequences

### Positive

- Rate-limit state is shared across API instances.
- Atomic Redis operations avoid increment/expiry races.
- All API routes receive a consistent default policy from one router middleware.
- Authenticated traffic from multiple devices shares one user bucket.
- Session rotation or multiple access tokens do not create separate rate-limit buckets.
- Modules can apply tighter anonymous-only or authenticated-only policies without creating new middleware APIs.
- CORS behavior is centralized and testable.

### Negative

- Redis availability affects requests protected by the limiter.
- IP-only keys do not prevent distributed attacks from many IP addresses.
- Trusted proxy handling must be designed before relying on forwarded client IP headers.
- Token parsing occurs at the global rate-limit boundary and is followed by the full authentication/session validation path.
- The global security policy is source-controlled, so a policy change requires a code change and deployment.
- The allowlist must be changed when deployed frontend origins change.

## Evidence

- `internal/platform/middleware/rate_limit.go`
- `internal/platform/middleware/rate_limit_test.go`
- `internal/config/rate_limit.go`
- `internal/config/cors.go`
- `internal/app/router.go`
- `internal/modules/auth/route.go`
- `internal/app/cors_test.go`
