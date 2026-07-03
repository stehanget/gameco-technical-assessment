# Getting Started

This guide is for someone who is new to the repo and wants to get it running once without guessing.

If you already have the project booted and want the day-to-day Docker workflow, read:
- `docs/docker-development-guide.md`

---

## 1. What you need installed

For the easiest path, install:
- Docker
- Docker Compose

Optional, but useful if you want host-side tooling too:
- Go
- Make
- Goose

You do **not** need PostgreSQL or Redis installed manually if you use Docker Compose.

### Windows users
If you are on Windows, the recommended path is:
- WSL2
- Ubuntu inside WSL2
- Docker Desktop with WSL integration

Read:
- `docs/windows-guide.md`

---

## 2. Clone and enter the project

```bash
git clone <your-repo-url>
cd gameco-technical-assessment
```

---

## 3. Create your local env file

```bash
cp .env.example .env
```

Open `.env` and adjust values if needed.

For Docker-first development, the defaults are fine most of the time.

Important fields:
- `APP_PORT`
- `APP_JWT_SECRET`
- `POSTGRES_*`
- `REDIS_*`
- `APP_ACCESS_TOKEN_TTL`
- `APP_REFRESH_TOKEN_TTL`

---

## 4. Start the full dev stack with live reload

```bash
make docker-dev-up
```

This starts:
- Postgres
- Redis
- API
- Worker

The API and worker are mounted from your source tree, so code changes can trigger rebuild/restart inside the dev containers.

---

## 5. Run database migrations

```bash
make docker-migrate-up
```

This creates the tables needed by the app.

### Optional: install Goose globally
If you want to create migration files manually from your terminal, install Goose globally:

```bash
go install github.com/pressly/goose/v3/cmd/goose@latest
export PATH="$PATH:$(go env GOPATH)/bin"
source ~/.zshrc
goose -h
```

Example create command:

```bash
goose -dir migrations create create_projects sql
```

---

## 6. Generate API docs

### If you have local Go

```bash
make swagger
```

### If you want Docker-only tooling

```bash
make docker-swagger
```

This generates Swagger/OpenAPI files under `docs/swagger/`.

---

## 7. Open the app

API base URL:

```text
http://localhost:8080
```

Swagger UI:

```text
http://localhost:8080/swagger/index.html
```

---

## 8. Recommended first manual test

### Register a buyer
Use Swagger UI or curl:

```bash
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H 'Content-Type: application/json' \
  -d '{
    "email": "buyer@example.com",
    "password": "password123"
  }'
```

That first register call will:
- create the user
- assign the default `buyer` role
- return access + refresh tokens

If this works, your local setup is basically healthy.

---

## 9. Optional: seed predictable demo data

If you want ready-made demo users and sample rows instead of registering everything manually:

```bash
make docker-seed
```

That gives you predictable demo credentials and sample rows.

---

## 10. What to read next

Now choose the next doc based on what you want:

- daily Docker workflow -> `docs/docker-development-guide.md`
- Windows / WSL2 help -> `docs/windows-guide.md`
- Swagger endpoint reference -> `http://localhost:8080/swagger/index.html`
- adding a new module -> `docs/module-template.md`

---

## 11. Common problems

### Port already used
Change `APP_PORT` in `.env`.

### Database connection failed
Make sure:
- Docker is running
- `make docker-dev-up` or `make docker-up` succeeded
- env values match compose config

### Redis connection failed
Same check as above — Redis is started by compose.

### Swagger route exists but docs are old
Run one of these:

```bash
make swagger
make docker-swagger
```

### `make` not found
If you do not have `make`, you can still run Docker commands manually.
Read:
- `docs/windows-guide.md`
- `docs/docker-development-guide.md`

---

## 12. Files you should know first

If you want to understand the repo quickly, read these in order:

1. `README.md`
2. `docs/getting-started.md`
3. `docs/docker-development-guide.md`
4. `internal/app/router.go`
5. `internal/modules/auth/`
6. `internal/modules/example/`
7. `docs/module-template.md`
