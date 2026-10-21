# ADR-0009: Buyer Cart Lifecycle and Mutation Contract

- **Status:** Accepted for the cart slice
- **Date:** 2026-10-21
- **Scope:** Buyer cart lifecycle before checkout

## Context

The marketplace schema already contains `carts` and `cart_items`, but schema presence is not runtime contract approval. The cart API must define ownership, lifecycle, mutation semantics, stock behavior, and the boundary between cart management and checkout before implementation is considered complete.

The first delivery slice must remain small enough to verify independently. Checkout has different transactional requirements: idempotency, order snapshots, atomic stock reservation, payment state, and rollback behavior. Those concerns must not be hidden inside ordinary cart mutations.

## Decision

Implement buyer cart lifecycle as a separate vertical slice. Checkout is a subsequent slice and is not part of the cart implementation.

### Ownership and lifecycle

- Every cart operation is scoped to the authenticated buyer actor.
- A buyer has at most one `active` cart, enforced by the existing partial unique index.
- Historical carts use `checked_out` or `abandoned` status.
- The API does not accept `buyer_id` as input.
- `carts.public_id` is an application-generated UUIDv7 response identifier.
- Product references use `product_public_id`; internal numeric IDs remain persistence-only.

### HTTP surface

```text
GET    /api/v1/cart
PUT    /api/v1/cart/items/{product_public_id}
DELETE /api/v1/cart/items/{product_public_id}
```

`GET /cart` returns the authenticated buyer's active cart. If none exists, the service creates an empty active cart and returns `items: []`.

`PUT /cart/items/{product_public_id}` uses replace semantics:

```json
{"quantity": 2}
```

The quantity replaces the existing quantity for that product. It is not a delta operation. The request must contain an integer quantity in the inclusive range `1..999`.

`DELETE /cart/items/{product_public_id}` removes the product from the authenticated buyer's active cart. Repeating the delete is safe and returns the resulting cart state.

### Product and stock rules

- Only active, non-deleted products can be added.
- The service checks current stock during add/update.
- Cart mutation does not reserve or decrement stock.
- A quantity greater than current stock is rejected with a conflict response.
- Checkout owns atomic stock reservation/decrement according to ADR-0002.
- Cart response prices are read from the current product price; checkout must later create immutable order-item price snapshots.

### Response contract

The cart response includes:

- cart public ID;
- cart status;
- `items`, containing currently available items;
- `unavailable_items`, containing retained cart items that cannot currently be purchased;
- item product public ID and display name;
- current unit price;
- requested quantity;
- calculated line subtotal;
- current stock quantity;
- `availability` and nullable `unavailable_reason` (`product_inactive`, `product_deleted`, or `insufficient_stock`);
- calculated total for available items only;
- created and updated timestamps.

The response always serializes both `items` and `unavailable_items` as arrays, including `[]` when empty. Unavailable items remain in the cart and can be removed explicitly. If a product becomes active again and has sufficient stock, it returns to `items` automatically.

## Alternatives considered

1. **Combine cart and checkout in one endpoint** — rejected because checkout requires idempotency, stock concurrency, order snapshots, payment state, and rollback semantics that are a separate transaction boundary.
2. **Use quantity deltas** — rejected because retries and duplicate client requests are harder to reason about. Replace semantics are deterministic and naturally retry-safe for the cart state.
3. **Reserve stock when adding to cart** — rejected because abandoned carts would hold inventory and require expiration/release workers. Reservation belongs to checkout.
4. **Accept buyer IDs or cart IDs as authorization input** — rejected because ownership must come from the authenticated actor and active-cart scope.
5. **Allow quantities above current stock and defer all validation to checkout** — rejected because it creates a misleading cart state and poor feedback, while still not reserving inventory.

## Consequences

### Positive

- Cart operations have a clear actor and ownership boundary.
- PUT retries converge on the same quantity.
- Cart state does not reserve inventory or require expiration logic.
- Checkout can implement its own idempotent atomic transaction without inheriting cart mutation behavior.
- The API contract can be tested independently with SQLite E2E, PostgreSQL integration, and runtime binary tests.

### Negative

- Stock can change after cart mutation, so checkout may still reject an item that was valid when added.
- Current cart prices can change before checkout; checkout must snapshot the authoritative price inside its transaction.
- `GET /cart` may create a row for an otherwise empty cart.
- Manual cart cleanup/abandonment policy remains a later operational concern.

## Required verification

- Unauthenticated cart request returns `401`.
- Authenticated buyer can add, read, replace, and remove an item.
- A buyer cannot affect another buyer's active cart.
- Inactive or soft-deleted products cannot be added.
- Quantity validation rejects zero, negative, and above-stock values.
- Cart mutation does not change `products.stock_quantity`.
- Empty cart returns `items: []`.
- PostgreSQL and runtime tests cover the lifecycle against the real migration and binary API.

## Explicitly deferred

- checkout endpoint and checkout idempotency key;
- order and order-item creation;
- atomic stock reservation and stock movement ledger writes;
- payment creation and webhook processing;
- cart expiration/abandonment worker;
- manual cart merge across devices.

## References

- `migrations/20261021091000_create_carts.sql`
- `docs/schema.dbml`
- `docs/ADR/0002-stock-concurrency.md`
