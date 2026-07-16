.PHONY: help tidy run build up down logs ps health db-reset clean kc-token kc-logs sqlc sqlc-vet swagger seed up-prod down-prod logs-prod config-prod seed-prod test test-unit test-integration test-e2e test-pkg test-file

# Production compose = base + prod overlay (dev uses the auto-applied override).
COMPOSE_PROD := docker compose -f docker-compose.yml -f docker-compose.prod.yml

# swag (OpenAPI generator) is pinned so regenerated specs stay reproducible.
SWAG_VERSION := v1.16.6

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

seed: ## Seed the default global admin on the DEV stack
	docker compose exec server ./server seed

seed-prod: ## Seed the global admin on the PRODUCTION stack (GLOBAL_ADMIN_EMAIL from .env)
	$(COMPOSE_PROD) exec server ./server seed

build: ## Build the server binary into ./server/bin
	cd server && go build -o bin/server .

sqlc: ## Generate type-safe Go from SQL into server/database/repo (sqlc via Docker)
	$(SQLC_RUN) generate

sqlc-vet: ## Lint the SQL queries (sqlc vet via Docker)
	$(SQLC_RUN) vet

swagger: | $(GO_CACHE) ## Regenerate the OpenAPI spec into server/docs from the handler annotations
	$(GO_RUN) go run github.com/swaggo/swag/cmd/swag@$(SWAG_VERSION) init -g main.go -o docs --parseDependency --parseInternal

$(GO_CACHE):
	@mkdir -p $@

test: | $(GO_CACHE) ## Run all Go tests (via Docker)
	$(GO_RUN) go test ./...

test-unit: | $(GO_CACHE) ## Run only the fast co-located unit tests (no DB/HTTP)
	$(GO_RUN) sh -c 'go test $$(go list ./... | grep -v /test/)'

test-integration: | $(GO_CACHE) ## Run DB-backed integration tests (needs the stack up)
	$(GO_RUN) sh -c 'pkgs=$$(go list ./test/integration/... 2>/dev/null); [ -n "$$pkgs" ] && go test $$pkgs || echo "no integration tests yet"'

test-e2e: | $(GO_CACHE) ## Run the full API end-to-end suite against the running stack (make up first)
	docker run --rm --network host -u "$$(id -u):$$(id -g)" -e HOME=/tmp -e GOCACHE=/cache/build -e GOMODCACHE=/cache/mod -e E2E_BASE_URL=$${E2E_BASE_URL:-http://localhost:8080} -v "$(GO_CACHE)":/cache -v "$(CURDIR)/server":/app -w /app golang:1.25 go test -v -count=1 ./test/e2e/...

test-pkg: | $(GO_CACHE) ## Run one area's tests verbosely (e.g. make test-pkg PKG=./services/ RUN=DecideGroupRole)
	$(GO_RUN) go test -v $(PKG) $(if $(RUN),-run '$(RUN)')

test-file: | $(GO_CACHE) ## Run only the tests defined in one file (FILE=services/settlement_test.go)
	@test -n "$(FILE)" || { echo "usage: make test-file FILE=services/<name>_test.go"; exit 1; }
	@f="$(FILE)"; f=$${f#server/}; \
		names=$$(grep -oE '^func Test[[:alnum:]_]+' "server/$$f" | awk '{print $$2}' | tr '\n' '|' | sed 's/|$$//'); \
		test -n "$$names" || { echo "no Test functions found in $$f"; exit 1; }; \
		dir=$$(dirname "$$f"); \
		echo ">> $$f  ->  -run '$$names'"; \
		$(GO_RUN) go test -v "./$$dir/" -run "$$names"

up: ## Start the DEV stack (docker-compose.override.yml is applied automatically)
	docker compose up --build -d

down: ## Stop and remove containers
	docker compose down

logs: ## Tail container logs
	docker compose logs -f

up-prod: ## Start the PRODUCTION stack (needs .env — see .env.prod.example)
	$(COMPOSE_PROD) up --build -d

down-prod: ## Stop the production stack
	$(COMPOSE_PROD) down

logs-prod: ## Tail production logs
	$(COMPOSE_PROD) logs -f

config-prod: ## Render + validate the production config (catches missing secrets)
	$(COMPOSE_PROD) config

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

demo: ## Seed a 2-member demo group (closed, one pending payment) and print all the ids
	@bash scripts/demo.sh

kc-logs: ## Tail Keycloak container logs
	docker compose logs -f keycloak

db-reset: ## Drop the db volume and recreate (migrations re-run on server start; DESTROYS all data)
	docker compose down -v && docker compose up --build -d

clean: ## Stop containers and remove volumes
	docker compose down -v
