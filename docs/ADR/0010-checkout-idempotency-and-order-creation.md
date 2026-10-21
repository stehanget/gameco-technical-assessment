# ADR-0010: Checkout Idempotency and Order Creation Boundary

- **Status:** Accepted and implemented for initial checkout slice
- **Date:** 2026-10-21
- **Scope:** Buyer checkout, order creation, stock reservation, and idempotent retries

## Context

The assessment requires `POST /v1/checkout` to accept an `Idempotency-Key` header. A retry with the same key must return the original result instead of creating a second order. Checkout also crosses several transactional boundaries: cart validation, self-purchase validation, stock reservation, order snapshots, payment initialization, and idempotency state.

The repository exposes business APIs below `/api/v1`, so the runtime endpoint is:

```text
POST /api/v1/checkout
Idempotency-Key: <client-generated-key>
```

The checkout request does not accept product IDs or quantities. It operates on the authenticated buyer's active cart so the server remains the only source of checkout quantities.

## Decision

Implement checkout as one authenticated buyer transaction protected by a PostgreSQL-backed idempotency record. The idempotency scope is the authenticated buyer and key pair, matching the unique constraint on `checkout_idempotency_keys (buyer_id, idempotency_key)`.

### Request contract

- Authentication is required.
- The `Idempotency-Key` header is required and must be non-empty.
- The key is treated as opaque input and is length-bounded by the API contract.
- The request body is empty for the initial checkout slice.
- The request fingerprint includes the API operation/version and canonical request body representation.
- The server derives the buyer from the authenticated actor; the request cannot provide `buyer_id` or `cart_id`.

### Idempotency behavior

| Condition | Behavior |
| --- | --- |
| New buyer/key pair | Claim the key and execute checkout. |
| Same buyer/key with the same request hash after success | Return the original order/payment result; do not create a second order or reserve stock again. |
| Same buyer/key with a different request hash | Return `409 Conflict` with `checkout.idempotency_conflict`; do not mutate checkout state. |
| Identical requests arriving concurrently | Serialize on the unique buyer/key record. One request creates the order; the other waits for the committed result and replays it. |
| First request rolls back before completion | Its idempotency claim rolls back with the transaction, allowing a later retry to execute normally. |

A replay returns the same order public ID, payment reference, status, totals, and HTTP response status as the committed original result. The implementation may reconstruct the response from the persisted order/payment records; it must not create a new order during replay.

### Checkout transaction order

The checkout transaction must:

1. Claim or lock the `(buyer_id, idempotency_key)` record and compare the request hash.
2. Read and lock the buyer's active cart.
3. Reject an empty cart or a cart containing retained unavailable items; no partial checkout is allowed.
4. Lock all referenced products in deterministic internal product-ID order.
5. Revalidate active/non-deleted status, current stock, and `seller_id != buyer_id` for every item.
6. Reserve/decrement stock conditionally so no concurrent checkout can oversell.
7. Create the order and immutable order-item snapshots using the locked current prices.
8. Create the initial payment state and durable payment reference required by the checkout response.
9. Insert matching `stock_movements` rows with before/after balances and signed quantity deltas.
10. Persist the idempotency result linkage (`order_id` and response status).
11. Commit all state together or roll back all state together.

The transaction must not send external payment-provider requests before the database commit. The initial payment implementation is a local/simulated payment boundary; provider webhook handling remains a later slice.

### Response and error definitions

A successful first request and a successful identical retry return the same order/payment result. Stable checkout errors include:

```go
CheckoutIdempotencyKeyRequired = faults.Definition{
    Code:    "checkout.idempotency_key_required",
    Kind:    faults.KindInvalidInput,
    Message: "idempotency key is required",
}

CheckoutIdempotencyConflict = faults.Definition{
    Code:    "checkout.idempotency_conflict",
    Kind:    faults.KindConflict,
    Message: "idempotency key was used with a different request",
}

CheckoutCartUnavailable = faults.Definition{
    Code:    "checkout.cart_unavailable",
    Kind:    faults.KindConflict,
    Message: "cart contains unavailable items",
}

CheckoutEmptyCart = faults.Definition{
    Code:    "checkout.empty_cart",
    Kind:    faults.KindConflict,
    Message: "cart is empty",
}
```

The shared `commerce.self_purchase_not_allowed` definition is reused when checkout revalidation detects a buyer-owned product. Stock and product errors must use checkout-specific definitions when the checkout slice is implemented, rather than leaking repository errors or reusing cart mutation codes without review.

## Alternatives considered

1. **Do not require idempotency** — rejected because client retries, network timeouts, and concurrent duplicate requests could create duplicate orders or reserve stock twice.
2. **Use a process-local or Redis-only idempotency map** — rejected because it fails across API instances and cannot atomically protect the PostgreSQL order/stock transaction.
3. **Accept product IDs and quantities in the checkout body** — rejected because it creates a second mutable source of cart state and allows the client to bypass the active cart contract.
4. **Return a conflict for every repeated key** — rejected because the assessment requires same-key retries to return the original result.
5. **Allow different request hashes under the same key** — rejected because a key must identify one logical operation and changing the payload would make retry semantics ambiguous.
6. **Send the payment-provider request inside the database transaction** — rejected because external side effects cannot be rolled back safely; the initial slice uses a durable local payment reference and defers provider interaction/webhooks.

## Consequences

### Positive

- Network retries converge on one durable order result.
- Concurrent identical checkout requests cannot create duplicate orders.
- A reused key with a different request is rejected deterministically.
- Stock, order snapshots, payment initialization, ledger rows, and idempotency linkage commit atomically.
- Checkout does not trust stale cart validation and rechecks all transaction-critical invariants.

### Negative

- A transaction may wait while another request owns the same idempotency key or product row locks.
- Failed checkout because of one unavailable item rejects the whole cart rather than partially creating an order.
- Idempotency records require retention/cleanup policy in a later operational slice.
- Payment-provider integration and webhook replay protection remain outside this initial checkout implementation.

## Required verification

- [x] Missing `Idempotency-Key` returns `400` with `checkout.idempotency_key_required`.
- [x] A first checkout creates exactly one order, one payment, and one set of stock movements.
- [x] A sequential retry with the same buyer, key, and request returns the same result and does not create another order or payment.
- [x] Two identical requests with the same buyer/key arriving concurrently create one order and both receive the same result.
- [x] Reusing a key with a different request hash returns `409` with `checkout.idempotency_conflict`.
- [x] Checkout creates the initial local payment state and durable payment reference in the same transaction.
- [x] A failed checkout must roll back stock, order, payment, stock ledger, and idempotency state.
- [x] Checkout must reject unavailable cart items without partially reserving stock.
- [x] Checkout must reject buyer-owned products even if an invalid cart state is inserted directly.
- [x] Concurrent checkout requests for the last stock unit cannot oversell.
- [x] Buyer order detail reads are scoped to the authenticated buyer through `GET /api/v1/orders/{order_public_id}`.
- [x] PostgreSQL integration and real-binary runtime tests prove the sequential transaction behavior.

## Explicitly deferred

- external payment-provider calls;
- HMAC-signed payment webhooks and provider event replay handling;
- payment retry/expiry workers;
- order cancellation and stock restoration;
- idempotency-record retention and cleanup worker;
- transactional outbox notifications.

## References

- `docs/technical-assessment.md`, sections 2.1.D and 3.3
- `docs/ADR/0002-stock-concurrency.md`
- `docs/ADR/0009-buyer-cart-lifecycle.md`
- `migrations/20261021092000_create_orders_and_checkout.sql`
- `docs/schema.dbml`
