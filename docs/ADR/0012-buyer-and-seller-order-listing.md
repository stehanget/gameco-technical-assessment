# ADR-0012: Buyer and Seller Order Listing

- **Status:** Accepted and implemented
- **Date:** 2026-10-22
- **Scope:** Authenticated buyer order history and seller-scoped order listing

## Context

Checkout already exposes buyer order detail, while seller order listing was implemented before its architecture decision was recorded. The marketplace needs explicit list contracts for both sides of the order lifecycle without leaking cross-user or cross-seller data.

The runtime endpoints are:

```text
GET /api/v1/orders
GET /api/v1/orders/{order_public_id}
GET /api/v1/seller/orders
GET /api/v1/seller/orders/{order_public_id}
```

## Decision

Use cursor-based keyset pagination for both list endpoints. Both endpoints accept:

- `q` — case-insensitive search over order public ID or immutable product-name snapshots;
- `status` — order status filter;
- `payment_status` — payment status filter;
- `sort_by` — endpoint-specific stable sort field;
- `sort_order` — `asc` or `desc`;
- `cursor` — opaque cursor returned by the previous page;
- `per_page` — default `20`, maximum `100`.

Cursors contain a version, selected sort, direction, named effective-filter fingerprint, primary sort value, and public-ID tie-breaker. A cursor is rejected when the effective search/filter/sort parameters differ from the parameters used to create it.

### Buyer list

`GET /api/v1/orders` requires buyer authentication and the buyer role. Repository visibility is enforced with:

```sql
orders.buyer_id = authenticated_user_id
```

Supported buyer sort fields are `created_at` and `total_amount`. Each result returns the buyer's complete immutable order projection:

- order public ID;
- order status;
- full order total;
- payment status;
- all immutable item snapshots;
- creation timestamp.

The endpoint returns an empty JSON array for no matches and never reveals whether a foreign order exists.

### Seller list

`GET /api/v1/seller/orders` requires seller authentication, seller role, and `seller.order.read`. Repository visibility is enforced through `order_items.seller_id = authenticated_user_id`.

Supported seller sort fields are `created_at` and `seller_total`. Results include only the authenticated seller's immutable item snapshots and seller subtotal; the full buyer order total and other sellers' items are not returned.

`GET /api/v1/seller/orders/{order_public_id}` uses the same seller visibility predicate as the list endpoint. A malformed, unknown, or order belonging only to another seller returns the same not-found contract. The response contains only that seller's item snapshots, seller subtotal, order status, payment status, and creation timestamp.

## Alternatives considered

1. **Buyer-only order detail** — rejected because buyers need order history without retaining every order public ID.
2. **Offset/page pagination** — rejected because keyset pagination is already the catalog contract and remains stable for changing order datasets.
3. **Application-only ownership checks** — rejected because repository predicates are the durable BOLA/IDOR boundary.
4. **One shared seller/buyer projection** — rejected because seller visibility must not expose the buyer total or other sellers' item snapshots.
5. **Internal numeric IDs in cursors** — rejected because public IDs are the stable external tie-breaker.

## Consequences

### Positive

- Buyer history and seller operations have explicit, consistent pagination contracts.
- Ownership is enforced inside repository queries.
- Seller responses cannot disclose another seller's line items or the buyer's full total.
- Opaque cursor fingerprints prevent accidental page traversal across changed filters or sorting.

### Negative

- Two role-specific projections and item queries must be maintained.
- Search over immutable product snapshots requires joins and bounded indexed/search-aware query work as order volume grows.
- This ADR does not add order cancellation, refund, or buyer order mutation APIs.

## Required verification

- [x] Buyer list route is authenticated and buyer-role protected.
- [x] Buyer repository query is scoped by authenticated buyer ID.
- [x] Buyer list returns full order/item snapshots and payment status.
- [x] Seller repository query is scoped by order-item seller ID.
- [x] Seller list returns only seller-owned items and seller subtotal.
- [x] Seller detail returns only seller-owned items and seller subtotal.
- [x] Seller detail returns not-found for malformed, unknown, or foreign-seller orders.
- [x] Both endpoints use bounded cursor pagination and cursor fingerprint validation.
- [x] Generated Swagger documents both list contracts.
- [x] Package, PostgreSQL integration, and runtime verification pass for the implemented slice.

## References

- `docs/technical-assessment.md`
- `docs/ADR/0010-checkout-idempotency-and-order-creation.md`
- `docs/ADR/0011-payment-webhook-and-state-transitions.md`
- `internal/modules/checkout/dto.go`
- `internal/modules/checkout/repository.go`
