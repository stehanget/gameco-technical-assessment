# Editor Setup Guide

This guide explains how to make the development experience smoother in editors and IDEs.

Main goal:
- format Go code automatically on save
- keep imports organized
- surface lint problems earlier

---

## 1. Recommended default

For this repo, the recommended editor behavior is:
- format on save
- organize imports on save
- use `goimports` as the save-time formatter
- use `gofumpt` + `goimports` + `golangci-lint` in repo workflow before commit

Why this split?

### On save
Use `goimports` because it is fast and practical.

### Before commit / PR
Use:
- `make fmt`
- `make lint`
- `make test`

That gives you both convenience and consistency.

---

## 2. VS Code setup

This repo now includes workspace files:
- `.vscode/settings.json`
- `.vscode/extensions.json`

### Recommended extension
Install:
- `golang.go`

That is the official Go extension for VS Code.

### What the workspace config does
The workspace settings now enable:
- `editor.formatOnSave`
- Go formatter via the Go extension
- import organization on save
- `goimports` as the main format tool
- `golangci-lint` as the lint tool
- `-count=1` in Go test flags to reduce confusion from cached tests during manual runs

### Expected save behavior
When you save a `.go` file in VS Code:
- formatting runs
- imports are organized

That gives you the “auto format on save” workflow you asked for.

---

## 3. GoLand / IntelliJ setup

If you use GoLand, enable these settings:
- Reformat code on save
- Optimize imports on save

That gets you close to the same workflow as VS Code.

Suggested habit:
- let the IDE handle save-time formatting
- still run `make fmt` and `make lint` before commit

---

## 4. Repo-level formatter/linter commands

### Format everything

```bash
make fmt
```

This runs:
- `gofumpt`
- `goimports`

### Check formatting only

```bash
make fmt-check
```

### Lint

```bash
make lint
```

### Full verification

```bash
make fmt
make lint
make build
make test
```

---

## 5. Docker equivalents

If you prefer Docker-only tooling:

```bash
make docker-fmt
make docker-lint
make docker-test
```

So even if someone does not want to rely on host Go tooling, the repo still has a shared workflow.

---

## 6. Why `goimports` on save and `gofumpt` in repo workflow?

Because it is a practical compromise.

### `goimports`
Good for editor save because it:
- formats
- fixes imports
- is fast

### `gofumpt`
Good for repo-level normalization because it is stricter than `gofmt`.

So the normal experience is:
- save file -> editor uses `goimports`
- before commit -> `make fmt` applies final normalization

---

## 7. Recommended team rule

### During coding
- rely on format-on-save

### Before commit / PR
always run:

```bash
make fmt
make lint
make test
```

### Optional: automate this with Git hooks

```bash
make install-hooks
```

Read:
- `docs/pre-commit-guide.md`

That keeps the local editing experience smooth without lowering repo quality.
