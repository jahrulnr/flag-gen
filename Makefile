SERVICE_VERSION := 0.0.1
SERVICE_NAME := flag-gen
# Bash for recipes (dev trap, [[ ]], etc.)
SHELL := /usr/bin/env bash

# Angular `ng serve` port (see web/package.json). Used so Go can proxy /app in dev.
DASHBOARD_DEV_SSR_URL ?= http://127.0.0.1:6124

##@ Development

# Load .env without shell-evaluating values (MYSQL_DSN tcp(...) and & are safe).
define LOAD_DOTENV
. "$(CURDIR)/scripts/load-dotenv.sh" "$(CURDIR)/.env"
endef

define DEV_ENV
$(LOAD_DOTENV) && export DASHBOARD_SSR_URL="$(DASHBOARD_DEV_SSR_URL)"
endef

.PHONY: dev
dev: ## Run Go API + Angular in parallel (same idea as make -j2 backend/frontend)
	@$(MAKE) -j2 dev-backend dev-frontend

.PHONY: dev-backend
dev-backend: ## Go API (:APP_PORT); loads .env; DASHBOARD_SSR_URL → Angular dev server
	cd "$(CURDIR)" && $(DEV_ENV) && go run ./cmd/app

.PHONY: dev-frontend
dev-frontend: ## Angular dashboard: npm install then npm run dev (:6124, /app/)
	cd "$(CURDIR)/web" && npm install && npm run dev

.PHONY: dev-api
dev-api: ## Go API only (same env as dev-backend)
	cd "$(CURDIR)" && $(DEV_ENV) && go run ./cmd/app

.PHONY: dev-web
dev-web: ## Angular only (no npm install)
	cd "$(CURDIR)/web" && npm run dev

##@ Database

.PHONY: migrate
migrate: ## Apply pending MySQL migrations (loads .env; empty users → dev admin admin@local / admin — see .env.example)
	cd "$(CURDIR)" && $(LOAD_DOTENV) && go run ./cmd/migrate

.PHONY: seed
seed: ## Idempotent seed: bootstrap project/env when empty, then demo segments + flags (loads .env; SEED_*)
	cd "$(CURDIR)" && $(LOAD_DOTENV) && go run ./cmd/seed

##@ Project

.PHONY: help
help: ## Display this help screen
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

.PHONY: build
build: ### Build app for cicd
	mkdir -p bin
	env GOOS=linux CGO_ENABLED=0 GO111MODULE=on GOOS=linux GOARCH=arm64 go build -o bin/main-arm64 ./cmd/app
	env GOOS=linux CGO_ENABLED=0 GO111MODULE=on GOOS=linux GOARCH=amd64 go build -o bin/main-amd64 ./cmd/app

unit-test-coverage: ### Run unit tests with coverage
	# Run tests sequentially (-p=1) to avoid intermittent parallel failures and
	# produce a combined coverage report.
	go clean -testcache && go test -p=1 -v -count=1 -covermode=count -coverprofile=coverage.cov `go list ./... | grep -v cmd | grep -v "mock" | grep -v "nft" | grep -v "tmp" | grep -v "pkg/broker" | grep -v "infrastructure/di" | grep -v "config"`
	go tool cover -func=coverage.cov

.PHONY: lint
lint: ### Run golangci-lint for code quality checks
	golangci-lint run --timeout=5m

.PHONY: gosec
gosec: ### Run gosec for security vulnerability scanning
	gosec ./...

.PHONY: quality
quality: unit-test-coverage lint gosec ### Run comprehensive quality checks: tests, linting, and security scanning

.PHONY: swagger
swagger: ### Generate Swagger documentation
	swag init -g cmd/app/main.go -o docs --parseDepth 3 --quiet

.PHONY: wire
wire: ### Generate dependency injection code using Google Wire
	wire ./infrastructure/di

.PHONY: wire-install
wire-install: ### Install Google Wire tool
	go install github.com/swaggo/swag/cmd/swag@v1.8.12

.PHONY: deps
deps: wire-install ### Install all development dependencies

.PHONY: clean-wire
clean-wire: ### Clean generated Wire files
	rm -f infrastructure/di/wire_gen.go
