#!/bin/bash
# Seeds a complete 2-member demo scenario against the running stack:
# payer (group-admin, paid everything) + debtor, group closed, ONE pending
# payment ready for the proof -> confirm -> finalize flow.
set -euo pipefail
B=${BASE_URL:-http://localhost:8080}

get() { python3 -c "import sys,json
d=json.load(sys.stdin)
print(d.get('$1',''))"; }

ADMIN=$(curl -s -X POST $B/auth/login -H 'Content-Type: application/json' \
  -d '{"email":"admin@expense-splitter.local","password":"admin"}' | get access_token)
if [ -z "$ADMIN" ]; then
  echo "could not log in as the seed admin — is the stack up and seeded?" >&2
  echo "  docker compose up -d && docker compose exec server ./server seed" >&2
  exit 1
fi

TS=$(date +%s)
PAYER_EMAIL="payer$TS@demo.local"
DEBTOR_EMAIL="debtor$TS@demo.local"
PASS="password123"

PAYER=$(curl -s -X POST $B/auth/register -H 'Content-Type: application/json' \
  -d "{\"email\":\"$PAYER_EMAIL\",\"password\":\"$PASS\",\"name\":\"Demo Payer\"}" | get id)
DEBTOR=$(curl -s -X POST $B/auth/register -H 'Content-Type: application/json' \
  -d "{\"email\":\"$DEBTOR_EMAIL\",\"password\":\"$PASS\",\"name\":\"Demo Debtor\"}" | get id)
curl -s -X POST $B/admin/users/$PAYER/approve  -H "Authorization: Bearer $ADMIN" >/dev/null
curl -s -X POST $B/admin/users/$DEBTOR/approve -H "Authorization: Bearer $ADMIN" >/dev/null

TP=$(curl -s -X POST $B/auth/login -H 'Content-Type: application/json' \
  -d "{\"email\":\"$PAYER_EMAIL\",\"password\":\"$PASS\"}" | get access_token)
TD=$(curl -s -X POST $B/auth/login -H 'Content-Type: application/json' \
  -d "{\"email\":\"$DEBTOR_EMAIL\",\"password\":\"$PASS\"}" | get access_token)

G=$(curl -s -X POST $B/groups -H "Authorization: Bearer $TP" -H 'Content-Type: application/json' \
  -d '{"name":"Demo Trip","start_date":"2026-07-01T00:00:00Z","end_date":"2026-07-10T00:00:00Z"}')
GID=$(echo "$G" | get id)
INVITE=$(echo "$G" | get invite_token)

curl -s -X POST $B/groups/join -H "Authorization: Bearer $TD" -H 'Content-Type: application/json' \
  -d "{\"invite_token\":\"$INVITE\"}" >/dev/null
curl -s -X POST $B/groups/$GID/members/$DEBTOR/approve -H "Authorization: Bearer $TP" >/dev/null
curl -s -X POST $B/groups/$GID/expenses -H "Authorization: Bearer $TP" -H 'Content-Type: application/json' \
  -d '{"amount_baisa":10000,"category":"food","description":"demo dinner","occurred_on":"2026-07-02"}' >/dev/null
curl -s -X POST $B/groups/$GID/close -H "Authorization: Bearer $TP" >/dev/null

PAY=$(curl -s $B/groups/$GID/settlement -H "Authorization: Bearer $TP" | python3 -c 'import sys,json
d=json.load(sys.stdin)
print(d["payments"][0]["id"] if d["payments"] else "")')

cat <<SUMMARY

Demo scenario ready
===================
group      : $GID  ("Demo Trip", closed)
payment    : $PAY  (5.000 OMR, pending)

people (password for both: $PASS)
  payer / group-admin / CREDITOR : $PAYER_EMAIL
  member            /  DEBTOR   : $DEBTOR_EMAIL

Two-key flow in Postman
  1. Login as  $DEBTOR_EMAIL  -> Submit proof
  2. Login as  $PAYER_EMAIL   -> Confirm receipt, then Finalize
Set collection vars: groupId=$GID  paymentId=$PAY
(or run "Get settlement plan" after logging in — it captures paymentId itself)
SUMMARY
