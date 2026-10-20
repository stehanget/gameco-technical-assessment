# Testing Guide

This guide explains how testing works in this boilerplate and how to add new tests without guessing.

---

## 1. Testing philosophy

This repo uses a practical testing approach.

The goal is not “maximum number of tests”.
The goal is:
- protect important behavior
- catch regressions early
- keep tests understandable
- make refactoring safer

---

## 2. Current testing layers

### Unit-ish / focused tests
Good for:
- helpers
- utility logic
- auth helpers
- role normalization

Examples in this repo:
- `internal/platform/auth/..._test.go`
- `internal/platform/middleware/rate_limit_test.go`


### End-to-end-ish router tests
Good for:
- real request flow
- middleware behavior
- auth behavior
- route/service/repository integration

These tests are very valuable because they exercise real HTTP flow when present.

---

## 3. Current test infrastructure

The router e2e tests currently use:
- SQLite in-memory
- miniredis
- `httptest`

That means the tests are not just mocks.
They actually hit:
- route
- middleware
- handler
- service
- repository
- DB/Redis

That is why they are useful.

---

## 4. Formatting and linting workflow

This repo now uses:
- `gofumpt` for stricter formatting
- `goimports` for formatting + import organization
- `golangci-lint` for lint aggregation

### Local commands

```bash
make fmt
make fmt-check
make lint
```

### Docker commands

```bash
make docker-fmt
make docker-lint
```

Use `make fmt` before commit.
Use `make lint` before PR/merge.

---

## 5. Common commands

### Run all tests

```bash
make test
```

### Run focused e2e tests

```bash
make test-e2e
```

### Run full verification

```bash
make fmt
make lint
make build
make test
```

### Run one Go package

```bash
go test ./internal/modules/auth -count=1
```

### Run one specific e2e test

```bash
go test ./internal/platform/auth -run TestJWTManagerGenerateAndParseAccessToken -count=1
```

The `-count=1` flag is useful when you want to avoid cached test results.

---

## 6. When to write what kind of test

### Write a focused/unit test when:
- function has clear deterministic logic
- no real HTTP flow needed
- no database behavior needed
- you want quick verification of a small rule

Examples:
- slug normalization
- permission mapping
- JWT generation/parsing
- password hashing helpers

### Write an e2e/router test when:
- middleware matters
- HTTP status codes matter
- request/response body matters
- permission checks matter
- several layers interact together

Examples:
- register -> login -> refresh
- seed roles -> permissions -> role_permissions -> user_roles
- rerun the seeder without duplicating assignments

---

## 7. Recommended workflow when adding a feature

### If the feature changes behavior:
1. add the test first if possible
2. run the focused test
3. implement or update code
4. rerun the focused test
5. run full verification

Suggested commands:

```bash
go test ./internal/app -run TestSomething -count=1
make fmt
make lint
make build
make test
```

---

## 8. Where to put tests

### For helper/service-specific logic
Put tests close to the package:

```text
internal/modules/project/service_test.go
internal/platform/auth/jwt_test.go
```

### For cross-module request flow
Put HTTP flow tests under the owning package or `internal/app/` when the flow spans
multiple modules. Keep the test focused on one business behavior.

---

## 9. What a good test should check

A good test should make it obvious:
- what is being tested
- what input was given
- what output/status is expected
- why failure matters

Bad test names:
- `TestItWorks`
- `TestAPI`
- `TestManager`

Better names:
- `TestJWTManagerGenerateAndParseAccessToken`
- `TestDefaultSeedersAreIdempotentAndAssignMultipleRoles`
- `TestHashPasswordAndCompare`

---

## 10. Example: adding a new feature test

Say you add a new `project` module.

You might add:

### Service-level test
- project slug normalization
- project validation rules

### Router/e2e test
- owner creates project
- member without `project.write` gets `403`
- owner can list projects

That gives both narrow and broad confidence.

---

## 11. What to verify before saying “done”

Before calling a feature complete, run:

```bash
make fmt
make lint
make build
make test
```

If you changed Swagger comments:

```bash
make swagger
```

If you changed DB shape:

```bash
make docker-migrate-up
```

---

## 12. Common mistakes

### Only testing happy path
Also test:
- permission denied
- not found
- invalid payload
- unsafe operation blocked

### Overusing mocks
For this repo, real-flow tests are often more useful than heavy mocks.

### Writing giant unreadable tests
Prefer small scenario-based tests over one monster test doing 20 unrelated things.

### Forgetting session refresh after role change
Roles and permissions are resolved from the active database-backed auth session, not embedded in the JWT claims. After changing role or permission assignments, use a newly issued or refreshed session before testing access again.

### Running only `go test` and skipping format/lint
That misses style/import/lint issues. Use the full check when finishing work.
