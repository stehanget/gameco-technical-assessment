# Windows Guide

This guide explains the **recommended** way to run this boilerplate on Windows.

Short version:

## Recommended path
Use:
- **Windows + WSL2**
- **Docker Desktop**
- Docker-first development commands
- project files opened from WSL/Linux side when possible

That is the cleanest and least painful setup.

---

## 1. Important expectation

This boilerplate is currently much more naturally aligned with:
- Linux
- macOS
- WSL2

It is **not** documented as a native PowerShell-first repo.

That means:
- the safest Windows setup is **WSL2-first**
- Docker-first development is the recommended path

If you want the smoothest experience on Windows, do **not** start with raw `cmd.exe` or plain PowerShell and hope everything behaves the same.

---

## 2. Recommended Windows setup

Install these:
1. **WSL2**
2. **Ubuntu (inside WSL2)**
3. **Docker Desktop** with WSL integration enabled
4. **Make** inside WSL if you want `make ...`
5. **Go** inside WSL only if you want host-side Go commands too

---

## 3. Why WSL2 is recommended

Because this repo uses workflows that feel Linux-like:
- Makefile commands
- shell scripts
- Docker CLI flow
- Go tooling assumptions
- path conventions that are easier in Unix-like shells

WSL2 gives you a Linux environment on Windows, so the repo behaves much closer to how it behaves on macOS/Linux.

---

## 4. Install WSL2

Open PowerShell as Administrator and run:

```powershell
wsl --install
```

Then reboot if Windows asks.

After that, install Ubuntu from the Microsoft Store if it is not installed automatically.

Verify:

```powershell
wsl -l -v
```

You want to see your distro using **version 2**.

---

## 5. Install Docker Desktop

Install Docker Desktop normally on Windows.

Then in Docker Desktop:
- go to **Settings**
- open **Resources > WSL Integration**
- enable integration for your Ubuntu distro

Verify from inside WSL:

```bash
docker version
docker compose version
```

If both commands work inside WSL, your Docker setup is in a good place.

---

## 6. Install Make inside WSL

Inside Ubuntu/WSL:

```bash
sudo apt update
sudo apt install make
```

Verify:

```bash
make --version
```

---

## 7. Optional: install Go inside WSL

If you want to run Go directly inside WSL:

```bash
sudo apt update
sudo apt install golang-go
```

Verify:

```bash
go version
```

You do **not** need local Go just to use the Docker-first workflow.

---

## 8. Optional: install Goose globally inside WSL

If you want the `goose` command directly:

```bash
go install github.com/pressly/goose/v3/cmd/goose@latest
```

Then add Go bin to PATH if needed:

```bash
echo 'export PATH="$PATH:$(go env GOPATH)/bin"' >> ~/.bashrc
source ~/.bashrc
```

Verify:

```bash
goose -h
```

---

## 9. Recommended way to open the repo

Best practice:
- clone and work with the repo **inside WSL filesystem**
- not directly inside a Windows-mounted path when possible

Example:

```bash
cd ~
git clone <your-repo-url>
cd gameco-technical-assessment
```

Why?
Because Docker, Go, file permissions, and path behavior are usually cleaner there.

---

## 10. Recommended development flow on Windows

Inside WSL terminal:

### Start full dev stack with live reload

```bash
make docker-dev-up
```

### Run migrations

```bash
make docker-migrate-up
```

### Generate Swagger docs

```bash
make docker-swagger
```

### Watch logs

```bash
make docker-dev-logs
```

### Stop the stack

```bash
make docker-dev-down
```

This is the recommended daily workflow.

---

## 11. If you want host-side Go commands too

You can still use:

```bash
make run
make test
make swagger
```

But for a mixed-OS team, Docker-first is the safer shared default.

---

## 12. If you do NOT want to use `make`

You can still run commands manually.

### Start full dev stack

```bash
docker compose -f deployments/compose/docker-compose.yml -f deployments/compose/docker-compose.dev.yml up --build -d postgres redis api worker
```

### Run migrations

```bash
docker compose -f deployments/compose/docker-compose.yml run --rm migrate up
```

### Generate Swagger

```bash
docker run --rm -v "$PWD:/app" -w /app golang:1.25 sh -lc 'go run github.com/swaggo/swag/cmd/swag@latest init -g cmd/api/main.go -o docs/swagger'
```

### Stop the stack

```bash
docker compose -f deployments/compose/docker-compose.yml -f deployments/compose/docker-compose.dev.yml down
```

---

## 13. Native Windows caveat

Can someone run this repo on Windows?

### With WSL2
Yes, that is the recommended path.

### With raw PowerShell only
Maybe partially, but this repo is not documented as native-Windows-first right now.

That means:
- fewer guarantees
- more chance of path/script friction
- more manual workaround risk

So if you want a reliable answer for your team:

## Supported Windows path = WSL2 + Docker Desktop + Docker-first workflow

---

## 14. Troubleshooting

### Docker works in Windows but not in WSL
Check Docker Desktop WSL integration settings.

### `make` not found
Install it in WSL:

```bash
sudo apt install make
```

### `goose` not found
Install it in WSL with:

```bash
go install github.com/pressly/goose/v3/cmd/goose@latest
```

### `go` not found
Install Go inside WSL only if you want local Go commands.

### Paths look weird
Make sure you are running commands in WSL shell, not mixing Windows shell and Linux shell randomly.

---

## 15. Bottom line

If your users are mixed across OSes, the cleanest shared guidance is:
- macOS/Linux: Docker-first preferred
- Windows: WSL2 + Docker Desktop + Docker-first strongly recommended

That gives you the most predictable team workflow without pretending native Windows behavior is fully identical.
