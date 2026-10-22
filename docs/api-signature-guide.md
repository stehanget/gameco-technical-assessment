# API Signature Guide

## Purpose

API request signing authenticates trusted server-to-server integrations and protects the signed request from payload, path, and query-string changes before it reaches the API.

This mechanism is separate from user authorization:

- `Authorization: Bearer ***` identifies the authenticated application user/session.
- `X-API-*` headers identify and authenticate the trusted integration that sent the request.
- HTTPS/TLS remains required for transport confidentiality and normal channel security.

The signing secret must remain only in the trusted integration and the API. Do not embed it in browser JavaScript, public mobile clients, or other clients controlled by the end user.

## Headers

A signed request uses these headers:

```http
Authorization: Bearer <access-token>
X-API-Key: <integration-key-id>
X-API-Timestamp: <unix-seconds>
X-API-Nonce: <unique-request-nonce>
X-API-Signature: v1=<lowercase-hex-hmac>
```

`Authorization` is required only by routes that require JWT/session authentication. The API signature headers are required when the route has `middleware.RequireAPISignature(...)` and API signature verification is enabled.

The current route groups using this route-scoped middleware are:

```text
POST /api/v1/auth/register
POST /api/v1/auth/login
POST /api/v1/auth/refresh
POST /api/v1/auth/logout
GET  /api/v1/auth/me
GET  /api/v1/products
GET  /api/v1/products/{product_public_id}
GET  /api/v1/categories/options
GET  /api/v1/seller/products
GET  /api/v1/seller/products/{product_public_id}
POST /api/v1/seller/products
PATCH /api/v1/seller/products/{product_public_id}
GET  /api/v1/cart
PUT  /api/v1/cart/items/{product_public_id}
DELETE /api/v1/cart/items/{product_public_id}
POST /api/v1/checkout
GET  /api/v1/orders/{order_public_id}
```

The middleware is not installed globally. Other route groups must opt in explicitly.

## Configuration

Enable verification and configure one or more integration keys through the environment:

```env
APP_API_SIGNATURE_ENABLED=true
APP_API_SIGNING_KEYS=partner-prod-01:<redacted-secret>,partner-stage-01:<redacted-secret>
APP_API_SIGNATURE_ROTATION_COUNT=4
```

`APP_API_SIGNATURE_ENABLED` is a startup setting. In the repository runtime test workflow, the launcher automatically starts one API phase with signature verification enabled, restarts the API, then runs the remaining ordinary-request tests with verification disabled. It does not need to be stored in `.env.test` or `.env.test.example`; the scripts set it per phase. `APP_API_SIGNING_KEYS` and `APP_API_SIGNATURE_ROTATION_COUNT` remain environment-driven test/deployment configuration.

The key format is:

```text
<key-id>:<secret>
```

The parser splits on the first colon, so a secret may contain additional colons. Key IDs must be unique.

Defaults:

```text
APP_API_SIGNATURE_ENABLED=false
APP_API_SIGNATURE_VERSION=v1
APP_API_SIGNATURE_ROTATION_COUNT=4
APP_API_SIGNATURE_TIMESTAMP_WINDOW=5m
APP_API_SIGNATURE_NONCE_KEY_PREFIX=api-signature:nonce:
APP_API_SIGNATURE_MAX_BODY_BYTES=1048576
```

When verification is disabled, the route middleware is a no-op. When verification is enabled, at least one signing key must be configured.

## Signing algorithm

### 1. Preserve the request values

Use the exact values that will be sent on the wire:

- HTTP method as used by the request, such as `POST` or `GET`;
- URL escaped path, such as `/api/v1/catalog/items`;
- raw query string, including its exact ordering and encoding;
- Unix timestamp in seconds;
- nonce;
- raw request body bytes before JSON decoding or re-encoding.

The signed path is the escaped path plus `?` and the raw query when a query exists:

```text
/api/v1/catalog/items?cursor=abc%2F123&limit=20
```

Do not parse and reserialize the query before signing. The signer and verifier must use the same exact path/query representation.

### 2. Hash the raw body

Calculate SHA-256 over the exact body bytes sent in the HTTP request:

```text
body_hash = hex_lowercase(SHA256(raw_body_bytes))
```

For a request without a body, hash the empty byte sequence.

Whitespace, field ordering, escaping, and trailing newlines are part of the body. Signing decoded and re-encoded JSON can produce a different hash.

### 3. Select the canonical rotation

Calculate:

```text
rotation = unix_timestamp % rotation_count
```

`rotation_count` must be between 1 and 5. The signer and verifier must use the same configured value. The verifier rotates the five canonical fields left by the selected rotation index; therefore a count of `N` accepts rotation indexes `0` through `N-1`.

With the default `rotation_count=4`, the canonical field orders are:

```text
rotation 0:
METHOD|PATH_WITH_QUERY|TIMESTAMP|NONCE|BODY_HASH

rotation 1:
PATH_WITH_QUERY|TIMESTAMP|NONCE|BODY_HASH|METHOD

rotation 2:
TIMESTAMP|NONCE|BODY_HASH|METHOD|PATH_WITH_QUERY

rotation 3:
NONCE|BODY_HASH|METHOD|PATH_WITH_QUERY|TIMESTAMP
```

Example for rotation `0`:

```text
POST|/api/v1/auth/login|1792584000|01JABCXYZ|<sha256-of-raw-body>
```

The rotation count is configuration-driven. The deployed signer and verifier must use the same value.

### 4. Generate the HMAC

Calculate HMAC-SHA256 using the integration secret:

```text
digest = HMAC-SHA256(integration_secret, canonical_string)
encoded = lowercase hexadecimal digest
header = v1= + encoded
```

Example header shape:

```http
X-API-Signature: v1=<lowercase-hex-hmac>
```

The `v1` prefix identifies the signature format/version. It is different from the `X-API-Key` value, which identifies the integration and selects the secret.

## Reference pseudocode

```text
body_hash = SHA256_HEX(raw_body)
timestamp = unix_seconds()
nonce = generate_unique_nonce()
rotation = timestamp % rotation_count

fields = [
  METHOD,
  PATH_WITH_QUERY,
  timestamp_as_string,
  nonce,
  body_hash,
]

canonical = join_with_pipe(rotate_left(fields, rotation))
digest = HMAC_SHA256(secret_for(X_API_KEY), canonical)

headers = {
  "X-API-Key": X_API_KEY,
  "X-API-Timestamp": timestamp_as_string,
  "X-API-Nonce": nonce,
  "X-API-Signature": "v1=" + LOWERCASE_HEX(digest),
}
```

Generate the signature after the final URL and body have been prepared, then send the same URL and body without modification.

## Verification behavior

The API verifies, in effect:

1. the key ID exists;
2. timestamp is valid and within the configured window, currently ±5 minutes by default;
3. the request body can be read within the configured signed-body limit;
4. signature version and hexadecimal digest format are valid;
5. HMAC matches using a constant-time comparison;
6. the nonce has not already been accepted for that key ID.

Accepted nonces are stored in Redis under a namespaced key with this shape:

```text
api-signature:nonce:<key-id>:<nonce>
```

The nonce is accepted atomically with Redis `SETNX`. A repeated nonce is rejected as a replay. Redis availability is therefore required for enabled API signature routes.

Validation failures use a generic API signature error and do not reveal whether the key, timestamp, nonce, body hash, or HMAC component failed.

## Example request flow

For a login request, first serialize the exact body:

```json
{"email":"user@example.com","password":"<redacted-password>"}
```

Then calculate its body hash, build the canonical string using the request path and timestamp, and send:

```http
POST /api/v1/auth/login HTTP/1.1
Content-Type: application/json
Authorization: Bearer <access-token-if-required>
X-API-Key: partner-prod-01
X-API-Timestamp: 1792584000
X-API-Nonce: 01JABCXYZ
X-API-Signature: v1=<lowercase-hex-hmac>

{"email":"user@example.com","password":"<redacted-password>"}
```

The example values are placeholders. Never copy real credentials, tokens, or signing secrets into source control or documentation.

## Key management

- Use a different secret per integration and environment.
- Treat the key ID as an identifier, not as a secret.
- Store secrets in the deployment secret manager or protected environment configuration.
- Rotate by adding a new key ID and secret, updating the integration, then removing the old key after the migration window.
- Never log `APP_API_SIGNING_KEYS`, raw authorization headers, or complete signed requests containing credentials.
- Keep nonce values unique per request and do not reuse them after a failed or successful submission when retrying.

The configured canonical rotations are format rotations, not secret-key rotation. The default count is four, but deployments may use any supported count from 1 through 5. HMAC security depends on the confidentiality of the integration secret, while timestamp and nonce checks provide freshness and replay protection.

## API signature versus payment webhook signature

Payment webhooks intentionally use a separate contract and must not be signed with this API signature scheme:

```http
X-Payment-Timestamp: <unix-seconds>
X-Payment-Signature: <payment-signature>
```

Do not replace those headers with `X-API-Key` or `X-API-Signature`.

## Development demo signers

When `APP_ENV=development`, the API exposes two unauthenticated helper endpoints for local demonstrations:

```text
POST /api/v1/dev/signatures/api
POST /api/v1/dev/signatures/webhook
```

These routes are not registered when `APP_ENV=production` or `APP_ENV=test`. They generate headers from the configured development secrets but never return the secrets themselves. Do not expose a development deployment to an untrusted network.

API signer request:

```json
{
  "key_id": "<configured-key-id>",
  "method": "POST",
  "path": "/api/v1/checkout",
  "body": "{\"amount\":1000}",
  "nonce": "demo-unique-nonce"
}
```

The `body` value must be the exact wire body that will be sent to the signed API route. The response includes `X-API-Key`, `X-API-Timestamp`, `X-API-Nonce`, and `X-API-Signature` headers plus the exact body string.

Webhook signer request:

```json
{
  "body": "{\"event_id\":\"provider-event-123\",\"event_type\":\"payment.paid\"}"
}
```

The response includes `X-Payment-Timestamp` and `X-Payment-Signature`. Copy the returned body without changing whitespace or field ordering before sending it to the webhook endpoint.


A trusted integration implementation should test at least:

- valid signature is accepted;
- changed body is rejected;
- changed path or query is rejected;
- missing or unknown key ID is rejected;
- stale timestamp is rejected;
- malformed signature/version is rejected;
- reused nonce is rejected;
- a fresh nonce permits a retry with the same body;
- the raw body used for signing is exactly the body sent on the wire.

Repository verification includes middleware unit tests plus PostgreSQL/Redis E2E, integration, and runtime coverage for valid requests, body tampering, and nonce replay.
