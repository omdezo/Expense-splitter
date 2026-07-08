package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

// End-to-end suite: drives EVERY endpoint against the running stack
// (docker compose up + seed). Skips itself when the server is unreachable.
//   make test-e2e

var base = func() string {
	if v := os.Getenv("E2E_BASE_URL"); v != "" {
		return v
	}
	return "http://localhost:8080"
}()

var httpc = &http.Client{Timeout: 20 * time.Second}

// actors maps tokens to readable names so every logged call shows WHO made it.
var actors = map[string]string{}

func actor(token string) string {
	if token == "" {
		return "public"
	}
	if n, ok := actors[token]; ok {
		return n
	}
	return "unknown"
}

// snippet compacts a response body to one readable log line.
func snippet(raw []byte) string {
	s := strings.Join(strings.Fields(string(raw)), " ")
	r := []rune(s)
	if len(r) > 220 {
		return string(r[:220]) + " …"
	}
	if s == "" {
		return "(empty body)"
	}
	return s
}

func call(t *testing.T, method, path, token string, body any) (int, map[string]any, []byte) {
	t.Helper()
	var rd io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		rd = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, base+path, rd)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := httpc.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	m := map[string]any{}
	_ = json.Unmarshal(raw, &m)
	t.Logf("%s %s  [as %s]", method, path, actor(token))
	t.Logf("   -> %d  %s", resp.StatusCode, snippet(raw))
	return resp.StatusCode, m, raw
}

func want(t *testing.T, got, exp int, what string, payload map[string]any) {
	t.Helper()
	if got != exp {
		t.Fatalf("   FAIL %s: expected status %d, got %d (%v)", what, exp, got, payload)
	}
	t.Logf("   OK   %s: status %d as expected", what, exp)
}

func str(m map[string]any, k string) string {
	s, _ := m[k].(string)
	return s
}

func items(t *testing.T, m map[string]any, what string) []any {
	t.Helper()
	arr, ok := m["items"].([]any)
	if !ok {
		t.Fatalf("%s: no items[] in page envelope (%v)", what, m)
	}
	return arr
}

func login(t *testing.T, email, password string) (string, string) {
	t.Helper()
	code, m, _ := call(t, "POST", "/auth/login", "", map[string]string{"email": email, "password": password})
	want(t, code, 200, "login "+email, m)
	tok := str(m, "access_token")
	actors[tok] = strings.SplitN(email, "@", 2)[0]
	return tok, str(m, "refresh_token")
}

func register(t *testing.T, admin, email, name string) string {
	t.Helper()
	code, m, _ := call(t, "POST", "/auth/register", "", map[string]string{"email": email, "password": "password123", "name": name})
	want(t, code, 201, "register "+email, m)
	id := str(m, "id")
	code, m, _ = call(t, "POST", "/admin/users/"+id+"/approve", admin, nil)
	want(t, code, 200, "verify "+email, m)
	return id
}

func tinyPNG(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 2, 2))); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	return buf.Bytes()
}

func uploadProof(t *testing.T, token, paymentID string, file []byte) (int, map[string]any) {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	fw, _ := w.CreateFormFile("image", "receipt.png")
	fw.Write(file)
	w.Close()
	req, _ := http.NewRequest("POST", base+"/payments/"+paymentID+"/proof", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := httpc.Do(req)
	if err != nil {
		t.Fatalf("upload proof: %v", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	m := map[string]any{}
	_ = json.Unmarshal(raw, &m)
	t.Logf("POST /payments/%s/proof  [as %s]  (multipart image, %d bytes)", paymentID, actor(token), len(file))
	t.Logf("   -> %d  %s", resp.StatusCode, snippet(raw))
	return resp.StatusCode, m
}

func TestEndToEnd(t *testing.T) {
	if code, _, _ := func() (int, map[string]any, []byte) {
		resp, err := httpc.Get(base + "/health")
		if err != nil {
			return 0, nil, nil
		}
		defer resp.Body.Close()
		return resp.StatusCode, nil, nil
	}(); code != 200 {
		t.Skipf("stack not running at %s — start it with: docker compose up -d && docker compose exec server ./server seed", base)
	}

	ts := time.Now().UnixNano()
	payerEmail := fmt.Sprintf("e2e.payer.%d@test.local", ts)
	debtorEmail := fmt.Sprintf("e2e.debtor.%d@test.local", ts)

	var admin, payerTok, debtorTok string
	var payerID, debtorID, groupID, group2ID, group3ID, expenseID, paymentID, statusToken string

	t.Run("admin login returns tokens and user", func(t *testing.T) {
		code, m, _ := call(t, "POST", "/auth/login", "", map[string]string{"email": "admin@expense-splitter.local", "password": "admin"})
		want(t, code, 200, "admin login", m)
		admin = str(m, "access_token")
		actors[admin] = "global-admin"
		u, _ := m["user"].(map[string]any)
		if u == nil || u["is_global_admin"] != true {
			t.Fatalf("login must return the global-admin user, got %v", m["user"])
		}
	})

	t.Run("register + verify two members", func(t *testing.T) {
		payerID = register(t, admin, payerEmail, "E2E Payer")
		debtorID = register(t, admin, debtorEmail, "E2E Debtor")
		payerTok, _ = login(t, payerEmail, "password123")
		debtorTok, _ = login(t, debtorEmail, "password123")
	})

	t.Run("session: refresh works, logout revokes", func(t *testing.T) {
		tok, refresh := login(t, payerEmail, "password123")
		code, m, _ := call(t, "POST", "/auth/refresh", "", map[string]string{"refresh_token": refresh})
		want(t, code, 200, "refresh", m)
		newAccess, newRefresh := str(m, "access_token"), str(m, "refresh_token")
		actors[newAccess] = "payer (refreshed session)"
		code, m, _ = call(t, "GET", "/me", newAccess, nil)
		want(t, code, 200, "me with refreshed token", m)
		code, _, _ = call(t, "POST", "/auth/logout", "", map[string]string{"refresh_token": newRefresh})
		want(t, code, 204, "logout", nil)
		code, m, _ = call(t, "POST", "/auth/refresh", "", map[string]string{"refresh_token": newRefresh})
		want(t, code, 401, "refresh after logout", m)
		_ = tok
	})

	t.Run("verification flow + rejected user is blocked", func(t *testing.T) {
		rejEmail := fmt.Sprintf("e2e.rej.%d@test.local", ts)
		code, m, _ := call(t, "POST", "/auth/register", "", map[string]string{"email": rejEmail, "password": "password123", "name": "E2E Reject"})
		want(t, code, 201, "register reject-user", m)
		rejID := str(m, "id")
		rejTok, _ := login(t, rejEmail, "password123")
		code, m, _ = call(t, "POST", "/verification", rejTok, nil)
		want(t, code, 200, "submit verification", m)
		if str(m, "verification_status") != "pending_verification" {
			t.Fatalf("expected pending_verification, got %v", m)
		}
		code, m, _ = call(t, "POST", "/admin/users/"+rejID+"/reject", admin, nil)
		want(t, code, 200, "admin reject", m)
		code, m, _ = call(t, "POST", "/groups", rejTok, map[string]any{"name": "nope", "start_date": "2026-07-01T00:00:00Z", "end_date": "2026-07-02T00:00:00Z"})
		want(t, code, 403, "rejected user cannot create groups", m)
	})

	t.Run("groups: create, admin-assign variant, get, update, list", func(t *testing.T) {
		code, m, _ := call(t, "POST", "/groups", payerTok, map[string]any{"name": "E2E Trip", "start_date": "2026-07-01T00:00:00Z", "end_date": "2026-07-10T00:00:00Z"})
		want(t, code, 201, "create group", m)
		groupID = str(m, "id")
		invite := str(m, "invite_token")

		code, m, _ = call(t, "POST", "/groups", admin, map[string]any{"name": "E2E Assigned", "start_date": "2026-07-01T00:00:00Z", "end_date": "2026-07-05T00:00:00Z", "admin_user_id": payerID})
		want(t, code, 201, "admin-assign create", m)
		group2ID = str(m, "id")

		code, m, _ = call(t, "PATCH", "/groups/"+groupID, payerTok, map[string]any{"name": "E2E Trip v2", "start_date": "2026-07-01T00:00:00Z", "end_date": "2026-07-12T00:00:00Z"})
		want(t, code, 200, "update group", m)

		code, m, _ = call(t, "GET", "/groups/"+groupID, payerTok, nil)
		want(t, code, 200, "get group", m)
		statusToken = str(m, "status_token")
		if statusToken == "" {
			t.Fatal("group detail must expose status_token")
		}

		code, m, _ = call(t, "GET", "/groups", payerTok, nil)
		want(t, code, 200, "list my groups", m)

		code, m, _ = call(t, "POST", "/groups/join", debtorTok, map[string]string{"invite_token": invite})
		want(t, code, 201, "join", m)
		code, m, _ = call(t, "GET", "/groups/"+groupID+"/requests", payerTok, nil)
		want(t, code, 200, "list requests", m)
		code, m, _ = call(t, "POST", "/groups/"+groupID+"/members/"+debtorID+"/approve", payerTok, nil)
		want(t, code, 200, "approve member", m)
	})

	t.Run("expenses: record equal + weighted, update (audited), delete, paginated list", func(t *testing.T) {
		code, m, _ := call(t, "POST", "/groups/"+groupID+"/expenses", payerTok, map[string]any{"amount_baisa": 10000, "category": "food", "description": "e2e dinner", "occurred_on": "2026-07-02"})
		want(t, code, 201, "record equal expense", m)
		expenseID = str(m, "id")

		code, m, _ = call(t, "POST", "/groups/"+groupID+"/expenses", payerTok, map[string]any{
			"amount_baisa": 9000, "category": "lodging", "description": "e2e single room", "occurred_on": "2026-07-03",
			"shares": []map[string]any{{"user_id": payerID, "weight": 2}, {"user_id": debtorID, "weight": 1}}})
		want(t, code, 201, "record weighted expense", m)
		if str(m, "split_type") != "weighted" {
			t.Fatalf("expected weighted split, got %v", m["split_type"])
		}

		code, m, _ = call(t, "PATCH", "/groups/"+groupID+"/expenses/"+expenseID, payerTok, map[string]any{"amount_baisa": 12000, "category": "food", "description": "e2e dinner fixed", "occurred_on": "2026-07-02"})
		want(t, code, 200, "update expense", m)

		code, m, _ = call(t, "POST", "/groups/"+groupID+"/expenses", payerTok, map[string]any{"amount_baisa": 500, "category": "other", "description": "oops", "occurred_on": "2026-07-02"})
		want(t, code, 201, "record throwaway expense", m)
		code, _, _ = call(t, "DELETE", "/groups/"+groupID+"/expenses/"+str(m, "id"), payerTok, nil)
		want(t, code, 204, "delete expense", nil)

		code, m, _ = call(t, "GET", "/groups/"+groupID+"/expenses?limit=1&offset=0", payerTok, nil)
		want(t, code, 200, "paginated expenses", m)
		if got := len(items(t, m, "expenses page")); got != 1 {
			t.Fatalf("limit=1 must return 1 item, got %d", got)
		}
		if m["total"].(float64) != 2 {
			t.Fatalf("total must be 2 active expenses, got %v", m["total"])
		}
		code, m, _ = call(t, "GET", "/groups/"+groupID+"/expenses?category=lodging&q=room", payerTok, nil)
		want(t, code, 200, "filtered expenses", m)
		if got := len(items(t, m, "filtered page")); got != 1 {
			t.Fatalf("filter should match exactly the weighted expense, got %d", got)
		}
	})

	t.Run("summary reflects weighted math", func(t *testing.T) {
		code, m, _ := call(t, "GET", "/groups/"+groupID+"/summary", debtorTok, nil)
		want(t, code, 200, "summary", m)
		if m["total_spent"].(float64) != 21000 {
			t.Fatalf("total_spent: want 21000, got %v", m["total_spent"])
		}
		for _, mm := range m["members"].([]any) {
			mem := mm.(map[string]any)
			switch str(mem, "user_id") {
			case payerID:
				if mem["net"].(float64) != 9000 {
					t.Fatalf("payer net: want 9000, got %v", mem["net"])
				}
			case debtorID:
				if mem["net"].(float64) != -9000 {
					t.Fatalf("debtor net: want -9000, got %v", mem["net"])
				}
			}
		}
	})

	t.Run("public status by share token", func(t *testing.T) {
		code, m, _ := call(t, "GET", "/public/groups/"+statusToken, "", nil)
		want(t, code, 200, "public status", m)
		if m["member_count"].(float64) != 2 {
			t.Fatalf("public member_count: want 2, got %v", m["member_count"])
		}
		code, m, _ = call(t, "GET", "/public/groups/00000000-0000-0000-0000-000000000000", "", nil)
		want(t, code, 404, "unknown share token", m)
		code, m, _ = call(t, "GET", "/public/groups/junk", "", nil)
		want(t, code, 400, "junk share token", m)
	})

	t.Run("close + settlement plan with snapshot", func(t *testing.T) {
		code, m, _ := call(t, "GET", "/groups/"+group2ID+"/settlement", payerTok, nil)
		want(t, code, 409, "plan before close", m)

		code, m, _ = call(t, "POST", "/groups/"+groupID+"/close", payerTok, nil)
		want(t, code, 200, "close", m)

		code, m, _ = call(t, "GET", "/groups/"+groupID+"/settlement", debtorTok, nil)
		want(t, code, 200, "plan", m)
		pays := m["payments"].([]any)
		if len(pays) != 1 {
			t.Fatalf("plan: want exactly 1 payment, got %d", len(pays))
		}
		p := pays[0].(map[string]any)
		if p["amount_baisa"].(float64) != 9000 || str(p, "from") != debtorID || str(p, "to") != payerID {
			t.Fatalf("plan payment wrong: %v", p)
		}
		paymentID = str(p, "id")
		if m["snapshot"] == nil {
			t.Fatal("plan must include the settlement snapshot")
		}
	})

	t.Run("two-key payment flow with tamper denials", func(t *testing.T) {
		code, m, _ := call(t, "POST", "/payments/"+paymentID+"/finalize", debtorTok, nil)
		want(t, code, 403, "debtor self-finalize denied", m)
		code, m, _ = call(t, "POST", "/payments/"+paymentID+"/confirm", debtorTok, nil)
		want(t, code, 403, "debtor self-confirm denied", m)
		code, m, _ = call(t, "POST", "/payments/"+paymentID+"/finalize", payerTok, nil)
		want(t, code, 409, "finalize before creditor key denied", m)

		code, m, _ = call(t, "POST", "/payments/"+paymentID+"/proof", debtorTok, map[string]string{"note": "paid 9000 cash"})
		want(t, code, 200, "text proof", m)
		code, m, _ = call(t, "GET", "/payments/"+paymentID+"/proof", payerTok, nil)
		want(t, code, 200, "proof metadata", m)
		if str(m, "proof_type") != "text" || str(m, "note") == "" {
			t.Fatalf("text proof metadata wrong: %v", m)
		}
		code, m, _ = call(t, "GET", "/payments/"+paymentID+"/proof/image", payerTok, nil)
		want(t, code, 409, "image bytes for a text note", m)

		code, m, _ = call(t, "POST", "/payments/"+paymentID+"/confirm", payerTok, nil)
		want(t, code, 200, "creditor confirm", m)
		code, m, _ = call(t, "POST", "/payments/"+paymentID+"/finalize", payerTok, nil)
		want(t, code, 200, "finalize", m)
		if str(m, "status") != "settled" {
			t.Fatalf("expected settled, got %v", m["status"])
		}
		code, m, _ = call(t, "POST", "/payments/"+paymentID+"/finalize", admin, nil)
		want(t, code, 409, "settled is terminal", m)

		code, m, _ = call(t, "GET", "/groups/"+groupID+"/settlement", payerTok, nil)
		want(t, code, 200, "plan after settle", m)
		if m["settled_count"].(float64) != 1 || str(m, "group_status") != "settled" {
			t.Fatalf("group should be fully settled: %v", m)
		}
	})

	t.Run("settlement report PDF", func(t *testing.T) {
		req, _ := http.NewRequest("GET", base+"/groups/"+groupID+"/report.pdf", nil)
		req.Header.Set("Authorization", "Bearer "+payerTok)
		resp, err := httpc.Do(req)
		if err != nil {
			t.Fatalf("report: %v", err)
		}
		defer resp.Body.Close()
		raw, _ := io.ReadAll(resp.Body)
		t.Logf("GET /groups/%s/report.pdf  [as %s]", groupID, actor(payerTok))
		t.Logf("   -> %d  %s, %d bytes, starts with %q", resp.StatusCode, resp.Header.Get("Content-Type"), len(raw), raw[:min(8, len(raw))])
		if resp.StatusCode != 200 || !bytes.HasPrefix(raw, []byte("%PDF")) {
			t.Fatalf("   FAIL report: expected 200 + %%PDF prefix")
		}
		t.Log("   OK   settlement report is a real PDF")
	})

	t.Run("group3: image proof, dispute loop, nudges, member mgmt", func(t *testing.T) {
		code, m, _ := call(t, "POST", "/groups", payerTok, map[string]any{"name": "E2E Proofs", "start_date": "2026-07-01T00:00:00Z", "end_date": "2026-07-10T00:00:00Z"})
		want(t, code, 201, "create group3", m)
		group3ID = str(m, "id")
		invite := str(m, "invite_token")

		code, m, _ = call(t, "POST", "/groups/join", debtorTok, map[string]string{"invite_token": invite})
		want(t, code, 201, "debtor join g3", m)
		code, m, _ = call(t, "POST", "/groups/"+group3ID+"/members/"+debtorID+"/approve", payerTok, nil)
		want(t, code, 200, "approve debtor g3", m)

		remEmail := fmt.Sprintf("e2e.rem.%d@test.local", ts)
		remID := register(t, admin, remEmail, "E2E Removable")
		remTok, _ := login(t, remEmail, "password123")
		code, m, _ = call(t, "POST", "/groups/join", remTok, map[string]string{"invite_token": invite})
		want(t, code, 201, "removable joins", m)
		code, m, _ = call(t, "POST", "/groups/"+group3ID+"/members/"+remID+"/reject", payerTok, nil)
		want(t, code, 200, "reject join request", m)

		code, m, _ = call(t, "POST", "/groups/"+group3ID+"/expenses", payerTok, map[string]any{"amount_baisa": 4000, "category": "fuel", "description": "e2e tank", "occurred_on": "2026-07-04"})
		want(t, code, 201, "g3 expense", m)
		code, m, _ = call(t, "POST", "/groups/"+group3ID+"/close", payerTok, nil)
		want(t, code, 200, "g3 close", m)
		code, m, _ = call(t, "GET", "/groups/"+group3ID+"/settlement", payerTok, nil)
		want(t, code, 200, "g3 plan", m)
		pay3 := str(items(t, map[string]any{"items": m["payments"]}, "g3 payments")[0].(map[string]any), "id")

		code, m, _ = call(t, "POST", "/groups/"+group3ID+"/nudges?hours=0", payerTok, nil)
		want(t, code, 200, "nudges run 1", m)
		if len(m["sent"].([]any)) != 1 {
			t.Fatalf("first nudge run should send 1, got %v", m)
		}
		code, m, _ = call(t, "POST", "/groups/"+group3ID+"/nudges?hours=1", payerTok, nil)
		want(t, code, 200, "nudges run 2", m)
		if len(m["sent"].([]any)) != 0 {
			t.Fatalf("second nudge run inside threshold should send 0, got %v", m)
		}

		code, m = uploadProof(t, debtorTok, pay3, []byte("this is definitely not an image"))
		want(t, code, 400, "fake image rejected by magic bytes", m)
		pngBytes := tinyPNG(t)
		code, m = uploadProof(t, debtorTok, pay3, pngBytes)
		want(t, code, 200, "image proof accepted", m)

		code, m, _ = call(t, "GET", "/payments/"+pay3+"/proof", payerTok, nil)
		want(t, code, 200, "image proof metadata", m)
		if str(m, "proof_type") != "image" || m["byte_size"].(float64) != float64(len(pngBytes)) {
			t.Fatalf("image metadata wrong: %v", m)
		}
		req, _ := http.NewRequest("GET", base+"/payments/"+pay3+"/proof/image", nil)
		req.Header.Set("Authorization", "Bearer "+payerTok)
		resp, err := httpc.Do(req)
		if err != nil {
			t.Fatalf("get image: %v", err)
		}
		raw, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		t.Logf("GET /payments/%s/proof/image  [as %s]", pay3, actor(payerTok))
		t.Logf("   -> %d  %s, %d bytes (uploaded %d)", resp.StatusCode, resp.Header.Get("Content-Type"), len(raw), len(pngBytes))
		if resp.StatusCode != 200 || !bytes.Equal(raw, pngBytes) {
			t.Fatalf("   FAIL image round-trip: bytes differ")
		}
		t.Log("   OK   image round-trip: downloaded bytes identical to the upload")

		code, m, _ = call(t, "POST", "/payments/"+pay3+"/dispute", payerTok, nil)
		want(t, code, 200, "creditor dispute", m)
		code, m, _ = call(t, "POST", "/payments/"+pay3+"/proof", debtorTok, map[string]string{"note": "re-sent via transfer"})
		want(t, code, 200, "re-submit after dispute", m)
		code, m, _ = call(t, "POST", "/payments/"+pay3+"/reject", admin, nil)
		want(t, code, 200, "admin reject", m)
		code, m, _ = call(t, "POST", "/payments/"+pay3+"/proof", debtorTok, map[string]string{"note": "third time is the charm"})
		want(t, code, 200, "re-submit again", m)
		code, m, _ = call(t, "POST", "/payments/"+pay3+"/confirm", payerTok, nil)
		want(t, code, 200, "confirm g3", m)
		code, m, _ = call(t, "POST", "/payments/"+pay3+"/finalize", admin, nil)
		want(t, code, 200, "global admin finalize", m)

		code, m, _ = call(t, "POST", "/groups/"+group3ID+"/members/"+debtorID+"/promote", payerTok, nil)
		want(t, code, 200, "promote handoff", m)
		if str(m, "role") != "group_admin" {
			t.Fatalf("promote should return group_admin role, got %v", m)
		}
	})

	t.Run("audit trail is complete and paginated", func(t *testing.T) {
		code, m, _ := call(t, "GET", "/groups/"+groupID+"/audit?limit=200", payerTok, nil)
		want(t, code, 200, "audit", m)
		seen := map[string]bool{}
		for _, e := range items(t, m, "audit page") {
			seen[str(e.(map[string]any), "action")] = true
		}
		for _, action := range []string{"expense.created", "expense.amount_changed", "expense.deleted", "membership.approved", "group.closed", "payment.proof_submitted", "payment.creditor_confirmed", "payment.settled", "group.fully_settled"} {
			if !seen[action] {
				t.Errorf("audit trail missing action %s (have %v)", action, seen)
			}
		}
		code, m, _ = call(t, "GET", "/groups/"+groupID+"/audit?limit=2&offset=0", payerTok, nil)
		want(t, code, 200, "audit page", m)
		if len(items(t, m, "audit page 2")) != 2 {
			t.Fatalf("audit limit=2 must return 2 items")
		}
	})

	t.Run("admin CRUD with pagination and guards", func(t *testing.T) {
		code, m, _ := call(t, "GET", "/admin/users?limit=1&offset=0", admin, nil)
		want(t, code, 200, "admin users page", m)
		if len(items(t, m, "users page")) != 1 || m["total"].(float64) < 3 {
			t.Fatalf("users pagination wrong: %v", m)
		}
		code, m, _ = call(t, "GET", "/admin/users?status=bogus", admin, nil)
		want(t, code, 400, "bogus status filter", m)
		code, m, _ = call(t, "GET", "/admin/users", payerTok, nil)
		want(t, code, 403, "non-admin denied", m)

		code, m, _ = call(t, "GET", "/admin/users/"+debtorID, admin, nil)
		want(t, code, 200, "admin user detail", m)
		if len(m["memberships"].([]any)) < 2 {
			t.Fatalf("debtor should hold >=2 memberships, got %v", m["memberships"])
		}

		code, m, _ = call(t, "GET", "/admin/groups?limit=200", admin, nil)
		want(t, code, 200, "admin groups", m)

		code, m, _ = call(t, "DELETE", "/admin/users/"+debtorID, admin, nil)
		want(t, code, 409, "cannot delete a member user", m)
		tmpEmail := fmt.Sprintf("e2e.tmp.%d@test.local", ts)
		tmpID := register(t, admin, tmpEmail, "E2E Temp")
		code, _, _ = call(t, "DELETE", "/admin/users/"+tmpID, admin, nil)
		want(t, code, 204, "delete pristine user", nil)

		code, m, _ = call(t, "DELETE", "/admin/groups/"+groupID, admin, nil)
		want(t, code, 409, "cannot delete a group with history", m)
		code, _, _ = call(t, "DELETE", "/admin/groups/"+group2ID, admin, nil)
		want(t, code, 204, "delete pristine group", nil)
	})

	t.Run("register link endpoint stays idempotent", func(t *testing.T) {
		code, m, _ := call(t, "POST", "/register", payerTok, nil)
		want(t, code, 201, "register link", m)
		if str(m, "id") != payerID {
			t.Fatalf("link must return the same user, got %v", m)
		}
	})

	if !strings.Contains(payerEmail, "e2e.payer") {
		t.Fatal("unreachable")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
