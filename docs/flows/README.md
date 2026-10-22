# Runtime Flow Documentation

This directory contains Mermaid-based runtime flow documents for the marketplace backend.

## Available flows

1. [Checkout request to stock reservation](checkout-stock-reservation.md) — traces `POST /api/v1/checkout` from HTTP middleware through idempotency, product locking, stock decrement, order/payment creation, stock ledger, audit writes, and commit/rollback behavior.
2. [Payment webhook state transition](payment-webhook-state-transition.md) — traces signed provider callbacks through replay protection, payment/order transitions, stock restoration, audit writes, and commit/rollback behavior.
3. [Authentication and session lifecycle](auth-session-lifecycle.md) — traces login, failed-login handling, session creation, refresh-token rotation, reuse detection, and logout revocation.
4. [Seller product lifecycle](seller-product-lifecycle.md) — traces seller authorization, ownership-scoped product create/update, validation, price audit, and the boundary between catalog stock and checkout reservation.

## How to use these documents

- Read the Mermaid sequence diagram for the normal request path.
- Read the failure flowchart for validation failures, idempotency conflicts, insufficient stock, and transaction rollback.
- Use the source-code map at the end of each document to jump from the runtime flow to the implementation.
- Keep flow documents focused on runtime behavior; record architectural decisions in `docs/ADR/` instead.
