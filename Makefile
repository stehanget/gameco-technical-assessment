GO ?= go
COMPOSE_FILE ?= deployments/compose/docker-compose.yml
DEV_COMPOSE_FILE ?= deployments/compose/docker-compose.dev.yml
DOCKER_GO_IMAGE ?= golang:1.25
DOCKER_WORKDIR ?= /app

.PHONY: run worker seed migrate-up migrate-down build fmt fmt-check tidy test test-e2e lint install-hooks swagger docker-swagger docker-up docker-dev-up docker-dev-down docker-dev-reset docker-down docker-logs docker-ps docker-migrate-up docker-migrate-down docker-build docker-test docker-test-e2e docker-fmt docker-lint docker-seed

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
	$(GO) test ./...

test-e2e:
	$(GO) test ./tests/e2e -count=1

lint:
	$(GO) run github.com/golangci/golangci-lint/cmd/golangci-lint@latest run

install-hooks:
	git config core.hooksPath .githooks
	chmod +x .githooks/pre-commit

swagger:
	$(GO) run github.com/swaggo/swag/cmd/swag@latest init -g cmd/api/swagger.go -o docs/swagger

docker-swagger:
	docker run --rm -v "$(CURDIR):$(DOCKER_WORKDIR)" -w $(DOCKER_WORKDIR) $(DOCKER_GO_IMAGE) sh -lc 'go run github.com/swaggo/swag/cmd/swag@latest init -g cmd/api/swagger.go -o docs/swagger'

docker-build:
	docker compose -f $(COMPOSE_FILE) build api worker migrate

docker-up:
	docker compose -f $(COMPOSE_FILE) up -d postgres redis

docker-dev-up:
	docker compose -f $(COMPOSE_FILE) -f $(DEV_COMPOSE_FILE) up --build -d postgres redis api worker

docker-dev-down:
	docker compose -f $(COMPOSE_FILE) -f $(DEV_COMPOSE_FILE) down

docker-dev-reset:
	docker compose -f $(COMPOSE_FILE) -f $(DEV_COMPOSE_FILE) down -v

docker-ps:
	docker compose -f $(COMPOSE_FILE) -f $(DEV_COMPOSE_FILE) ps

docker-migrate-up:
	docker compose -f $(COMPOSE_FILE) run --rm migrate up

docker-migrate-down:
	docker compose -f $(COMPOSE_FILE) run --rm migrate down

docker-test:
	docker run --rm -v "$(CURDIR):$(DOCKER_WORKDIR)" -w $(DOCKER_WORKDIR) $(DOCKER_GO_IMAGE) sh -lc 'go test ./...'

docker-test-e2e:
	docker run --rm -v "$(CURDIR):$(DOCKER_WORKDIR)" -w $(DOCKER_WORKDIR) $(DOCKER_GO_IMAGE) sh -lc 'go test ./tests/e2e -count=1'

docker-fmt:
	docker run --rm -v "$(CURDIR):$(DOCKER_WORKDIR)" -w $(DOCKER_WORKDIR) $(DOCKER_GO_IMAGE) sh -lc 'go run mvdan.cc/gofumpt@latest -w . && go run golang.org/x/tools/cmd/goimports@latest -w .'

docker-lint:
	docker run --rm -v "$(CURDIR):$(DOCKER_WORKDIR)" -w $(DOCKER_WORKDIR) $(DOCKER_GO_IMAGE) sh -lc 'go run github.com/golangci/golangci-lint/cmd/golangci-lint@latest run'

docker-seed:
	docker compose -f $(COMPOSE_FILE) -f $(DEV_COMPOSE_FILE) run --rm api go run ./cmd/seed

docker-down:
	docker compose -f $(COMPOSE_FILE) down

docker-logs:
	docker compose -f $(COMPOSE_FILE) -f $(DEV_COMPOSE_FILE) logs -f api worker postgres redis
