# Identifier Conventions

The repository uses two identifier layers:

- `id BIGINT`: internal database primary key and foreign-key target.
- `public_id UUIDv7`: stable API-facing identifier for externally addressable resources.

`public_id` is generated in the owning Go create flow, is immutable, and has a unique constraint. Request DTOs must not accept it during create or update operations.

Use stable role and permission `code` values for access-control catalogs. Relation tables do not need their own public identifiers.

The shared catalog constants live under:

```text
internal/constants/permission/codes.go
internal/constants/role/codes.go
```

Authentication uses the same split:

- `auth_sessions.id` remains an internal BIGINT.
- JWT `jti` is a UUIDv7 session identifier.
- Refresh tokens remain opaque random values; only their hashes are persisted.

Public identifiers do not replace authorization. Resource queries must still enforce ownership or explicit administrative capability in the same repository/service operation.
