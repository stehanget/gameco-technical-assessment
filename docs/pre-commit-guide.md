# Pre-commit Guide

This guide explains how to install and use the repo's Git pre-commit hook.

---

## 1. What it does

The repo includes a Git pre-commit hook at:

```text
.githooks/pre-commit
```

When installed, it runs before every commit:

```bash
make fmt-check
make lint
make test
```

If any step fails, the commit is blocked.

---

## 2. Why use it?

It helps catch problems before code lands in history:
- formatting drift
- lint issues
- broken tests

It does not replace CI, but it reduces silly mistakes.

---

## 3. Install the hook

From the repo root:

```bash
make install-hooks
```

This points Git to use the repo's `.githooks/` directory.

You only need to do this once per clone.

---

## 4. Verify it

Check Git config:

```bash
git config core.hooksPath
```

Expected output:

```text
.githooks
```

---

## 5. What happens during commit

When you run:

```bash
git commit -m "your message"
```

The hook runs:

1. `make fmt-check`
2. `make lint`
3. `make test`

If all pass, the commit continues.
If any fail, the commit stops and you fix the issue first.

---

## 6. If the hook fails

Typical flow:

### Formatting issue
Run:

```bash
make fmt
```

Then re-stage files and commit again.

### Lint issue
Run:

```bash
make lint
```

Read the output and fix the code.

### Test issue
Run:

```bash
make test
```

Fix the failing behavior before committing.

---

## 7. Docker-first teams

This hook uses host commands by default:

- `make fmt-check`
- `make lint`
- `make test`

If your team wants a stricter Docker-only hook later, you can switch it to:

- `make docker-fmt`
- `make docker-lint`
- `make docker-test`

But the current default is faster and more practical for most developers.

---

## 8. If you want to skip it temporarily

You can bypass Git hooks with:

```bash
git commit --no-verify
```

Do this sparingly.

If someone uses `--no-verify` all the time, the hook becomes useless.

---

## 9. Team recommendation

Recommended rule:

- install hooks once per clone
- keep format-on-save enabled
- let the hook catch the rest
- still rely on CI as final safety net

This is the best balance between convenience and discipline.
