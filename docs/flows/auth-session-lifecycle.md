# Authentication and Session Lifecycle

This document traces registration, login, refresh-token rotation, refresh-token reuse detection, and logout.

## Flow overview

```mermaid
flowchart LR
    A[Register] --> B[Create user]
    B --> C[Issue access + refresh token pair]
    C --> D[Authenticated session]
    D --> E[Refresh]
    E --> F[Rotate refresh token]
    F --> D
    D --> G[Logout]
    G --> H[Revoke database session]
    E --> I{Refresh token reused?}
    I -- Yes --> J[Revoke session and audit reuse]
    I -- No --> F
```

## Login flow

```mermaid
sequenceDiagram
    autonumber

    participant C as Client
    participant R as Auth Route
    participant M as Auth Middleware
    participant H as Auth Handler
    participant S as Auth Service
    participant DB as PostgreSQL
    participant Redis as Redis
    participant JWT as JWT Manager

    C->>R: POST /api/v1/auth/login
    R->>M: Rate limit request
    M->>H: Continue
    H->>S: Login(email, password)
    S->>DB: Load user and login security state
    S->>S: bcrypt compare
    alt Invalid credentials
        S->>S: Dummy bcrypt compare if email unknown
        S->>DB: Record failed attempt/lockout state
        S->>DB: Write auth.login_failed audit event
        S-->>H: Generic invalid credentials
        H-->>C: 401 Unauthorized
    else Valid credentials
        S->>DB: Reset failed login state
        S->>DB: Create auth session and refresh hash
        S->>JWT: Sign access token
        S->>DB: Write auth.login_succeeded audit event
        S-->>H: Token pair
        H-->>C: 200 OK
    end
```

## Refresh-token rotation

```mermaid
sequenceDiagram
    autonumber

    participant C as Client
    participant H as Auth Handler
    participant S as Auth Service
    participant DB as PostgreSQL
    participant Redis as Redis
    participant JWT as JWT Manager

    C->>H: POST /api/v1/auth/refresh
    H->>S: Refresh(refresh token)
    S->>S: Hash presented refresh token
    S->>DB: Load and lock active session
    S->>DB: Compare stored refresh hash

    alt Valid unused refresh token
        S->>S: Generate replacement refresh token
        S->>DB: Persist new refresh hash and rotation state
        S->>JWT: Sign new access token
        S->>Redis: Refresh session cache/revocation state
        S->>DB: Write auth.refresh_succeeded audit event
        S-->>C: New access + refresh token pair
    else Reused or invalid refresh token
        S->>DB: Revoke session when reuse is detected
        S->>DB: Write auth.refresh_token_reused audit event
        S-->>C: 401 Unauthorized
    end
```

## Logout

Logout revokes the authenticated database session. Redis is treated as a cache/acceleration layer, not the source of truth.

```mermaid
sequenceDiagram
    participant C as Client
    participant H as Auth Handler
    participant S as Auth Service
    participant DB as PostgreSQL
    participant Redis as Redis

    C->>H: POST /api/v1/auth/logout
    H->>S: Logout(actor, session metadata)
    S->>DB: Revoke session
    S->>Redis: Remove or invalidate cached session state
    S->>DB: Write auth.logout audit event
    S-->>C: 204 No Content
```

## Security behavior

- Unknown-email login performs a dummy bcrypt comparison to reduce timing differences.
- Unknown-email and wrong-password responses use the same invalid-credentials contract.
- Failed-login audit metadata uses a sanitized one-way reference, not a raw password or token.
- Access tokens are short-lived and checked against active database-backed sessions.
- Refresh tokens are stored as hashes, never as raw token values.
- Refresh-token reuse revokes the session and produces a security audit event.
- Registration currently returns a distinct existing-email conflict by intentional assessment contract; full verification-first registration remains deferred.

## Source-code map

- Routes: `internal/modules/auth/route.go`
- HTTP handlers: `internal/modules/auth/handler.go`
- Service and token/session lifecycle: `internal/modules/auth/service.go`
- Persistence and refresh reuse handling: `internal/modules/auth/repository.go`
- JWT/session middleware: `internal/platform/middleware/auth.go`
- Audit writer: `internal/platform/audit/audit.go`
- Session decision: `docs/ADR/0001-session-strategy.md`
