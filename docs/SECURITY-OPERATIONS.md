# Security Operations

This document defines the operational security procedures for the current backend. It describes the supported application-level key overlap and the deployment responsibilities that remain outside the Go process.

## Secret ownership

| Secret | Runtime use | Current support | Rotation owner |
| --- | --- | --- | --- |
| `APP_JWT_SECRET` | Signs new access tokens | Current key plus bounded previous-key verification | Deployment/secret-manager operator |
| `APP_JWT_PREVIOUS_SECRETS` | Verifies tokens issued before the latest JWT rotation | Up to three previous keys, newest first | Deployment/secret-manager operator |
| `APP_PAYMENT_WEBHOOK_SECRET` | Verifies payment-provider callbacks | One active secret | Deployment/payment-provider operator |
| `APP_API_SIGNING_KEYS` | Verifies trusted integration requests | Key-ID based configured key set | Deployment/integration operator |
| PostgreSQL/Redis credentials | Infrastructure access | Environment/configuration supplied | Deployment/secret-manager operator |

Secrets must come from the deployment secret store or an equivalent protected runtime injection mechanism. Never commit them to Git, `.env.example`, Docker images, logs, audit metadata, or test artifacts.

## JWT rotation with overlap

The application supports a bounded verification overlap:

1. Generate a new random JWT secret outside the repository.
2. Move the current `APP_JWT_SECRET` into the first position of `APP_JWT_PREVIOUS_SECRETS`.
3. Set the new secret as `APP_JWT_SECRET`.
4. Deploy the new configuration and verify login, refresh, and authenticated requests.
5. Keep the old key in the previous-key list for at least the maximum access-token lifetime plus an agreed clock-skew margin.
6. Remove keys older than the supported overlap window in a later deployment.

The application always signs with `APP_JWT_SECRET`. Previous keys are verification-only and are bounded to three entries. `APP_JWT_PREVIOUS_SECRETS` must not contain duplicates, empty entries, or the current key; configuration loading rejects those cases.

### Emergency JWT compromise response

If the current JWT key is suspected to be compromised:

1. Replace `APP_JWT_SECRET` immediately with a newly generated secret.
2. Do not retain the compromised key in `APP_JWT_PREVIOUS_SECRETS`.
3. Revoke all active database sessions using the operational session-revocation procedure or a controlled database operation.
4. Flush only the related positive session-cache entries if required; do not treat Redis as the source of truth.
5. Redeploy the API and worker with the new secret.
6. Check login, refresh rejection for old sessions, error rates, and security logs.
7. Preserve the incident timeline and rotate dependent credentials if the exposure path may include deployment infrastructure.

This application does not implement a remote secret-manager API or automatic rotation scheduler. The deployment system remains responsible for generating, storing, distributing, and revoking secrets.

## Payment webhook secret rotation

The payment webhook currently has one active `APP_PAYMENT_WEBHOOK_SECRET`. A provider rotation must therefore be coordinated as a deployment/provider change:

1. Confirm whether the provider supports an overlap window or dual-signing delivery.
2. If the provider supports overlap, configure the provider and application rollout so callbacks signed with the accepted secret are not rejected during the transition.
3. If the provider does not support overlap, schedule the provider and application change together and monitor rejected-signature events during the cutover.
4. Generate and inject the replacement secret through the deployment secret store.
5. Deploy the API, send a provider test callback, and verify the expected `204` response and persisted event state.
6. Remove the old secret from deployment configuration and provider configuration after the overlap window.

Do not place webhook secrets in the development signature-generator response or in audit metadata. The development helper must remain disabled outside `APP_ENV=development`.

## Anti-enumeration expectations

Login returns the same invalid-credentials contract for an unknown email, wrong password, and bounded account lockout. Unknown-email login performs a dummy password comparison and failed attempts are recorded without persisting the raw email in the audit aggregate ID.

Registration currently auto-issues tokens for a newly-created account and returns a distinct conflict for an existing email. That response shape is an intentional product/API contract but still permits account-existence discovery. A full registration anti-enumeration change requires a contract slice that separates account creation from authentication, such as an accepted/verification response and an out-of-band confirmation flow. Do not claim registration is enumeration-safe until that contract is implemented.

## Security event logging and audit

Security-relevant events should be represented by both:

- an append-only `audit_events` row for investigation;
- a structured application log/metric suitable for alerting.

Current auth producers include failed login, successful registration/login/refresh, logout, and refresh-token reuse. Domain producers now include cart item changes, product price changes, and order status transitions from payment webhook processing. Audit records may contain actor ID, event type, aggregate reference, IP address, user agent, and bounded reason metadata. They must never contain passwords, tokens, API keys, secrets, or raw payment payloads.

Successful domain mutations write their sanitized audit event directly to the append-only PostgreSQL `audit_events` table in the same transaction as the business mutation. This keeps audit evidence consistent with committed business state without introducing a second queue or worker for the assessment scope.

Recommended alerts:

- repeated `auth.login_failed` or `account_locked` events;
- `auth.refresh_token_reused` responses;
- payment webhook signature/timestamp failures;
- spikes in `429` responses;
- recovery/panic events;
- repeated database/Redis unavailable errors.

Auth events produced outside a business transaction are written directly to `audit_events` when the enqueue succeeds. If PostgreSQL is unavailable, the event cannot be persisted; the request path records a structured failure for operational alerting. This is an explicit assessment-scope trade-off rather than a durable asynchronous queue.

## Release security checks

Before release, run:

```text
make check
make security-check
```

The security check runs Go vulnerability and static-analysis tools against the repository. Dependency and container findings must be triaged before production deployment; a clean local run is not a substitute for image scanning and secret scanning in the CI provider.

## Deployment checklist

- [ ] Secrets are injected from a protected deployment store.
- [ ] No secret appears in Git, image layers, logs, audit metadata, or evidence files.
- [ ] JWT current/previous-key overlap is within the configured lifetime window.
- [ ] Payment-provider webhook rotation has an explicit cutover/rollback plan.
- [ ] PostgreSQL and Redis are internal-only and not publicly exposed.
- [ ] Application images are pinned to reviewed versions or digests.
- [ ] `make check` and `make security-check` passed.
- [ ] CI dependency, secret, and container scans passed or have an approved exception.
- [ ] Security alerts and log retention are configured.
