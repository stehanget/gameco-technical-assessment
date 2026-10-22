# Render and Read the PostgreSQL ERD

This guide explains how to render the repository's conceptual schema from `docs/schema.dbml` using [dbdiagram.io](https://dbdiagram.io/d).

## What this diagram represents

`docs/schema.dbml` is the repository's DBML-based ERD documentation. It describes tables, columns, indexes, enums, and declared relationships in a form that dbdiagram.io can visualize. DBML is a schema-definition language used to define and document database structures, while dbdiagram.io renders the DBML editor alongside the diagram view.

The runtime database source of truth remains the PostgreSQL migrations under `migrations/`. The DBML file is the human-readable conceptual/schema documentation and must stay synchronized with those migrations; rendering it does not execute migrations or inspect the live database.

## Render the ERD in the browser

1. Open [dbdiagram.io/d](https://dbdiagram.io/d).
2. Create a new diagram or open an existing diagram workspace.
3. Open the repository file:

   ```text
   docs/schema.dbml
   ```

4. Copy the complete DBML content.
5. Paste it into the DBML editor on the left side of the dbdiagram.io workspace.
6. Read the rendered ERD in the diagram pane on the right. The diagram updates as the DBML editor content changes.

### Copy commands

macOS:

```bash
pbcopy < docs/schema.dbml
```

Windows PowerShell:

```powershell
Get-Content .\docs\schema.dbml -Raw | Set-Clipboard
```

Linux with `xclip`:

```bash
xclip -selection clipboard < docs/schema.dbml
```

If clipboard tooling is unavailable, open the file in an editor and copy its contents manually.

## How to read the rendered ERD

### Table boxes

Each table is rendered as a box containing its columns. Read the column markers as follows:

- `pk` — primary key;
- `increment` — auto-incrementing database identifier;
- `not null` — required value;
- `unique` — unique value or unique constraint;
- `default` — database default expression/value;
- `Note` — additional behavior or ownership/concurrency context documented for the table.

### Index sections

The `indexes` block inside a table documents important lookup, uniqueness, and ordering indexes. Pay particular attention to indexes used by:

- ownership queries, such as buyer/seller scoping;
- idempotency lookups;
- payment webhook replay protection;
- stock movement history;
- audit event investigation;
- cursor-based ordering and filtering.

An index shown in DBML is documentation of schema intent. Confirm the exact physical index definition in the corresponding SQL migration when reviewing performance or deployment behavior.

### Relationship lines

Relationship lines use DBML `Ref` declarations. The direction and symbols show how keys relate. For example:

```dbml
Ref: orders.id < order_items.order_id
```

This represents one order related to many order items. A relationship rendered in the ERD reflects a declared DBML reference. Do not infer an actual PostgreSQL foreign key from an application-level naming convention alone; verify the migration before treating it as a physical constraint.

### Enums and status values

Enums such as order/payment/product or stock-movement statuses are rendered as shared value definitions. Read them together with the table columns that use them and the migration that creates/alters the PostgreSQL enum.

## Marketplace areas to inspect first

For the assessment flow, start with these table groups:

### Identity and authorization

```text
users
roles
permissions
user_roles
role_permissions
auth_sessions
user_login_security_states
```

### Catalog and cart

```text
categories
products
carts
cart_items
```

### Checkout and payment

```text
checkout_idempotency_keys
orders
order_items
payments
payment_webhook_events
```

### Inventory and audit

```text
stock_movements
audit_events
```

A useful reading path is:

```text
users
  → carts
  → cart_items
  → products
  → orders
  → order_items
  → payments
  → stock_movements
  → audit_events
```

This is a conceptual reading path, not a claim that every relation in that list is represented by a direct foreign key.

## Reading the checkout area

For `POST /api/v1/checkout`, inspect the relationships and notes around:

1. `checkout_idempotency_keys` — prevents duplicate order creation for the same buyer/key pair;
2. `carts` and `cart_items` — source of the buyer's active cart;
3. `products` — current price and current stock balance;
4. `orders` and `order_items` — immutable checkout snapshots;
5. `payments` — initial payment lifecycle state;
6. `stock_movements` — before/after inventory ledger;
7. `audit_events` — append-only security/business evidence.

The ERD shows the data model. The runtime ordering and transaction boundary are documented separately in [Checkout request to stock reservation](flows/checkout-stock-reservation.md).

## Exporting the diagram

After the diagram renders, use the dbdiagram.io toolbar to export a visual artifact such as PNG or PDF when a review or assessment handoff needs a static ERD. The repository should continue to store `docs/schema.dbml` as the editable source rather than committing a browser-exported image as the primary schema documentation. dbdiagram.io documents export actions in its basic editing workflow.

## Troubleshooting

### The diagram is blank or partially rendered

- Confirm that the complete contents of `docs/schema.dbml` were copied.
- Check the editor for a syntax error marker.
- Make sure fenced Markdown markers such as `````dbml`` are not pasted into the DBML editor; paste only the DBML source itself.
- Re-open the file locally and verify that the copied content was not truncated.

### A relationship is missing

- Check whether the table has a corresponding `Ref:` declaration in `docs/schema.dbml`.
- Check the migration SQL to see whether the relationship is a physical foreign key or only an application-level association.
- Do not add a DBML `Ref` merely because two columns have matching names.

### The rendered schema differs from PostgreSQL

Treat this as schema-documentation drift. Compare, in order:

1. the relevant Goose migration files;
2. the current Go model and repository query assumptions;
3. `docs/schema.dbml`;
4. the rendered diagram.

Update DBML only after confirming the physical PostgreSQL schema and migration ownership. Do not use dbdiagram.io as a replacement for migration verification.

### PostgreSQL-specific syntax does not render exactly

DBML is a documentation/visualization format, not a complete PostgreSQL migration parser. Preserve important PostgreSQL behavior in table notes when DBML cannot express an index predicate, check expression, partial index, or native enum detail exactly. Keep the authoritative implementation in the migration SQL.

## Related documentation

- [Runtime flow documentation](flows/README.md)
- [Database schema](schema.dbml)
- [Migrations guide](migrations-guide.md)
- [Checkout request to stock reservation](flows/checkout-stock-reservation.md)
- [Stock concurrency ADR](ADR/0002-stock-concurrency.md)
- [Database and identifier ADR](ADR/0003-database-and-identifier-model.md)

## Source references

- [dbdiagram.io](https://dbdiagram.io/d)
- [DBML documentation](https://docs.dbdiagram.io/dbml/)
- [Basic editing experience](https://docs.dbdiagram.io/basic-editing-experience/)
- [DBML syntax](https://dbml.dbdiagram.io/docs/)
