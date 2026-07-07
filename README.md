# Expense Splitter

Go + Echo server with PostgreSQL, run via Docker Compose. Authentication is
delegated to **Keycloak** (the server only validates tokens); authorization
(per-group roles, global admin) lives in the database.

## Layout

```
.
├── docker-compose.yml
├── postman/                # importable Postman collection for every endpoint
├── keycloak/
│   ├── Dockerfile          # stock Keycloak + custom theme baked in
│   ├── realm-export.json   # auto-imported realm: client + seed users
│   └── themes/             # login/account/email theme (en + ar)
└── server/
    ├── main.go
    ├── Dockerfile
    ├── cmd/          # cobra: serve, seed
    ├── config/       # env config (incl. Keycloak)
    ├── database/
    │   ├── migration/schema/   # goose migrations (run at startup)
    │   ├── queries/            # ALL SQL lives here (sqlc input)
    │   ├── repo/               # sqlc-generated typed queries (do not edit)
    │   └── seeding/            # default global admin seeder
    ├── keycloak/     # server-side Keycloak client (login + admin user create)
    ├── handler/      # thin HTTP handlers
    ├── middleware/   # Keycloak JWT auth middleware
    ├── router/       # echo routes
    ├── services/     # business logic, authorization, settlement math
    └── types/        # shared domain types, API errors
```

## API

Public (no token): `POST /auth/register`, `POST /auth/login`,
`GET /public/groups/:token` (share-token status), `GET /health`.
Everything else requires `Authorization: Bearer <token>` from `/auth/login`.

| Area | Endpoints |
|---|---|
| Account | `GET /me` · `POST /register` (link token→local row) · `POST /verification` |
| Admin | `POST /admin/users/:id/approve` · `POST /admin/users/:id/reject` |
| Groups | `GET /groups` · `POST /groups` · `GET /groups/:id` · `PATCH /groups/:id` · `POST /groups/:id/close` |
| Membership | `POST /groups/join` · `GET /groups/:id/requests` · `POST /groups/:id/members/:userId/approve\|reject\|promote` · `DELETE /groups/:id/members/:userId` |
| Expenses | `POST /groups/:id/expenses` · `GET /groups/:id/expenses?category=&paid_by=&q=` · `PATCH /groups/:id/expenses/:expenseId` · `DELETE /groups/:id/expenses/:expenseId` |
| Settlement | `GET /groups/:id/summary` · `GET /groups/:id/settlement` (plan + "N of M settled") · `GET /groups/:id/report.pdf` (fully-settled only) |
| Payments (two-key) | `POST /payments/:id/proof` (debtor) · `/confirm` `/dispute` (creditor) · `/finalize` `/reject` (admin) |
| Audit | `GET /groups/:id/audit` (group-admin/global-admin) |

The `postman/` collection covers all of these with auto-chained variables —
run **Login** first, then top to bottom.

## Testing

```bash
make test          # all unit tests (settlement math, truncation, authz, tamper suite)
make test-pkg PKG=./services RUN='TestTamper'   # a specific area, verbose
```

Unit tests are co-located with the code; the settlement algorithm, the
80-char truncation rule, the authorization matrix, and the tamper-resistance
claims are all covered. For end-to-end verification use the Postman
collection or `make kc-token` + curl against the running stack.

## Run

```bash
docker compose up --build      # or: make up
```

- Server:           http://localhost:8080
- Postgres (app):   localhost:5433 → 5432
- Keycloak:         http://localhost:8081  (admin console: `admin` / `admin`)
- Postgres (KC):    internal only
- MinIO (proofs):   http://localhost:9001 console (`minioadmin` / `minioadmin`)

Proof images live in MinIO (S3-compatible); the DB stores only metadata plus a
sha256, so a swapped file is detectable. Uploads are validated by **magic
bytes** (jpeg/png/gif/webp), never by extension.

Keycloak imports its realm on first boot (~30–60s). The app server starts
immediately and fetches signing keys lazily, so startup order doesn't matter.

## Health check

```bash
curl http://localhost:8080/health
# {"status":"ok","database":"up"}
```

## Settlement algorithm

All money is integer **baisa** (`1.000 OMR = 1000`); there are no floats anywhere
in the money path. Settlement runs when the group-admin closes the group and has
two stages, both pure functions in `server/services/settlement.go`.

### 1. Fair share with deterministic remainder — `fairShares`

`total / n` rarely divides evenly, and the leftover baisa have to go *somewhere*
— deterministically, so the same input always produces the same plan.

- base share = `total / n` (integer division)
- remainder `r = total % n` (always `0 ≤ r < n`)
- the **first `r` members in stable order (sorted by `user_id`)** each absorb
  exactly **one extra baisa**

Example: `1000` baisa across 3 members → `334, 333, 333`. The shares always sum
to exactly `total` — no baisa is created or lost. Tested in `TestFairShares`
(same input twice → identical output).

### 2. Payment plan — greedy **largest-debtor ↔ largest-creditor matching**

After fair shares, each member has a net balance (`paid − fair_share`);
negative = owes, positive = is owed. The plan generator:

1. Partition members into **debtors** (net < 0) and **creditors** (net > 0);
   zeros drop out.
2. Sort debtors by most-negative net, creditors by most-positive net
   (ties broken by `user_id` → determinism).
3. Two pointers: transfer `min(|debtor.net|, creditor.net)` from the current
   largest debtor to the current largest creditor; whoever reaches zero
   advances their pointer. Repeat until both lists are exhausted.

**Transfer-count bound:** every transfer fully zeroes at least one participant,
so the plan has at most `D + C − 1 ≤ n − 1` transfers — versus the naive
`n × (n−1)` everyone-pays-everyone matrix.

**Complexity:** `O(n log n)` (the two sorts) + `O(n)` for the matching loop →
**`O(n log n)` time, `O(n)` space**.

**Why a heuristic:** the true *minimum* number of transfers is **NP-hard**
(finding subsets of debtors and creditors that cancel exactly reduces to
subset-sum). The greedy bound of `≤ n − 1` transfers with exact reconciliation
in integer baisa is the standard, defensible trade-off.

Reference case from the spec (`TestComputePlanReferenceCase`): shares 60 each,
Ahmed +40, Omar +20, Mohammed −60 → **Mohammed→Ahmed 40, Mohammed→Omar 20**
(2 transfers, reconciles to zero). `TestComputePlanReconciles` additionally
proves `sum(out) == sum(in)` and all nets reach zero on every case, including
the **multi-debtor / multi-creditor** shape (`+30, +10, −25, −15`) that the
single-debtor example hides.

```bash
make test-pkg PKG=./services RUN='TestFairShares|TestComputePlan'   # see it run
```

## Tamper resistance (bonus 1)

Every payment-state transition funnels through one pure function,
`validatePaymentTransition` (`server/services/payments.go`), and
`server/services/tamper_test.go` proves the three integrity claims by
**enumerating the full (role-combination × action × status) space**, not just
happy paths:

1. **No path lets a debtor mark their own payment settled** — including a
   debtor who is also the group-admin (`TestTamperClaim1DebtorCannotSelfSettle`).
2. **Settled requires both keys**: the only transition producing `settled` is
   an admin's finalize from `creditor_confirmed`, and the only transition
   producing `creditor_confirmed` is the creditor's confirm from
   `proof_submitted` (`TestTamperClaim2TwoKeysRequired`).
3. **A disputed payment cannot be reported settled** — it is only resolvable by
   the debtor re-submitting proof and both keys running again
   (`TestTamperClaim3DisputedNeverSettled`).

The one sanctioned exception — the global admin's override (req #15) — is
pinned by `TestTamperGlobalOverrideIsBounded`: they may finalize or reject any
non-settled payment, and can do **nothing else** (no debtor/creditor keys, and
`settled` stays terminal even for them).

```bash
make test-pkg PKG=./services RUN='TestTamper'
```

## Concurrency & locking strategy

The dangerous window is settlement: it must be computed **exactly once over a
consistent snapshot** while expenses may be landing concurrently.

| Operation | Lock taken | Race it kills |
|---|---|---|
| **Close group** (`services/close.go`) | one tx; `SELECT … FOR UPDATE` on the group row, then the status guard re-checked *under the lock* | two concurrent closes — the second blocks, then sees `status=closed` → 409 |
| **Record / update expense** (`services/expenses.go`) | one tx; `SELECT … FOR SHARE` on the group row | an expense landing *during* a close. `FOR SHARE` is compatible with other `FOR SHARE` (members record concurrently, no bottleneck) but conflicts with close's `FOR UPDATE` — so an in-flight close blocks new expenses until it commits, after which they see `closed` → 409. Nothing slips past the snapshot. |
| **Settle-once backstop** | `settlement_runs.group_id UNIQUE` | even if application logic ever regressed, a second settlement row is a constraint violation — the DB is the last line of defense |
| **Admin handoff** (`services/memberships.go`) | demote-then-promote inside one tx, under the `one_group_admin_per_group` partial unique index | two admins existing for any instant — the index makes it impossible, the ordering makes the tx valid |
| **Group update** | `UPDATE … WHERE status='open'` with `RETURNING`; zero rows → 409 | lost-update against a concurrent close |
| **Finalize payment** *(lands with the confirmation state machine)* | optimistic locking via `payments.version`: `UPDATE … SET status=…, version=version+1 WHERE id=$1 AND version=$2` — zero rows affected means someone else finalized first → 409 | two admins finalizing the same payment at once; each payment is finalized exactly once |

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

## Deployment

The whole stack is one `docker compose up --build -d` on any Docker host
(a VM on DigitalOcean/Hetzner/EC2, etc.):

```bash
git clone https://github.com/omdezo/Expense-splitter && cd Expense-splitter
docker compose up --build -d
docker compose exec server ./server seed     # create the default global admin
```

Before exposing it publicly, change the dev defaults in `docker-compose.yml`:

1. **Secrets** — replace every `admin`/`postgres`/`keycloak` password
   (`KC_BOOTSTRAP_ADMIN_PASSWORD`, `KEYCLOAK_ADMIN_PASSWORD`, both Postgres
   credentials) with real values, ideally via an `.env` file.
2. **Keycloak in prod mode** — swap `start-dev --import-realm` for
   `start --import-realm` with `KC_HOSTNAME=<your-domain>` and TLS in front
   (Keycloak refuses plain-HTTP admin access in prod mode).
3. **Issuers** — set `KEYCLOAK_ISSUERS` to the public URL tokens will be
   minted from (replacing `http://localhost:8081/...`).
4. **Ports** — keep Postgres unpublished (drop the `5433` mapping) and put the
   API + Keycloak behind a reverse proxy with TLS.

Data lives in the named volumes `pgdata` (app) and `keycloak_pgdata`
(identity); after a volume wipe, re-run the seed and have the admin call
`POST /register` once to re-link their Keycloak identity.
