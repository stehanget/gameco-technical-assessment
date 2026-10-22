GO ?= go
COMPOSE_FILE ?= deployments/compose/docker-compose.yml
DEV_COMPOSE_FILE ?= deployments/compose/docker-compose.dev.yml
DOCKER_WORKDIR ?= /app
DOCKER_GO_IMAGE ?= golang:1.26.10
DOCKER_GO_RUN ?= docker run --rm -v "$(CURDIR):$(DOCKER_WORKDIR)" -w $(DOCKER_WORKDIR) $(DOCKER_GO_IMAGE)
SECURITY_GOTOOLCHAIN ?= go1.26.10
ENV_FILE ?= .env
DOCKER_COMPOSE ?= docker compose --env-file $(ENV_FILE) -f $(COMPOSE_FILE)
DOCKER_DEV_COMPOSE ?= docker compose --env-file $(ENV_FILE) -f $(COMPOSE_FILE) -f $(DEV_COMPOSE_FILE)
TEST_ENV_FILE ?= .env.test

.PHONY: run worker seed migrate-up migrate-down build fmt fmt-check tidy test test-e2e test-integration test-runtime check verify lint security-check install-hooks swagger docker-swagger docker-up docker-infra-up docker-dev-up docker-dev-down docker-dev-reset docker-down docker-logs docker-dev-logs docker-ps docker-dev-ps docker-migrate-up docker-migrate-down docker-dev-migrate-up docker-dev-migrate-down docker-build docker-test docker-test-e2e docker-test-integration docker-test-runtime docker-fmt docker-lint docker-seed docker-dev-seed

run:
	$(GO) run ./cmd/api

worker:
	$(GO) run ./cmd/worker

seed:
	set -a; if [ -f ./.env ]; then . ./.env; else . ./.env.example; fi; set +a; $(GO) run ./cmd/seed

migrate-up:
	$(GO) run ./cmd/migrate up

migrate-down:
	$(GO) run ./cmd/migrate down

build:
	$(GO) build ./...

fmt:
	$(GO) run mvdan.cc/gofumpt@latest -w .
	$(GO) run golang.org/x/tools/cmd/goimports@latest -w .

fmt-check:
	$(GO) run mvdan.cc/gofumpt@latest -l .
	$(GO) run golang.org/x/tools/cmd/goimports@latest -l .

tidy:
	$(GO) mod tidy

test:
	$(GO) test -v ./internal/...

test-e2e:
	TEST_ENV_FILE="$(TEST_ENV_FILE)" ./scripts/test-e2e.sh

test-integration:
	TEST_ENV_FILE="$(TEST_ENV_FILE)" ./scripts/test-integration.sh

test-runtime:
	TEST_ENV_FILE="$(TEST_ENV_FILE)" ./scripts/test-runtime.sh

check: fmt-check lint test
	$(GO) vet ./...
	git --no-pager diff --check --no-ext-diff --no-textconv HEAD --

verify: check test-e2e test-integration test-runtime

lint:
	$(GO) run github.com/golangci/golangci-lint/cmd/golangci-lint@latest run

security-check:
	GOTOOLCHAIN=$(SECURITY_GOTOOLCHAIN) $(GO) run golang.org/x/vuln/cmd/govulncheck@latest ./...
	GOTOOLCHAIN=$(SECURITY_GOTOOLCHAIN) $(GO) run github.com/securego/gosec/v2/cmd/gosec@latest -quiet ./...

install-hooks:
	git config core.hooksPath .githooks
	chmod +x .githooks/pre-commit

swagger:
	$(GO) run github.com/swaggo/swag/cmd/swag@latest init -g cmd/api/swagger.go -o docs/swagger

docker-swagger:
	$(DOCKER_GO_RUN) sh -c 'go run github.com/swaggo/swag/cmd/swag@latest init -g cmd/api/swagger.go -o docs/swagger'

docker-up:
	$(DOCKER_COMPOSE) up -d postgres redis api worker

docker-infra-up:
	$(DOCKER_COMPOSE) up -d postgres redis

docker-down:
	$(DOCKER_COMPOSE) down

docker-logs:
	$(DOCKER_COMPOSE) logs -f

docker-ps:
	$(DOCKER_COMPOSE) ps

docker-build:
	$(DOCKER_DEV_COMPOSE) build api worker migrate

docker-test:
	$(DOCKER_GO_RUN) sh -c 'go test -v ./internal/...'

docker-test-e2e:
	TEST_ENV_FILE="$(TEST_ENV_FILE)" ./scripts/docker-test-suite.sh e2e

docker-test-integration:
	TEST_ENV_FILE="$(TEST_ENV_FILE)" ./scripts/docker-test-suite.sh integration

docker-test-runtime:
	TEST_ENV_FILE="$(TEST_ENV_FILE)" ./scripts/docker-test-suite.sh runtime

docker-fmt:
	$(DOCKER_GO_RUN) sh -c 'go run mvdan.cc/gofumpt@latest -w . && go run golang.org/x/tools/cmd/goimports@latest -w .'

docker-lint:
	$(DOCKER_GO_RUN) sh -c 'go run github.com/golangci/golangci-lint/cmd/golangci-lint@latest run'

docker-migrate-up:
	$(DOCKER_COMPOSE) run --rm migrate up

docker-migrate-down:
	$(DOCKER_COMPOSE) run --rm migrate down

docker-seed:
	$(DOCKER_COMPOSE) run --rm --no-deps --entrypoint /app/seed api --mode=production

docker-dev-up:
	$(DOCKER_DEV_COMPOSE) up --build -d postgres redis api worker

docker-dev-down:
	$(DOCKER_DEV_COMPOSE) down

docker-dev-reset:
	$(DOCKER_DEV_COMPOSE) down -v

docker-dev-logs:
	$(DOCKER_DEV_COMPOSE) logs -f

docker-dev-ps:
	$(DOCKER_DEV_COMPOSE) ps

docker-dev-migrate-up:
	$(DOCKER_DEV_COMPOSE) run --rm migrate up

docker-dev-migrate-down:
	$(DOCKER_DEV_COMPOSE) run --rm migrate down

docker-dev-seed:
	$(DOCKER_DEV_COMPOSE) run --rm api go run ./cmd/seed
