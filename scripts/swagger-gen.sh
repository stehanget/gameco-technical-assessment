#!/usr/bin/env sh
set -eu

if ! command -v go >/dev/null 2>&1; then
  echo "go command not found" >&2
  exit 1
fi

GO_BIN_DIR="$(go env GOPATH)/bin"
export PATH="$GO_BIN_DIR:$PATH"

if ! command -v swag >/dev/null 2>&1; then
  go install github.com/swaggo/swag/cmd/swag@latest
fi

swag init -g cmd/api/main.go -o docs/swagger
