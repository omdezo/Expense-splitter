.PHONY: help tidy run build up down logs ps health db-reset clean kc-token kc-logs sqlc sqlc-vet

# sqlc runs via Docker (the local Go toolchain is too old to `go install` it).
SQLC_IMAGE := sqlc/sqlc:1.27.0
SQLC_RUN   := docker run --rm -u "$$(id -u):$$(id -g)" -v "$(CURDIR)/server/database":/src -w /src $(SQLC_IMAGE)

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-10s %s\n", $$1, $$2}'

tidy: ## Resolve and tidy Go module dependencies
	cd server && go mod tidy

run: ## Run the server locally (loads server/.env if present; needs a reachable Postgres)
	cd server && set -a && { test -f .env && . ./.env || true; } && set +a && go run .

build: ## Build the server binary into ./server/bin
	cd server && go build -o bin/server .

sqlc: ## Generate type-safe Go from SQL into server/database/repo (sqlc via Docker)
	$(SQLC_RUN) generate

sqlc-vet: ## Lint the SQL queries (sqlc vet via Docker)
	$(SQLC_RUN) vet

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
