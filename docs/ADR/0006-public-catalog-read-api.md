# ADR-0006: Public Catalog Read API

- **Status:** Accepted
- **Date:** 2026-10-21
- **Scope:** Public product catalog listing and detail endpoints

## Context

The marketplace needs a public catalog that can be browsed by guests and first-party clients without authentication. The catalog must support useful filtering and sorting while preserving safe SQL construction, deterministic pagination, and the separation between internal database identifiers and public resource identifiers.

The first catalog slice covers product listing and product detail only. Seller product mutations, cart operations, checkout, order reads, and payment webhooks remain separate decisions and implementation slices.

## Options considered

1. Expose only an unfiltered product list and add query capabilities later.
2. Accept arbitrary sort column and direction values from the request.
3. Build a public read API with explicit filters, allow-listed sorting, cursor pagination, and UUIDv7 public identifiers.
4. Add a cache immediately to optimize catalog reads before measuring query behavior and invalidation needs.

## Decision

Implement the following public endpoints:

```text
GET /api/v1/products
GET /api/v1/products/{public_id}
```

Both endpoints are accessible without authentication.

## List contract

`GET /api/v1/products` accepts these optional query parameters:

```text
q
category
min_price
max_price
sort_by
sort_order
cursor
per_page
```

### Search and filters

- `q` performs a case-insensitive substring search against product `name` and `description`.
- The search path is part of this slice and uses PostgreSQL `pg_trgm` GIN indexes so the search does not rely on an unindexed wildcard scan as the catalog grows.
- `category` resolves by the category slug, never by an internal category ID.
- `min_price` and `max_price` use integer monetary amounts and never floating-point values.
- Prices cannot be negative.
- When both bounds are supplied, `min_price` must not exceed `max_price`.
- Invalid query values return the centralized `400` validation error envelope.

### Sorting

Sorting uses the explicit parameters:

```text
sort_by
sort_order
```

The accepted values are:

```text
sort_by:    created_at | name | price
sort_order: asc | desc
```

Defaults are:

```text
sort_by=created_at
sort_order=desc
```

Request values are translated through an application allow-list to known SQL columns. They are never interpolated directly into SQL. `price` maps to the persisted integer price column.

Every result set uses `products.id ASC` as a final tie-breaker so identical sort values produce deterministic pagination.

### Pagination

The list uses opaque cursor pagination:

```text
cursor=<opaque-cursor>
per_page=20
```

`per_page` is capped at `100`.

The first request omits `cursor`. Subsequent requests send the `next_cursor` returned by the previous response. The cursor contains the ordered position required by the selected `sort_by`/`sort_order` plus the deterministic public-ID tie-breaker, but its representation is opaque and is not part of the public contract. Internal database IDs are not encoded in the cursor. A cursor is valid only with the same filter and sorting parameters used to create it.

The response includes:

```json
{
  "data": [],
  "meta": {
    "per_page": 20,
    "has_more": false,
    "next_cursor": null
  }
}
```

Cursor pagination is selected because it keeps page traversal stable when rows are inserted or removed between requests and avoids the deep-page cost of large offsets. The public-ID tie-breaker is included in the cursor position so equal names, prices, or timestamps do not produce duplicates or skipped rows without exposing internal database IDs.

## Public visibility

Only products satisfying both conditions are visible in the public catalog:

```text
status = active
deleted_at IS NULL
```

Draft, inactive, and soft-deleted products are treated as unavailable. A detail lookup for an unavailable product returns the same safe `404` resource-not-found behavior as an unknown product.

## Identifier and response rules

- Product detail uses the UUIDv7 `public_id` path parameter.
- Internal `BIGINT` product, seller, and category IDs are never returned to clients.
- Category data is represented with its public `slug` and display `name`.
- Seller information is intentionally omitted from the initial public catalog response. Seller-facing data can be added later through an explicit contract decision.

## Query and performance rules

The list query joins the category table in one query to avoid an N+1 query pattern. The query must use parameter binding for values and an application allow-list for dynamic sort expressions.

The search implementation uses PostgreSQL `pg_trgm` with GIN indexes on `products.name` and `products.description`. The extension and indexes are managed by migrations. Search input remains parameterized, and the query must preserve the active/non-deleted visibility predicates.

The catalog slice does not introduce caching initially. Database reads are simple, catalog volume is not yet measured, and premature caching would introduce invalidation and stale-data behavior before the product mutation contract exists.

The implementation must review or add indexes for the selected access paths, including status, category, creation time, price, and soft-delete filtering, without adding indexes solely by assumption.

## Error behavior

- Invalid filter, sort, cursor, or pagination input returns `400` through `httpx.HandleError`.
- Unknown, inactive, or soft-deleted products return `404` without revealing whether a hidden product exists.
- Unexpected repository or infrastructure failures return the safe centralized `500` response.
- No SQL, stack traces, framework details, or raw internal errors are exposed to clients.

## Consequences

### Positive

- Guests can browse a predictable and useful catalog without authentication.
- Sorting is safe because dynamic SQL fragments come from an allow-list.
- UUIDv7 public identifiers prevent internal database IDs from becoming part of the API contract.
- Deterministic tie-breaking prevents unstable page boundaries.
- The initial implementation avoids cache invalidation complexity and N+1 queries.
- Seller and transactional concerns remain separate from the public read model.

### Negative

- Cursor pagination requires opaque cursor encoding, validation, and a stable ordering contract.
- Trigram indexes add migration and write/storage overhead to support substring search over name and description.
- The catalog initially omits seller display data.
- Without caching, repeated public reads continue to hit PostgreSQL.
- The sort allow-list requires explicit maintenance when new sort options are introduced.

## Implementation status

Decision accepted. The next implementation slice is:

1. catalog read DTOs;
2. PostgreSQL `pg_trgm` extension and product search indexes;
3. repository query with filters, allow-listed sorting, cursor pagination, and search;
4. public list/detail service and handlers;
5. Swagger contract;
6. focused tests for filtering, sorting, cursor traversal, visibility, public identifiers, search, and error behavior.

## Evidence and requirements

- `docs/technical-assessment.md`, section 2.1.B: public catalog endpoints.
- `docs/technical-assessment.md`, section 3.4: input validation, safe dynamic filtering, and response leakage rules.
- `docs/technical-assessment.md`, section 4: indexing, N+1 avoidance, and pagination decisions.
- `internal/modules/catalog/model.go`
- `migrations/20261021090000_create_catalog.sql`
- `docs/schema.dbml`
