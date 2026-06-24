.PHONY: help tidy run build up down logs ps health clean

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-10s %s\n", $$1, $$2}'

tidy: ## Resolve and tidy Go module dependencies
	cd server && go mod tidy

run: ## Run the server locally (loads server/.env if present; needs a reachable Postgres)
	cd server && set -a && { test -f .env && . ./.env || true; } && set +a && go run .

build: ## Build the server binary into ./server/bin
	cd server && go build -o bin/server .

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

clean: ## Stop containers and remove volumes
	docker compose down -v
