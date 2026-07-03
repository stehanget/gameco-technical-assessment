# Migration Guide

This guide explains how to create, write, run, and review database migrations in this boilerplate.

If you are new to backend work, read this before touching anything inside `migrations/`.

---

## 1. What is a migration?

A migration is a versioned database change.

Examples:
- create a new table
- add a column
- add an index
- rename a field
- backfill old data
- remove unused schema

In this project, migrations are executed with **Goose**.

---

## 2. Where migrations live

All migration files are stored in:

```text
migrations/
```

Current examples:
- `20260701090000_create_examples.sql`
- `20260701090500_create_users.sql`
- `20260701091000_create_tenant_rbac.sql`
- `20260701091500_make_users_global.sql`

---

## 3. Migration naming rule

For this boilerplate, prefer Goose's **default timestamp-based naming**.

Use this pattern:

```text
20260701221530_create_projects.sql
20260701221610_add_status_to_projects.sql
20260701221705_create_project_members.sql
```

### Good names
- `20260701221530_create_projects.sql`
- `20260701221610_add_avatar_url_to_users.sql`
- `20260701221705_drop_legacy_sessions.sql`

### Bad names
- `new.sql`
- `fix.sql`
- `update_table.sql`

Rule of thumb:
- keep Goose default timestamp prefix
- make the filename describe the schema change clearly

---

## 4. SQL migration file format

A Goose SQL migration should have two sections:

```sql
-- +goose Up
CREATE TABLE ...;

-- +goose Down
DROP TABLE ...;
```

### Example

```sql
-- +goose Up
CREATE TABLE IF NOT EXISTS projects (
    id BIGSERIAL PRIMARY KEY,
    tenant_id VARCHAR(64) NOT NULL,
    name VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- +goose Down
DROP TABLE IF EXISTS projects;
```

---

## 5. How to create a new migration

Yes — Goose **can** create migration files from a command.

### Recommended flow for this boilerplate
Use Goose's default timestamp-based naming:

```bash
goose -dir migrations create create_projects sql
```

If Goose is not installed globally, you can still run it with Go:

```bash
go run github.com/pressly/goose/v3/cmd/goose@latest -dir migrations create create_projects sql
```

### What this does
It generates a new file inside `migrations/`.

Example result:

```text
migrations/20260701221530_create_projects.sql
```

### Good examples

Create SQL migration:

```bash
goose -dir migrations create add_status_to_projects sql
```

Create Go migration:

```bash
goose -dir migrations create backfill_project_status go
```

### Recommendation for this repo
Use this pattern:

```bash
goose -dir migrations create <name> sql
```

That matches the timestamp-based style we want to keep in this boilerplate.

## 5.1 Does Goose have seeders?

Not as a separate first-class concept.

Goose gives you migration commands like `create`, `up`, `down`, `redo`, `status`, `fix`, and `validate`.
There is no dedicated `seed` command.

So in practice you have three common choices:

1. put essential bootstrap data inside a migration
   - good for things like permission catalog rows or required system defaults
2. create a Go-based migration when the seed logic is procedural
   - useful for backfills or more complex inserts
3. keep a separate app-level seed command or script outside Goose
   - better for optional demo data, fake users, or local playground fixtures

For this boilerplate, the best default is:
- required system data -> Goose migration
- optional sample/demo data -> separate seed command later if needed

### Install Goose globally
If you want `goose ...` to work directly in your shell, install it globally:

```bash
go install github.com/pressly/goose/v3/cmd/goose@latest
```

Then make sure your Go bin directory is on your `PATH`.

On macOS/Linux with the default Go setup, add this to your shell config (`~/.zshrc`, `~/.bashrc`, or similar):

```bash
export PATH="$PATH:$(go env GOPATH)/bin"
```

Reload your shell:

```bash
source ~/.zshrc
```

Then verify:

```bash
goose -h
```

If `go env GOPATH` returns empty or you use a custom setup, you can also inspect:

```bash
go env GOPATH
go env GOBIN
```

Use whichever directory actually contains the installed `goose` binary.

---

## 6. How to think before writing a migration

Before you write SQL, answer these questions:

### A. What are you changing?
Examples:
- new table
- new column
- new relation
- rename
- delete old structure

### B. Is this a schema change or data change?
- schema change = table/column/index/constraint
- data change = update existing rows / backfill / cleanup

### C. Can this break existing data?
If yes, be extra careful.

### D. Can the change be reversed safely?
That determines what your `Down` block should do.

---

## 7. Safe habits when writing migrations

### Prefer additive changes first
Safer pattern:
1. add new column/table
2. backfill data
3. switch application logic
4. remove old column later

That is safer than destructive one-step changes.

### Use explicit names
Bad:
- generic index names
- vague constraints

Good:
- names that explain what they are

### Keep one migration focused
One migration should usually do one logical thing.

Good:
- create projects table

Bad:
- create 4 unrelated tables, rename 2 columns, and delete old data in the same file

### Be careful with destructive down migrations
Sometimes `DROP TABLE` or `DROP COLUMN` is fine for starter-phase work.
But once real data exists, down migrations can be dangerous.

---

## 8. Running migrations

### Run up

```bash
make migrate-up
```

Or inside Docker:

```bash
make docker-migrate-up
```

### Run down

```bash
make migrate-down
```

Use down carefully.

---

## 9. Suggested workflow when adding a feature with DB changes

Example: you want to add a `project` feature.

### Step 1
Create migration with Goose:

```bash
goose -dir migrations create create_projects sql
```

### Step 2
Write table SQL.

### Step 3
Run migration:

```bash
make docker-migrate-up
```

### Step 4
Add Go model in:

```text
internal/modules/project/model.go
```

### Step 5
Add repository/service/handler/route.

### Step 6
Run verification:

```bash
make fmt
make build
make test
```

---

## 10. Example recipes

### Create a new table

```sql
-- +goose Up
CREATE TABLE IF NOT EXISTS projects (
    id BIGSERIAL PRIMARY KEY,
    tenant_id VARCHAR(64) NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- +goose Down
DROP TABLE IF EXISTS projects;
```

### Add a column

```sql
-- +goose Up
ALTER TABLE projects
ADD COLUMN IF NOT EXISTS status VARCHAR(50) NOT NULL DEFAULT 'active';

-- +goose Down
ALTER TABLE projects
DROP COLUMN IF EXISTS status;
```

### Add an index

```sql
-- +goose Up
CREATE INDEX IF NOT EXISTS idx_projects_tenant_id ON projects(tenant_id);

-- +goose Down
DROP INDEX IF EXISTS idx_projects_tenant_id;
```

### Backfill existing data

```sql
-- +goose Up
UPDATE projects
SET status = 'active'
WHERE status IS NULL;

-- +goose Down
UPDATE projects
SET status = NULL
WHERE status = 'active';
```

---

## 11. What not to do

### Do not edit old migrations casually
Once a migration has been used, treat it as history.

Editing old migrations after other people already ran them causes confusion.

For starter-only cleanup during heavy scaffolding, it can still happen — but once the repo is shared, stop doing that.

### Do not mix too many unrelated changes
Keep migrations focused.

### Do not rely on ORM auto-migrate as your main schema strategy
This repo uses Goose migrations as the real schema history.

### Do not forget the `Down` section
Even if the down path is simple, include it.

---

## 12. Quick checklist before you commit a migration

- [ ] filename uses Goose timestamp prefix
- [ ] filename clearly describes the change
- [ ] has `-- +goose Up`
- [ ] has `-- +goose Down`
- [ ] SQL is focused on one logical change
- [ ] verified with `make docker-migrate-up`
- [ ] verified app still passes `make fmt build test`

---

## 13. Rule of thumb

If you are not sure whether something belongs in a migration, ask:

> “Does this change the database structure or stored data?”

If yes, it probably belongs in a migration.

If it only changes request validation, service logic, or response shape, it probably does **not**.
