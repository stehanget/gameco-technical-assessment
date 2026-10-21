# ADR-0008: Seller Product Reads and Shared Category Options

- **Status:** Accepted
- **Date:** 2026-10-21
- **Scope:** Seller-owned product reads, shared category options, route authorization, and object ownership checks

## Context

The public catalog read API intentionally allows guests to browse active, non-deleted products without authentication. Seller tooling needs a separate product surface because sellers must manage and inspect their own products, including product state that is not publicly visible.

A role or permission check alone is insufficient for seller product detail and future mutations. A seller with valid seller permissions must still be prevented from reading or changing another seller's product. This is an object-level authorization boundary and directly addresses BOLA/IDOR risk.

The product-management flow also needs reusable category options for the public catalog filters and seller product forms. This is a shared catalog lookup rather than a seller-owned resource.

## Options considered

1. Reuse the public catalog routes for seller reads and infer ownership in the client.
2. Add seller routes protected only by the seller role/permission.
3. Add a separate authenticated seller product surface with permission middleware plus owner-scoped repository queries.
4. Expose category options as a hardcoded list in the seller handler.

## Decision

Provide a separate authenticated seller product surface:

```text
GET   /api/v1/seller/products
GET   /api/v1/seller/products/{product_public_id}
GET   /api/v1/categories/options
POST  /api/v1/seller/products
PATCH /api/v1/seller/products/{product_public_id}
```

The seller product list/detail routes are the seller read slice. The category options route is a shared public catalog lookup and is not protected by seller permission middleware. Create and update remain part of the seller write slice and share the seller authorization and identifier rules.

## Authorization layers

Seller product routes use both layers:

1. **Authentication, seller-role, and permission middleware** establish that the request has an authenticated seller with the required product capability.
2. **Object-level ownership checks** scope product reads and mutations to the authenticated seller.

The permission catalog remains the source of permission codes. The seller route requires the `seller` role and reuses the existing product read permission unless a later contract decision requires a distinct seller-specific capability:

```text
product.read
product.create
product.update
```

The public catalog routes remain unauthenticated and do not use product permission middleware.

A seller product lookup must be scoped to the authenticated seller at the persistence boundary, conceptually:

```sql
WHERE p.public_id = ?
  AND p.seller_id = ?
  AND p.deleted_at IS NULL
```

The seller list applies the same owner scope. It may return the seller's active and non-public product statuses so the seller can manage its catalog, but soft-deleted products remain excluded from the normal read surface. Product status is included in the seller response when needed to distinguish public visibility from seller ownership state.

A seller must receive the same safe not-found behavior for another seller's product as for an unknown product. The API must not reveal that a product exists under another owner.

## Route ownership and ordering

Register category options under the shared catalog namespace, separate from the seller product wildcard:

```go
router.GET("/categories/options", categoryHandler.Options)

sellerGroup.GET("", sellerHandler.List)
sellerGroup.GET("/:product_public_id", sellerHandler.Detail)
```

Keeping category options outside `/seller/products` makes its shared public purpose explicit and prevents clients from coupling catalog filter options to seller product ownership.

## Seller list contract

`GET /api/v1/seller/products` uses the catalog's established list conventions where applicable:

- opaque cursor pagination;
- bounded `per_page`;
- explicit `sort_by` and `sort_order` allow-lists;
- indexed, parameterized search over supported product text fields;
- public UUIDv7 identifiers in responses and cursors;
- no internal `BIGINT` IDs in the API contract;
- response timestamps use `timestamp.Timestamp`;
- response fields are explicit about required and nullable presence.

Seller-specific filtering must never weaken the owner scope. Query filters are applied in addition to, not instead of, `seller_id = authenticated_user_id`.

## Seller detail contract

`GET /api/v1/seller/products/{product_public_id}` returns the seller-owned product read model. The path parameter is explicitly named `product_public_id` so future nested routes remain unambiguous:

```text
/products/{product_public_id}/comments/{comment_public_id}
```

The response may include seller-management fields such as status and stock that are not exposed by the public catalog, but it must continue to exclude internal database identifiers and sensitive seller/account fields.

## Shared category options contract

`GET /api/v1/categories/options` reads the canonical category catalog for public catalog filters and seller product forms. It must not maintain a second hardcoded seller category list.

The endpoint is public because the public catalog is browsable without authentication and clients need the same category options before applying a product-list filter. It does not expose seller-owned data and does not require seller permission middleware.

The options response provides the public category identifier and display fields required by product create/update forms, for example:

```json
{
  "public_id": "uuidv7",
  "slug": "pc-components",
  "name": "PC Components"
}
```

All categories in the canonical catalog are returned in stable name/public-ID order. The endpoint is public and does not require authentication or product permissions.

## DTO and mutation boundaries

Seller request DTOs must explicitly allow-list mutable fields. Clients must not be able to set:

```text
seller_id
public_id
status, unless a separate status-transition contract permits it
created_at
updated_at
deleted_at
```

Persistence models are not returned directly from handlers. API DTOs use the repository's response nullability rules: guaranteed fields are required, nullable fields are returned as `null`, and `omitempty` is not used to hide contract fields.

## Consequences

### Positive

- Guest public browsing remains separate from authenticated seller tooling.
- Seller A cannot read or mutate Seller B's products through a valid product permission.
- Category options remain canonical and reusable across catalog and seller clients.
- Resource-specific path names remain clear for future nested resources.
- Public and seller read models can evolve independently without leaking internal fields.

### Negative

- Seller routes need separate handlers/services or explicit service methods rather than reusing public visibility logic blindly.
- Every seller product repository query must preserve the owner predicate.
- Permission middleware and owner scoping both need focused tests.
- Seller status visibility requires a separate response contract from the public catalog.

## Verification requirements

Before considering the seller read slice complete, verify:

- unauthenticated requests are rejected;
- authenticated users without product permission are rejected;
- seller list returns only the authenticated seller's products;
- seller detail for another seller returns safe `404` behavior;
- `/categories/options` is available without seller authentication and is owned by the shared catalog/category route;
- category options come from the canonical category table;
- cursor/filter/sort behavior preserves the owner scope;
- Swagger paths and path parameter names match runtime routes;
- response JSON returns explicit `null` for nullable fields and required fields are reflected in generated Swagger;
- mass-assignment fields are ignored or rejected;
- focused tests and full repository quality gates pass.

## Evidence and requirements

- `docs/technical-assessment.md`, section 2.1.C: seller product endpoints.
- `docs/technical-assessment.md`, section 3.2: per-object authorization, BOLA/IDOR, default deny, and mass assignment.
- `internal/constants/permission/codes.go`
- `internal/modules/catalog/route.go`
- `internal/modules/catalog/handler.go`
- `internal/modules/catalog/repository.go`
- `docs/ADR/0006-public-catalog-read-api.md`
