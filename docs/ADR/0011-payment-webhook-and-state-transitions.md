# ADR-0011: Payment Webhook and State Transitions

- **Status:** Accepted and implemented for the current webhook slice
- **Date:** 2026-10-21
- **Scope:** Simulated payment-provider webhook verification, idempotent event processing, and payment/order state transitions

## Context

The initial checkout slice creates a local `payments` row with `provider = simulated`, `status = pending`, a durable payment reference, and an order in `pending_payment`. The assessment requires a simulated payment callback at:

```text
POST /api/v1/webhooks/payment
```

The callback is an untrusted provider-to-API boundary. It must verify an HMAC signature with constant-time comparison, reject replayed or stale callbacks, and process provider retries idempotently.

Payment-provider calls are not part of this slice. The API only consumes simulated provider callbacks after checkout has committed.

## Decision

Implement one public webhook endpoint owned by the checkout/payment module:

```text
POST /api/v1/webhooks/payment
Content-Type: application/json
X-Payment-Timestamp: <unix-seconds>
X-Payment-Signature: <hex-hmac-sha256>
```

The endpoint does not use buyer Bearer authentication. Trust is established by the configured webhook secret and the signed request body.

### Signature contract

The signed message is:

```text
<timestamp>.<raw-request-body>
```

The signature is:

```text
hex(HMAC-SHA256(webhook_secret, signed_message))
```

Verification rules:

1. Read and preserve the exact raw request body before JSON decoding.
2. Require a valid numeric timestamp in Unix seconds.
3. Reject timestamps outside a five-minute freshness window using the injected application clock.
4. Compute the expected HMAC with the configured webhook secret.
5. Compare the supplied and expected signatures with `crypto/subtle.ConstantTimeCompare`.
6. Reject malformed, missing, stale, or mismatched signatures without querying payment/order state.
7. Never log the webhook secret, signature, raw authorization material, or full sensitive payload.

The webhook secret is configuration-backed and must be supplied through the repository's normal secret configuration path. It must not be hardcoded, seeded, committed, or returned by any API.

The implemented slice covers configuration loading, clock-injected signature verification, rejection of missing, malformed, stale, or mismatched authentication headers, bounded event decoding, durable event idempotency, payment/order transitions, expiry-triggered cancellation and stock restoration, and processed-event retention cleanup.

### Request contract

The initial simulated payload is:

```json
{
  "event_id": "provider-event-123",
  "event_type": "payment.paid",
  "payment_reference": "pay_<uuid>",
  "amount": 199900,
  "occurred_at": "2026-10-21T12:00:00Z"
}
```

Rules:

- `event_id` is required and bounded to 128 characters.
- `event_type` is one of `payment.paid`, `payment.failed`, or `payment.expired`.
- `payment_reference` is required and resolves the local payment row.
- `amount` is an integer minor-unit amount and must equal `payments.amount`.
- `occurred_at` is required, timezone-aware, and must be a valid timestamp.
- The provider identity is derived from the configured webhook route/provider boundary; it is not accepted as an arbitrary authorization scope from the body.
- Unknown fields are ignored only if the JSON decoder remains bounded; malformed JSON is rejected.

### Event idempotency

Persist one provider event record with a unique provider/event identity:

```text
UNIQUE(provider, event_id)
```

The event record stores bounded metadata sufficient to prove replay behavior:

- provider;
- event ID;
- event type;
- payment reference;
- payload hash;
- processing status;
- processed timestamp;
- created timestamp.

Do not persist the raw signature or webhook secret. Persisting the full raw payload is not required for the initial slice.

Behavior:

| Condition | Behavior |
| --- | --- |
| New valid event | Process payment/order transition transactionally and mark the event processed. |
| Same provider/event ID with the same payload hash | Return the original successful result; do not transition state again. |
| Same provider/event ID with a different payload hash | Return `409 Conflict`; do not mutate payment or order state. |
| Unknown event ID but unknown payment reference | Return a stable not-found/invalid contract; do not create payment or order rows. |
| Valid event with amount mismatch | Return `409 Conflict`; do not mutate payment or order state. |
| Concurrent duplicate delivery | Serialize on the unique event record; exactly one transition may commit. |
| Failed transaction before completion | Roll back the event claim and payment/order transition so a later retry can process normally. |

### State transitions

For the initial implementation:

```text
payment.pending → payment.paid
order.pending_payment → order.paid

payment.pending → payment.failed
order.pending_payment → order.payment_failed

payment.pending → payment.expired
order.pending_payment → order.cancelled
```

A paid, failed, or expired payment is terminal for this slice. Out-of-order transitions that would regress a terminal state are rejected or treated as a safe no-op according to the stable event result; they must never move `paid` back to `pending` or `failed`.

The webhook transaction updates the payment, order, provider-event record, and expiry-triggered stock restoration together. It does not call an external provider, publish a notification, or create an audit event. Those remain separate follow-up slices.

### HTTP error contract

The implementation will use centralized typed definitions for:

- missing or malformed signature/timestamp;
- stale webhook timestamp;
- malformed payload;
- unknown payment reference;
- invalid event type;
- amount mismatch;
- event idempotency conflict.

The handler must return the shared safe error envelope and must not expose database errors, HMAC material, or raw provider payloads.

## Alternatives considered

1. **Bearer-authenticated webhook** — rejected because provider callbacks do not have a buyer session and must use a provider trust boundary.
2. **Compare signatures with `==`** — rejected because the assessment explicitly requires constant-time comparison.
3. **Timestamp-only replay protection** — rejected because provider retries are legitimate and event identity must provide durable idempotency.
4. **Redis-only event deduplication** — rejected because event idempotency must survive process restarts and remain atomic with PostgreSQL state transitions.
5. **Store raw signatures/secrets for debugging** — rejected because it increases credential and replay exposure without improving transition correctness.
6. **Restore stock for every payment failure in the webhook** — rejected for this slice; only `payment.expired` cancels the order and restores the checkout reservation.

## Consequences

### Positive

- Provider callbacks have an explicit cryptographic trust boundary.
- Replay and concurrent duplicate delivery converge on one durable state transition.
- Payment and order state cannot diverge from the processed provider event.
- Amount/reference validation prevents a validly signed event from mutating the wrong payment state.
- The webhook remains independent from buyer authentication and checkout idempotency.

### Negative

- A webhook request may wait on the provider-event or payment row lock.
- Processed provider events are retained for the configured webhook replay window and removed by the maintenance worker; pending/unprocessed event rows are preserved.
- Payment expiry scheduling/retry is not implemented, so cancellation/restoration currently occurs when a valid `payment.expired` webhook arrives.
- External provider calls, retry workers, audit events, and notifications remain deferred.

## Required verification

- [x] Missing signature is rejected.
- [x] Invalid signature is rejected with constant-time verification in the implementation.
- [x] Stale timestamp is rejected.
- [x] Malformed JSON and invalid event type are rejected safely.
- [x] Unknown payment reference is rejected without creating rows.
- [x] Amount mismatch is rejected without changing payment/order state.
- [x] A valid `payment.paid` event transitions payment and order once.
- [x] Duplicate delivery with the same event ID and payload hash replays safely.
- [x] Same event ID with a different payload hash returns a conflict.
- [x] Concurrent duplicate delivery creates one transition and one processed event in PostgreSQL integration coverage.
- [x] A failed transaction rolls back event claim and payment/order state in PostgreSQL integration coverage.
- [x] PostgreSQL/Redis E2E, PostgreSQL integration, and runtime smoke cover the authenticated order state after a successful webhook.
- [x] An expired payment transitions the order to `cancelled` and restores reserved stock exactly once.
- [x] Processed webhook events older than the configured retention cutoff are deleted by the cleanup repository; worker scheduling uses the same path.

## Explicitly deferred

- outbound payment-provider API calls;
- payment expiry scheduler and retry worker;
- refunds and partial refunds;
- transactional audit-event producer;
- notification/outbox publication;

Webhook event retention defaults to seven days through `APP_PAYMENT_WEBHOOK_EVENT_RETENTION`; its cleanup cadence is controlled by `APP_PAYMENT_WEBHOOK_CLEANUP_INTERVAL`.

## References

- `docs/technical-assessment.md`, sections 2.1.D, 3.3, and 3.4
- `docs/ADR/0010-checkout-idempotency-and-order-creation.md`
- `docs/ADR/0002-stock-concurrency.md`
- `migrations/20261021093000_create_payments.sql`
- `docs/schema.dbml`
