# ADR-0004: Shared Rate Limiting and Explicit CORS

- **Status:** Accepted
- **Date:** 2026-10-20
- **Scope:** Public HTTP boundary

## Context

The API can run in multiple instances. An in-memory limiter would allow an attacker to bypass limits by distributing requests across instances. Browser clients also need an explicit cross-origin policy; wildcard CORS is unsafe for an authenticated API and does not express the intended client boundary.

The API needs bounded abuse controls without forcing one anonymous bucket and one authenticated bucket onto every endpoint. Webhook callbacks also need a higher-capacity policy than user-facing routes.

## Options considered

1. In-memory rate limiting per process.
2. Redis-backed route-specific rate limiting owned by each module.
3. One global API rate-limit middleware.
4. API gateway-only rate limiting with no application enforcement.

For CORS:

1. Wildcard origin.
2. Reflected request origin.
3. Explicit allowlist with credentials disabled.

## Decision

Use Redis-backed rate limiting with an atomic Lua increment/expiry script.

### Route-specific rate-limit middleware

- Rate-limit policy values are source-controlled beside the owning routes; there is no automatically installed global limiter or rate-limit configuration section.
- The public middleware API is `middleware.RateLimit`.
- `RateLimitPolicySet` supports optional `Anonymous` and `Authenticated` policies; either policy may be omitted.
- The selected route policy is keyed by remote IP for anonymous traffic or by validated Bearer user ID when a valid token is present.
- The global anonymous policy is `global-anonymous`, and the global authenticated policy is `global-authenticated`.
- All non-auth, non-payment-webhook routes that opt into the shared global policy use 60 requests per minute for anonymous traffic and 120 requests per minute for authenticated traffic. Anonymous traffic is keyed by remote IP; authenticated traffic is keyed by validated Bearer user ID.
- Auth routes retain their dedicated tighter policies: 10 requests per minute per IP for public auth routes and 60 requests per minute per user for protected `/auth/me`.
- The payment webhook retains its dedicated anonymous provider-callback policy of 10,000 requests per minute per source IP.
- The global policy is shared across opted-in modules: catalog public routes share the anonymous bucket, while authenticated catalog, cart, and checkout/order routes share the authenticated user bucket.
- Other routes do not receive an application-wide default policy until their module explicitly opts into the global policy.
- The middleware parses a Bearer token only to select the rate-limit bucket. Full signature, expiry, active-session, revoke, and permission checks remain the responsibility of the authentication middleware.

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
- Route-specific Redis policies protect auth and payment webhook traffic.
- Authenticated traffic from multiple devices shares one user bucket where a protected route opts into that policy.
- Webhooks are not double-limited by a generic anonymous bucket.
- Modules can apply tighter anonymous-only or authenticated-only policies without creating new middleware APIs.
- CORS behavior is centralized and testable.

### Negative

- Redis availability affects requests protected by the limiter.
- IP-only keys do not prevent distributed attacks from many IP addresses.
- The application has no generic protection for routes that do not opt into a policy; deployment-level gateway/WAF controls and future module-owned policies remain responsible for those surfaces.
- Trusted proxy handling must be designed before relying on forwarded client IP headers.
- Policy changes require a code change and deployment.

## Evidence

- `internal/platform/middleware/rate_limit.go`
- `internal/platform/middleware/rate_limit_test.go`
- `internal/modules/checkout/route.go`
- `internal/config/cors.go`
- `internal/app/router.go`
- `internal/modules/auth/route.go`
- `internal/app/cors_test.go`
