# Seller Product Lifecycle

This document traces seller product mutations from HTTP authorization through product persistence and audit behavior.

## Product create/update flow

```mermaid
sequenceDiagram
    autonumber

    participant S as Seller Client
    participant R as Catalog Route
    participant M as Auth/Permission Middleware
    participant H as Catalog Handler
    participant V as Catalog Service
    participant DB as PostgreSQL

    S->>R: POST/PATCH /api/v1/seller/products
    R->>M: Validate JWT/session
    M->>M: Require seller role and product permission
    M->>H: Authenticated seller request
    H->>V: CreateProduct or UpdateProduct(actor, request)
    V->>DB: Resolve category and seller-scoped product
    V->>V: Validate names, price, stock, and identifiers

    alt Create product
        V->>DB: Insert product with seller_id
        V->>H: Return seller product response
    else Update product
        V->>DB: BEGIN
        V->>DB: Update seller-owned product
        V->>DB: Write product price audit when price changes
        V->>DB: COMMIT
        V->>H: Reload seller product response
    end

    H-->>S: 201 Created or 200 OK
```

## Authorization boundary

```mermaid
flowchart TD
    A[Seller product request] --> B{Authenticated session?}
    B -- No --> E1[401 Unauthorized]
    B -- Yes --> C{Seller role and permission?}
    C -- No --> E2[403 Forbidden]
    C -- Yes --> D{Product belongs to seller?}
    D -- No --> E3[404 Product not found]
    D -- Yes --> E4[Apply mutation and return response]
```

The seller ID is derived from the authenticated actor. It is not accepted as a trusted request field. Update queries include seller ownership so a seller cannot mutate another seller's product by changing a public identifier.

## Product and stock semantics

- Product creation sets the seller ownership, category, active status, price, and initial stock.
- Product update validates all resulting values before persistence.
- A price change creates a bounded `product.price_changed` audit event in the same update transaction.
- Product stock is the current catalog balance; checkout performs the transactional stock reservation.
- Cart and checkout flows must not trust seller-provided ownership fields from the request body.

## Source-code map

- Routes and seller authorization: `internal/modules/catalog/route.go`
- HTTP handlers: `internal/modules/catalog/handler.go`
- Validation and service orchestration: `internal/modules/catalog/service.go`
- Seller-scoped persistence and price audit: `internal/modules/catalog/repository.go`
- Product authorization decision: `docs/ADR/0008-seller-product-read-and-ownership-authorization.md`
- Checkout stock mutation: [Checkout request to stock reservation](checkout-stock-reservation.md)
