# Expense Splitter

Go + Echo server with PostgreSQL, run via Docker Compose. Authentication is
delegated to **Keycloak** (the server only validates tokens); authorization
(per-group roles, global admin) lives in the database.

## Layout

```
.
├── docker-compose.yml
├── keycloak/
│   └── realm-export.json   # auto-imported realm: client + seed users
└── server/
    ├── main.go
    ├── go.mod
    ├── Dockerfile
    ├── config/       # env config (incl. Keycloak)
    ├── database/     # postgres connection (pgx)
    ├── handler/      # health + /me handlers
    ├── middleware/   # Keycloak JWT auth middleware
    └── router/       # echo routes
```

## Run

```bash
docker compose up --build      # or: make up
```

- Server:           http://localhost:8080
- Postgres (app):   localhost:5433 → 5432
- Keycloak:         http://localhost:8081  (admin console: `admin` / `admin`)
- Postgres (KC):    internal only

Keycloak imports its realm on first boot (~30–60s). The app server starts
immediately and fetches signing keys lazily, so startup order doesn't matter.

## Health check

```bash
curl http://localhost:8080/health
# {"status":"ok","database":"up"}
```

## Authentication (Keycloak)

The realm `expense-splitter` is imported from `keycloak/realm-export.json` with:

- a public client **`expense-splitter-api`** (direct-access-grants enabled, with
  an audience mapper so access tokens carry `aud: expense-splitter-api`);
- two seed users — **`admin@expense-splitter.local`** / `admin` (the intended
  default global admin) and **`alice@expense-splitter.local`** / `alice`.

The server validates the token signature against the realm JWKS, plus the
issuer, audience, and expiry. It deliberately fetches keys from the **internal**
address (`keycloak:8080`) while trusting the **issuer** tokens are minted with
(`localhost:8081`) — validating these separately avoids Keycloak's host/container
issuer-mismatch problem. After validation it resolves the token's `email` to a
`users` row (`local_user` is `null` when no row exists yet).

### Get a token and call a protected endpoint

```bash
# Fetch an access token (defaults to the seed admin; override KC_USER / KC_PASS)
make kc-token
# ...or directly:
TOKEN=$(curl -s -X POST \
  http://localhost:8081/realms/expense-splitter/protocol/openid-connect/token \
  -d grant_type=password -d client_id=expense-splitter-api \
  -d username=admin@expense-splitter.local -d password=admin \
  | sed -e 's/.*"access_token":"//' -e 's/".*//')

curl http://localhost:8080/me -H "Authorization: Bearer $TOKEN"
# {"subject":"...","email":"admin@expense-splitter.local","local_user":null,...}
```

Note: access tokens expire after ~5 minutes — re-fetch if you get `invalid token`.

### Configuration

The server reads these env vars (set in `docker-compose.yml` for `make up`, and
in `server/.env` for `make run`):

| Var                  | Purpose                                              |
|----------------------|------------------------------------------------------|
| `KEYCLOAK_JWKS_URL`  | Where signing keys are fetched (must be reachable).   |
| `KEYCLOAK_ISSUERS`   | Comma-separated `iss` values to trust.               |
| `KEYCLOAK_AUDIENCE`  | Required `aud` value (empty disables the aud check).  |

### Custom theme

Keycloak runs from a **custom image** (`keycloak/Dockerfile`) that bakes in our
theme, so every environment gets the same branding. The theme lives in
`keycloak/themes/expense-splitter/` (login / account / email), and the realm is
configured to use it (`loginTheme`/`accountTheme`/`emailTheme` in
`realm-export.json`), with English + Arabic (RTL) enabled.

- The **login** theme sets `parent=keycloak.v2` and brands the *native* form
  **via CSS only** (`resources/css/custom.css` overrides PatternFly-5 color
  variables) plus custom messages (`messages/`). It does **not** override any
  `.ftl` — we inherit Keycloak's page so it looks standard and stays upgrade-safe.
- The **account** theme must use `parent=keycloak.v3` (the React account console
  base); pointing it at the login theme (`keycloak.v2`) breaks the console (401).
- In dev, `docker compose` bind-mounts `./keycloak/themes`, so theme edits show
  on the next page load (no rebuild; `start-dev` disables theme caching).

View the themed login page in a browser:

```
http://localhost:8081/realms/expense-splitter/account
```

(redirects you to the branded login). Append `?ui_locales=ar` to any login URL to
see the Arabic/RTL version.

> Changing `loginTheme`/seed users/clients in `realm-export.json` only takes
> effect on a **fresh** realm import. To re-import, reset Keycloak's DB volume:
> `docker compose rm -sf keycloak keycloak-db && docker volume rm expense-splitter_keycloak_pgdata && docker compose up --build -d keycloak`
