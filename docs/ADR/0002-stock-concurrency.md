# ADR-0002: Atomic Stock Reservation During Checkout

- **Status:** Accepted for marketplace implementation
- **Date:** 2026-10-20
- **Scope:** Checkout and inventory mutation

## Context

Two buyers may attempt to check out the last units of the same product at the same time. A read-then-write implementation can let both requests observe the same stock and oversell. Checkout also needs an idempotent retry boundary and an auditable stock history.

## Options considered

1. Read stock, calculate in application code, then update without a lock.
2. Pessimistic row locking with `SELECT ... FOR UPDATE` inside the checkout transaction.
3. Optimistic version column and retry loops.
4. Conditional atomic update such as `stock_quantity = stock_quantity - quantity WHERE stock_quantity >= quantity`.

## Decision

Use a PostgreSQL transaction with a conditional stock decrement as the primary invariant, and keep the affected product row locked for the remainder of the transaction where the order snapshot and stock ledger are written.

The checkout flow must:

1. Claim or create the buyer's checkout idempotency record under a unique `(buyer_id, idempotency_key)` constraint.
2. Validate that a repeated key has the same request hash; replay the committed result for the same hash and reject a different hash with a conflict.
3. Lock/conditionally decrement each product in a deterministic product-ID order.
4. Reject the transaction if any decrement affects zero rows.
5. Create the order and immutable order-item snapshots in the same transaction.
6. Insert matching `stock_movements` rows with `stock_before`, `stock_after`, and signed `quantity_delta` in the same transaction.
7. Commit all state together or roll back all state together.

The exact SQL shape may use `UPDATE ... WHERE stock_quantity >= $quantity RETURNING ...`; the acceptance criterion is that no concurrent checkout can commit a negative or oversold balance.

## Consequences

### Positive

- The database enforces the critical stock invariant under concurrency.
- Stock ledger rows cannot exist without the corresponding committed stock mutation when written in the same transaction.
- Deterministic product ordering reduces deadlock risk for multi-item carts.
- The idempotency record protects retries and concurrent duplicate requests.

### Negative

- Transactions may wait on product rows during contention.
- A multi-product checkout can fail because one item is unavailable even when other items remain available.
- Deadlock/retry handling and concurrency tests are required.
- `products.stock_quantity` remains the current balance; `stock_movements` is an append-only ledger, not a replacement source of truth.

## Required verification

- Concurrent checkout test for the last available unit: covered by
  `TestCheckoutConcurrencyAgainstPostgres`.
- Idempotency test with sequential retry: covered by the existing checkout E2E
  and PostgreSQL integration tests.
- Idempotency test with two identical requests arriving concurrently: covered
  by `TestCheckoutConcurrencyAgainstPostgres`.
- Rollback test proving a failed order does not leave a stock or ledger
  mutation, followed by a successful retry with the same key: covered by
  `TestCheckoutConcurrencyAgainstPostgres`.

## Evidence/design references

- `docs/schema.dbml`: `products`, `orders`, `order_items`, `checkout_idempotency_keys`, and `stock_movements`.
- `docs/technical-assessment.md`: stock race-condition and checkout idempotency requirements.

The initial marketplace checkout code is implemented in `internal/modules/checkout`. SQLite E2E, PostgreSQL integration, and real-binary runtime tests prove sequential checkout, idempotent replay, stock reservation, payment creation, and stock-ledger writes. `TestCheckoutConcurrencyAgainstPostgres` additionally proves last-stock contention, concurrent same-key replay, and rollback/retry behavior without duplicate order, payment, or stock-movement rows.
