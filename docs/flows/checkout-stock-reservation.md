# Checkout Request to Stock Reservation

This document traces the implemented HTTP checkout flow from the incoming request to the PostgreSQL stock decrement and stock ledger write.

## Scope

The flow covers:

- `POST /api/v1/checkout`;
- authentication and route protections;
- buyer/cart/idempotency validation;
- product row locking and conditional stock decrement;
- order, payment, order-item, stock-ledger, and audit writes;
- commit and rollback behavior.

Payment-provider webhook processing is a separate flow documented in [ADR-0011](../ADR/0011-payment-webhook-and-state-transitions.md).

## Request contract

```text
POST /api/v1/checkout
Authorization: Bearer <access-token>
Idempotency-Key: <client-generated-key>
```

The request body is empty for the initial checkout slice. The server derives the buyer and active cart from the authenticated actor; the client does not provide `buyer_id`, `cart_id`, product IDs, or quantities.

## Happy-path request flow

```mermaid
sequenceDiagram
    autonumber

    participant C as Client
    participant R as Gin Router
    participant G as Global Middleware
    participant A as Checkout Middleware
    participant H as Checkout Handler
    participant S as Checkout Service
    participant DB as PostgreSQL

    C->>R: POST /api/v1/checkout<br/>Authorization + Idempotency-Key
    R->>G: Request ID, body limit, security headers, CORS, recovery, logging
    G->>A: Continue request
    A->>A: Validate JWT and active session
    A->>A: Apply authenticated rate limit
    A->>A: Validate API signature when enabled
    A->>A: Require buyer role
    A->>H: Authenticated buyer request
    H->>H: Require non-empty Idempotency-Key
    H->>S: Create(actor, idempotency key)

    S->>DB: BEGIN
    S->>DB: Lock or create buyer/key idempotency record
    S->>DB: Compare request hash
    S->>DB: Lock buyer active cart
    S->>DB: Load and lock products in product-ID order
    S->>DB: Validate status, ownership, and available quantity

    loop Each cart line
        S->>DB: Conditional stock decrement<br/>stock_quantity >= quantity
        S->>DB: Insert order_item snapshot
        S->>DB: Insert stock_movement<br/>before, after, signed delta
    end

    S->>DB: Insert order
    S->>DB: Insert pending payment
    S->>DB: Clear active cart items
    S->>DB: Link idempotency key to order
    S->>DB: Insert checkout/payment audit events
    S->>DB: COMMIT

    S-->>H: Order and payment response
    H-->>C: 201 Created
```

## Where stock actually decreases

Stock is not reserved when the buyer adds an item to the cart. It decreases during checkout inside the PostgreSQL transaction:

```sql
UPDATE products
SET stock_quantity = stock_quantity - $quantity
WHERE id = $product_id
  AND stock_quantity >= $quantity;
```

The implementation checks `RowsAffected`:

- `1` row updated: stock reservation succeeds;
- `0` rows updated: stock is insufficient or the row no longer satisfies the condition, so checkout fails and the transaction rolls back.

For each successful decrement, the same transaction writes a `stock_movements` row containing:

- product ID;
- order and order-item IDs;
- actor/buyer ID;
- movement type `checkout`;
- negative quantity delta;
- `stock_before`;
- `stock_after`;
- reason `checkout stock reservation`.

`products.stock_quantity` remains the current balance. `stock_movements` is the append-only history of inventory changes.

## Transaction boundary

The following state changes commit together:

| Operation | Storage | Transaction boundary |
| --- | --- | --- |
| Idempotency claim/linkage | `checkout_idempotency_keys` | Checkout transaction |
| Active cart lock/read | `carts` | Checkout transaction |
| Product row lock/read | `products` | Checkout transaction |
| Stock decrement | `products.stock_quantity` | Checkout transaction |
| Order creation | `orders` | Checkout transaction |
| Immutable item snapshot | `order_items` | Checkout transaction |
| Payment initialization | `payments` | Checkout transaction |
| Inventory ledger | `stock_movements` | Checkout transaction |
| Audit events | `audit_events` | Checkout transaction |
| Cart item removal | `cart_items` | Checkout transaction |

If any operation fails before commit, all of these writes roll back. This prevents an order, payment, or stock ledger from existing without the corresponding stock mutation.

## Idempotency and retry behavior

```mermaid
flowchart TD
    A[POST /api/v1/checkout] --> B{Idempotency-Key valid?}
    B -- No --> E1[400 invalid request]
    B -- Yes --> C{Existing buyer/key record?}

    C -- Same hash + completed --> E2[Replay original order/payment]
    C -- Different hash --> E3[409 idempotency conflict]
    C -- New or pending key --> D[Begin checkout transaction]

    D --> F{Active cart available?}
    F -- No --> X[Rollback] --> E4[409 empty/unavailable cart]
    F -- Yes --> G{All products valid and stock sufficient?}

    G -- No --> X --> E5[409 checkout conflict]
    G -- Yes --> H[Create order/payment/items/ledger/audit]

    H --> I{All writes succeed?}
    I -- No --> X --> E6[Rollback and return error]
    I -- Yes --> J[Commit]
    J --> E7[201 Created]
```

Concurrent requests using the same buyer and idempotency key serialize on the PostgreSQL-backed idempotency record. A successful retry replays the existing order instead of decrementing stock a second time.

Concurrent requests for the same product lock product rows in deterministic internal product-ID order and still rely on the conditional decrement to prevent overselling.

## Important boundaries

- No external payment-provider call is made inside this checkout transaction.
- Payment-provider callbacks enter through `POST /webhooks/payment` and use the separate webhook/state-transition flow.
- The audit write is direct to the final `audit_events` table; there is no audit outbox or custom metrics endpoint in the assessment scope.
- Redis is used for request/security controls, not as the source of truth for order or stock state.

## Source-code map

- Route and middleware composition: `internal/modules/checkout/route.go`
- HTTP handler: `internal/modules/checkout/handler.go`
- Checkout service: `internal/modules/checkout/service.go`
- Checkout transaction and stock mutation: `internal/modules/checkout/repository.go`
- Audit writer: `internal/platform/audit/audit.go`
- Checkout schema: `migrations/`
- Stock concurrency decision: `docs/ADR/0002-stock-concurrency.md`
- Checkout/idempotency decision: `docs/ADR/0010-checkout-idempotency-and-order-creation.md`
- Payment webhook/state transitions: `docs/ADR/0011-payment-webhook-and-state-transitions.md`
