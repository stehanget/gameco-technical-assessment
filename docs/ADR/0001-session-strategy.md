# ADR-0001: First-Party Session Strategy

- **Status:** Accepted
- **Date:** 2026-10-20
- **Scope:** First-party web SPA and mobile clients

## Context

The API is consumed by public clients that cannot safely hold a client secret. Logout must invalidate a session server-side, refresh-token replay must not create an unlimited session, and roles and permissions must not be trusted from a client-controlled token.

The implementation also needs to remain operational across multiple API instances without making Redis the source of truth for authentication state.

## Options considered

1. Stateless JWT access and refresh tokens with no server-side session state.
2. Cookie-only sessions stored in Redis.
3. Short-lived JWT access tokens plus opaque refresh tokens backed by PostgreSQL sessions.

## Decision

Use short-lived signed JWT access tokens and opaque refresh tokens backed by PostgreSQL sessions.

### Access tokens

- The access token contains only identity/session claims: user subject and UUIDv7 `jti`, plus standard issued/expiry timestamps.
- Roles and permissions are resolved from the active database-backed session; they are not embedded in the JWT.
- The default access-token TTL is 15 minutes.
- Only `HS256` is accepted by the parser; unexpected algorithms, invalid signatures, and expired tokens are rejected.

### JWT secret rotation

- `APP_JWT_SECRET` is the current signing secret.
- `APP_JWT_PREVIOUS_SECRETS` is an ordered, bounded list used only to verify tokens issued before rotation.
- New tokens are always signed with the current secret.
- The previous-secret list is limited to three entries and rejects empty, duplicate, or current-secret duplicates.
- The token format does not require a `kid`; rotation is controlled by the current/previous secret configuration.

### Database-backed sessions

- PostgreSQL is the source of truth for `auth_sessions`.
- Only a hash of the opaque refresh token is stored.
- Middleware requires both a valid JWT and an active, non-revoked database session identified by `jti`.
- Redis may cache active session lookups for up to 30 seconds, but a revoke marker is checked before the positive cache entry.
- A cache miss, Redis failure, or invalid cache state falls back to PostgreSQL; Redis cannot make a revoked session active.

### Refresh rotation and reuse handling

Refresh rotation is performed atomically in a PostgreSQL transaction:

1. Lock the matching session row with `FOR UPDATE`.
2. Verify the session is active and the refresh-token hash matches.
3. Conditionally revoke the old session.
4. Create the replacement session and token pair.
5. Invalidate the old session cache and write the revoke marker.
6. Commit all changes together.

A refresh-token reuse attempt against an already revoked session triggers reuse handling and revokes all active sessions belonging to the affected user. The default refresh-token TTL is 24 hours.

### Logout and client transport

- Logout revokes the matching database session and invalidates its Redis support state.
- Clients send the access token as a Bearer token.
- The current API does not use cookie credentials.

## Consequences

### Positive

- Server-side logout and revocation are real, not client-only deletion.
- JWT payloads do not become a stale authorization catalog shared across sessions.
- Refresh-token database compromise does not reveal usable plaintext refresh tokens.
- PostgreSQL remains authoritative if Redis is lost or flushed.
- Concurrent refresh requests cannot both rotate the same active session successfully.
- Reuse detection limits the impact of a stolen refresh token by revoking the affected user's active sessions.
- Secret rotation does not require a `kid` format in environment configuration.

### Negative

- Session authorization remains mandatory on every authenticated request; Redis is only a bounded optimization and never replaces the PostgreSQL session decision.
- The implemented refresh rotation requires row locking, conditional updates, cache invalidation, and careful transaction handling.
- Public clients still need secure in-memory or platform-protected token handling. A stolen access token can be used until it expires or until its database session is revoked earlier.
- A stolen refresh token may be used before legitimate rotation or revocation; the 24-hour TTL limits but does not eliminate that window.
- Secret-manager storage and automated rotation orchestration remain deployment responsibilities.

## Evidence

- `internal/platform/auth/jwt.go`
- `internal/platform/auth/jwt_test.go`
- `internal/platform/middleware/auth.go`
- `internal/modules/auth/repository.go`
- `internal/modules/auth/service.go`
- `tests/e2e/auth_test.go`
- `migrations/20260723120500_create_auth_sessions.sql`
