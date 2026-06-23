# Expense Splitter

Go + Echo server with PostgreSQL, run via Docker Compose. Just a skeleton with a health check for now.

## Layout

```
.
├── docker-compose.yml
└── server/
    ├── main.go
    ├── go.mod
    ├── Dockerfile
    ├── config/      # env config
    ├── database/    # postgres connection (pgx)
    ├── handler/     # health handler
    └── router/      # echo routes
```

## Run

```bash
docker compose up --build
```

- Server: http://localhost:8080
- Postgres: localhost:5433 → 5432

## Health check

```bash
curl http://localhost:8080/health
# {"status":"ok","database":"up"}
```
