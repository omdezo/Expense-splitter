# Postman collection

`expense-splitter.postman_collection.json` — **every API endpoint** (44 requests, 10 folders, ordered like a real session).

## Import
Postman → **Import** → select the file (replace the old copy if prompted). No separate environment needed — variables live on the collection.

## Folders (run top to bottom)
| # | Folder | What's inside |
|---|--------|----------------|
| 1 | Public / Auth | health, sign-up, **Login** (stores the token), public share-token status |
| 2 | Account | me, link token→local row, submit verification |
| 3 | Admin — Users | list (+status filter), detail with memberships, approve/reject, delete |
| 4 | Admin — Groups | list all groups, delete pristine group |
| 5 | Groups | list mine, create (user / admin-assign), details, update, close |
| 6 | Membership | join by invite, requests, approve/reject/promote, remove/leave |
| 7 | Expenses | record (equal + weighted), list w/ filters, update, delete |
| 8 | Settlement | summary, plan (+"N of M settled"), PDF report |
| 9 | Payments & Proofs | text + image proof, proof metadata/bytes, confirm/dispute/finalize/reject |
| 10 | Ops | audit log, idempotent nudges |

## Happy-path order (as the seed admin)
1. **1 → Login** (defaults: `admin@expense-splitter.local` / `admin`)
2. **1 → Register** — creates a member, stores `memberUserId` (unique email every run)
3. **3 → Approve user** — verifies that member
4. **5 → Create group (as global admin — assign member)** — stores `groupId` + `inviteToken`
5. Continue with membership, expenses, close, then folder 9 for the two-key payment flow

To act as a *member* instead: set the collection vars `email`/`password` to a registered user's and re-run Login.

## Auto-chained variables
`access_token` (Login) · `memberUserId` (Register / List join requests) · `userId` (Get me) ·
`groupId` + `inviteToken` (Create group) · `statusToken` (Get group) · `expenseId` (Record expense) ·
`paymentId` (Settlement plan). All editable per-request in the **Params** tab.

## Notes
- Tokens expire in ~5 minutes — re-run **Login** on 401s.
- Image proof upload: pick a real image file in the request's Body tab (magic-byte validated; max 5 MiB).
- The PDF report: use **Send and Download**.
- After a DB wipe (`make db-reset`): `docker compose exec server ./server seed`, then admin **Login** + **2 → Link token**.
