# Postman collection

`expense-splitter.postman_collection.json` — every current API endpoint (22 requests, 7 folders).

## Import
Postman → **Import** → select the file. Everything lives in one collection; variables are stored on the collection itself (no separate environment needed).

## Run order (happy path)
1. Start the stack: `docker compose up --build -d` (server → http://localhost:8080).
2. **Auth → Login** — stores `access_token`; all authed requests inherit it via collection-level Bearer auth.
3. **Account → Get me** — stores your `userId`.
4. **Groups → Create group** — stores `groupId` and `inviteToken`.
5. **Expenses → Record expense** — stores `expenseId`.
6. Explore the rest (members, list/update, close, admin).

## Variables (collection → Variables tab)
| var | default | notes |
| --- | --- | --- |
| `baseUrl` | `http://localhost:8080` | the API (login is proxied, so you never hit Keycloak directly) |
| `email` / `password` | `admin@expense-splitter.local` / `admin` | seed global admin |
| `newUserEmail` / `newUserPassword` / `newUserName` | `testuser@example.com` / `password123` / `Test User` | used by Auth → Register (password ≥ 8 chars) |
| `access_token`, `groupId`, `userId`, `expenseId`, `inviteToken` | — | auto-filled by test scripts; can be set manually |

## Notes
- Public (no token): `GET /health`, `POST /auth/register`, `POST /auth/login`. Everything else needs a valid token — run **Login** first.
- List-expenses filters (`category`, `paid_by`, `q`) are added but **disabled** by default — enable them in that request's Params tab.
- Dates: group `start_date`/`end_date` are RFC3339 timestamps; expense `occurred_on` is `YYYY-MM-DD` and must fall within the trip range.
- Admin endpoints require the caller to be the global admin (log in as the seed admin).
