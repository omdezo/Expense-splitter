.PHONY: help tidy run build up down logs ps health db-reset clean kc-token kc-logs sqlc sqlc-vet seed test test-unit test-integration test-e2e test-pkg

# sqlc runs via Docker (the local Go toolchain is too old to `go install` it).
SQLC_IMAGE := sqlc/sqlc:1.27.0
SQLC_RUN   := docker run --rm -u "$$(id -u):$$(id -g)" -v "$(CURDIR)/server/database":/src -w /src $(SQLC_IMAGE)

# Go tests run via the pinned toolchain container (the local Go is too old).
GO_CACHE := $(HOME)/.cache/expense-splitter-go
GO_RUN   := docker run --rm -u "$$(id -u):$$(id -g)" -e HOME=/tmp -e GOCACHE=/cache/build -e GOMODCACHE=/cache/mod -v "$(GO_CACHE)":/cache -v "$(CURDIR)/server":/app -w /app golang:1.25

# Default package selection for `test-pkg` (override on the command line).
PKG ?= ./...

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-10s %s\n", $$1, $$2}'

tidy: ## Resolve and tidy Go module dependencies
	cd server && go mod tidy

run: ## Run the server locally (loads server/.env if present; needs a reachable Postgres)
	cd server && set -a && { test -f .env && . ./.env || true; } && set +a && go run . serve

seed: ## Seed the default global admin (runs inside the running server container)
	docker compose exec server ./server seed

build: ## Build the server binary into ./server/bin
	cd server && go build -o bin/server .

sqlc: ## Generate type-safe Go from SQL into server/database/repo (sqlc via Docker)
	$(SQLC_RUN) generate

sqlc-vet: ## Lint the SQL queries (sqlc vet via Docker)
	$(SQLC_RUN) vet

$(GO_CACHE):
	@mkdir -p $@

test: | $(GO_CACHE) ## Run all Go tests (via Docker)
	$(GO_RUN) go test ./...

test-unit: | $(GO_CACHE) ## Run only the fast co-located unit tests (no DB/HTTP)
	$(GO_RUN) sh -c 'go test $$(go list ./... | grep -v /test/)'

test-integration: | $(GO_CACHE) ## Run DB-backed integration tests (needs the stack up)
	$(GO_RUN) sh -c 'pkgs=$$(go list ./test/integration/... 2>/dev/null); [ -n "$$pkgs" ] && go test $$pkgs || echo "no integration tests yet"'

test-e2e: | $(GO_CACHE) ## Run end-to-end tests (needs the stack up)
	$(GO_RUN) sh -c 'pkgs=$$(go list ./test/e2e/... 2>/dev/null); [ -n "$$pkgs" ] && go test $$pkgs || echo "no e2e tests yet"'

test-pkg: | $(GO_CACHE) ## Run one area's tests verbosely (e.g. make test-pkg PKG=./services/ RUN=DecideGroupRole)
	$(GO_RUN) go test -v $(PKG) $(if $(RUN),-run $(RUN))

up: ## Start db + server via docker compose
	docker compose up --build -d

down: ## Stop and remove containers
	docker compose down

logs: ## Tail container logs
	docker compose logs -f

ps: ## Show running services
	docker compose ps

health: ## Curl the health endpoint
	curl -s http://localhost:8080/health

kc-token: ## Fetch a Keycloak access token (override KC_USER / KC_PASS; defaults to the seed admin)
	@curl -s -X POST http://localhost:8081/realms/expense-splitter/protocol/openid-connect/token \
		-d grant_type=password \
		-d client_id=expense-splitter-api \
		-d username=$${KC_USER:-admin@expense-splitter.local} \
		-d password=$${KC_PASS:-admin} | sed -e 's/.*"access_token":"//' -e 's/".*//'

kc-logs: ## Tail Keycloak container logs
	docker compose logs -f keycloak

db-reset: ## Drop the db volume and recreate (migrations re-run on server start; DESTROYS all data)
	docker compose down -v && docker compose up --build -d

clean: ## Stop containers and remove volumes
	docker compose down -v
