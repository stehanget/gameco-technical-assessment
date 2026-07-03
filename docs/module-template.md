# Module Template Guide

This guide is the single reference for adding a new feature module to the boilerplate.

Think of this as:

> “I want to add a new feature. What files do I create, what goes inside them, and what should I do from start to finish?”

---

## 1. Start from the example module

Before creating a new feature, look at:

```text
internal/modules/example/
```

That module exists so you can copy the default pattern instead of inventing a new one.

---

## 2. Default module shape

If you add a new feature named `project`, start with:

```text
internal/modules/project/
  dto.go
  model.go
  repository.go
  service.go
  handler.go
  route.go
```

You do not always need every file immediately, but this is the default shape because it is easy to read, easy to teach, and easy to maintain.

---

## 3. Suggested step-by-step workflow

A practical flow for a new feature usually looks like this:

1. inspect `internal/modules/example/`
2. create the new module folder
3. define DTOs
4. define database models if needed
5. write repository queries
6. write service logic
7. write handler methods
8. register routes in `internal/app/router.go`
9. add migration if the feature changes database shape
10. add Swagger comments
11. add tests
12. run verification commands

This order keeps the work boring and predictable.

---

## 4. What each file should contain

### `dto.go`
Put request/response structs here.

Examples:
- `CreateProjectRequest`
- `UpdateProjectRequest`
- `ProjectResponse`
- envelope types if needed for Swagger docs

### `model.go`
Put database structs here.

Examples:
- `Project`
- `ProjectMember`
- `TableName()` methods

### `repository.go`
Put storage access here.

Examples:
- find by ID
- list by tenant
- insert
- update
- delete

Do not put business policy here.

### `service.go`
Put business rules here.

Examples:
- validation beyond basic request binding
- transaction orchestration
- permission-sensitive operations
- conflict rules

### `handler.go`
Put HTTP endpoint handlers here.

Examples:
- parse body
- parse params
- call service
- map errors to status codes

Keep handlers thin.

### `route.go`
Register routes here.

Examples:
- route group path
- middleware requirements
- handler mapping

---

## 5. Example skeleton

### dto.go

```go
type CreateProjectRequest struct {
    Name string `json:"name" binding:"required,max=255"`
}

type ProjectResponse struct {
    ID   uint   `json:"id"`
    Name string `json:"name"`
}
```

### model.go

```go
type Project struct {
    ID       uint   `gorm:"primaryKey"`
    TenantID string `gorm:"size:64;index;not null"`
    Name     string `gorm:"size:255;not null"`
}
```

### repository.go

```go
type Repository struct {
    db *gorm.DB
}
```

### service.go

```go
type Service struct {
    repository *Repository
}
```

### handler.go

```go
type Handler struct {
    service *Service
}
```

### route.go

```go
func RegisterRoutes(router *gin.RouterGroup, handler *Handler) {
    group := router.Group("/projects")
    group.GET("", handler.List)
    group.POST("", handler.Create)
}
```

---

## 6. How to wire a module into the app

After the module exists, register it in:

```text
internal/app/router.go
```

Typical pattern:

1. create repository
2. create service
3. create handler
4. register routes

That keeps wiring explicit and keeps dependency direction obvious.

---

## 7. If the module needs database changes

You probably also need a migration.

Read:
- `docs/migrations-guide.md`

Typical flow:

1. create migration file
2. write SQL
3. run migration
4. create model
5. write repository/service/handler/route
6. test everything

Do not edit old migrations unless you are still reshaping the starter before real use.

---

## 8. If the module needs authorization

Ask these questions:

- should everyone logged in be able to access it?
- should read and write permissions be separate?
- does it require an explicit ownership scope?

Typical permission naming:
- `project.read`
- `project.write`

Then protect routes with authentication middleware and explicit object-ownership checks:

```go
middleware.Auth(jwtManager)
```

---

## 9. If the module needs docs

Document each endpoint in `handler.go` with Swagger comments.

Then run:

```bash
make swagger
```

Do not leave real endpoints undocumented.

---

## 10. If the module needs tests

At minimum, decide whether you need:
- package-level focused test
- router/e2e test
- both

Read:
- `docs/testing-guide.md`

Good default:
- one focused test for internal logic
- one e2e test for important flow

---

## 11. Suggested checklist when creating a new module

- [ ] inspect `internal/modules/example/`
- [ ] create module folder
- [ ] add `dto.go`
- [ ] add `model.go`
- [ ] add `repository.go`
- [ ] add `service.go`
- [ ] add `handler.go`
- [ ] add `route.go`
- [ ] wire module in `internal/app/router.go`
- [ ] add migration if needed
- [ ] add Swagger comments
- [ ] run `make swagger`
- [ ] add tests
- [ ] run `make fmt build test`

---

## 12. Anti-patterns to avoid

### Bad: fat handlers
If handlers start deciding too many rules, move that logic into service.

### Bad: smart repositories
If repository starts owning product policy, pull that logic back into service.

### Bad: random helpers everywhere
If a function only matters to one module, keep it in that module.

### Bad: interfaces everywhere by default
Only add interfaces when they solve a real problem.

### Bad: creating a different architecture for every module
Stay boring. Boring is maintainable.

### Bad: skipping docs for real endpoints
If the feature is real, document it.

---

## 13. Golden rule

If you are not sure how to structure a feature, follow this:

```text
handler -> service -> repository
```

And use the example module as your template.
