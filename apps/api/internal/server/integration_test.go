package server_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/fisca-app/fisca/apps/api/internal/auth"
	"github.com/fisca-app/fisca/apps/api/internal/auth/testjwks"
	"github.com/fisca-app/fisca/apps/api/internal/config"
	"github.com/fisca-app/fisca/apps/api/internal/server"
)

// Integration tests for the core flows in PRD §23 (exit criteria 1-20) and the
// cross-user isolation rule (exit criterion 19). They run the real router
// against a migrated database and create their own random users, which are
// deleted on cleanup (every user-owned table cascades from users).
//
//	TEST_DATABASE_URL=postgres://fisca:fisca@localhost:5434/fisca?sslmode=disable go test ./internal/server/...

type testEnv struct {
	t      *testing.T
	router http.Handler
	pool   *pgxpool.Pool
}

// newEnv builds the real router with a stub verifier. The production build
// tidak lagi punya token yang dipercaya apa adanya (ISSUE-15 menghapus
// AUTH_MODE=dev), jadi test bisnis menyuntik verifier miliknya sendiri —
// jalur auth sungguhan diuji terpisah di auth_flow_test.go.
func newEnv(t *testing.T) *testEnv {
	t.Helper()
	return newEnvWithVerifier(t, config.AuthModeSession, func(*pgxpool.Pool) auth.Verifier {
		return stubVerifier{}
	})
}

// newEnvWithSessionAuth membangun router dengan verifier sesi ASLI, supaya
// alur register → login → request terautentikasi → logout bisa diuji ujung ke
// ujung (AC ISSUE-15). Test bisnis memakai stubVerifier; jalur inilah yang
// membuktikan verifier produksi bekerja.
func newEnvWithSessionAuth(t *testing.T) *testEnv {
	t.Helper()
	return newEnvWithVerifier(t, config.AuthModeSession, auth.NewSessionVerifier)
}

const stubTokenPrefix = "stub:"

// stubVerifier menerima "stub:<uuid>". Didefinisikan di file _test sehingga
// tidak mungkin ikut terkompilasi ke binary API.
type stubVerifier struct{}

func (stubVerifier) Mode() string { return "stub" }

func (stubVerifier) Verify(_ context.Context, raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if !strings.HasPrefix(raw, stubTokenPrefix) {
		return "", auth.ErrInvalidToken
	}
	id := strings.TrimSpace(strings.TrimPrefix(raw, stubTokenPrefix))
	if _, err := uuid.Parse(id); err != nil {
		return "", auth.ErrInvalidToken
	}
	return id, nil
}

// newEnvWithVerifier builds the real router in a chosen auth mode. The verifier
// is built from the pool so pool-backed verifiers (session) can be injected;
// it runs after the TEST_DATABASE_URL skip, so a missing database never reaches
// the factory.
func newEnvWithVerifier(t *testing.T, mode string, build func(*pgxpool.Pool) auth.Verifier) *testEnv {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set TEST_DATABASE_URL to run integration tests")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(pool.Close)

	gin.SetMode(gin.TestMode)
	cfg := &config.Config{
		Env:                    config.EnvDevelopment,
		AuthMode:               mode,
		CORSOrigins:            []string{},
		RateLimitRPS:           10000,
		RateLimitBurst:         10000,
		BudgetWarningThreshold: 0.8,
		OpenAIBaseURL:          "https://api.openai.com/v1",
		AITimeout:              5 * time.Second,
	}
	router := server.New(server.Options{
		Config:   cfg,
		DB:       pool,
		Verifier: build(pool),
		Version:  "test",
	})
	return &testEnv{t: t, router: router, pool: pool}
}

// newUser returns a fresh user token and registers cleanup for its rows.
// newUser returns a token for a fresh user, with cleanup scheduled.
func (e *testEnv) newUser() string {
	return stubTokenPrefix + e.newUserID().String()
}

// newUserID registers a fresh user id without deciding which token format to
// use for it, and always schedules cleanup.
func (e *testEnv) newUserID() uuid.UUID {
	id := uuid.New()
	t := e.t
	t.Cleanup(func() {
		_, _ = e.pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, id)
	})
	return id
}

type response struct {
	Status int
	Body   map[string]any
	Raw    string
}

func (e *testEnv) call(token, method, path string, body any, idemKey string) response {
	e.t.Helper()
	var reader *bytes.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			e.t.Fatalf("marshal: %v", err)
		}
		reader = bytes.NewReader(raw)
	} else {
		reader = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	if idemKey != "" {
		req.Header.Set("Idempotency-Key", idemKey)
	}
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)

	out := response{Status: rec.Code, Raw: rec.Body.String()}
	_ = json.Unmarshal(rec.Body.Bytes(), &out.Body)
	return out
}

func dataOf(r response) map[string]any {
	d, _ := r.Body["data"].(map[string]any)
	return d
}

func listOf(r response) []any {
	d, _ := r.Body["data"].([]any)
	return d
}

// defaultAccount returns the auto-provisioned Cash account id for a user.
func (e *testEnv) defaultAccount(token string) string {
	e.t.Helper()
	r := e.call(token, http.MethodGet, "/api/v1/accounts", nil, "")
	if r.Status != http.StatusOK {
		e.t.Fatalf("GET /accounts = %d: %s", r.Status, r.Raw)
	}
	for _, a := range listOf(r) {
		acc := a.(map[string]any)
		if acc["isDefault"] == true {
			return acc["id"].(string)
		}
	}
	e.t.Fatal("no default account provisioned")
	return ""
}

func (e *testEnv) createExpense(token, accountID, desc string, amount int64, key string) response {
	return e.call(token, http.MethodPost, "/api/v1/transactions", map[string]any{
		"type":        "expense",
		"amount":      amount,
		"description": desc,
		"accountId":   accountID,
		"source":      "manual",
	}, key)
}

// ---------------------------------------------------------------------------
// Exit 19: no cross-user access on any read or write path.
// ---------------------------------------------------------------------------

func TestCrossUserAccessIsRejected(t *testing.T) {
	e := newEnv(t)
	alice := e.newUser()
	bob := e.newUser()

	aliceAcc := e.defaultAccount(alice)
	created := e.createExpense(alice, aliceAcc, "Rahasia Alice", 50000, uuid.NewString())
	if created.Status != http.StatusCreated {
		t.Fatalf("alice create = %d: %s", created.Status, created.Raw)
	}
	txID := dataOf(created)["transaction"].(map[string]any)["id"].(string)

	budget := e.call(alice, http.MethodPost, "/api/v1/budgets", map[string]any{
		"period": "this_month", "overallLimit": 1000000,
	}, "")
	if budget.Status != http.StatusCreated {
		t.Fatalf("alice budget = %d: %s", budget.Status, budget.Raw)
	}
	budgetID := dataOf(budget)["id"].(string)

	proposal := e.call(alice, http.MethodPost, "/api/v1/budget-proposals", map[string]any{
		"budgetId": budgetID,
		"changes":  map[string]any{"overallLimit": 2000000},
	}, "")
	if proposal.Status != http.StatusCreated {
		t.Fatalf("alice proposal = %d: %s", proposal.Status, proposal.Raw)
	}
	proposalID := dataOf(proposal)["id"].(string)
	baseVersion := dataOf(proposal)["baseVersion"]

	// Bob must not see, edit, void, or approve any of Alice's rows.
	checks := []struct {
		name   string
		method string
		path   string
		body   any
		key    bool
	}{
		{"get transaction", http.MethodGet, "/api/v1/transactions/" + txID, nil, false},
		{"patch transaction", http.MethodPatch, "/api/v1/transactions/" + txID, map[string]any{"version": 1, "amount": 1}, false},
		{"void transaction", http.MethodPost, "/api/v1/transactions/" + txID + "/void", map[string]any{"reason": "x"}, true},
		{"get budget", http.MethodGet, "/api/v1/budgets/" + budgetID, nil, false},
		{"patch budget", http.MethodPatch, "/api/v1/budgets/" + budgetID, map[string]any{"version": 1, "overallLimit": 1}, false},
		{"get budget status", http.MethodGet, "/api/v1/budgets/" + budgetID + "/status", nil, false},
		{"get proposal", http.MethodGet, "/api/v1/budget-proposals/" + proposalID, nil, false},
		{"approve proposal", http.MethodPost, "/api/v1/budget-proposals/" + proposalID + "/approve", map[string]any{"baseVersion": baseVersion}, true},
		{"reject proposal", http.MethodPost, "/api/v1/budget-proposals/" + proposalID + "/reject", nil, false},
	}
	for _, c := range checks {
		key := ""
		if c.key {
			key = uuid.NewString()
		}
		r := e.call(bob, c.method, c.path, c.body, key)
		if r.Status != http.StatusNotFound {
			t.Errorf("%s: status = %d, want 404 (body: %s)", c.name, r.Status, r.Raw)
		}
	}

	// Bob's listings must not contain Alice's data.
	list := e.call(bob, http.MethodGet, "/api/v1/transactions", nil, "")
	if list.Status != http.StatusOK {
		t.Fatalf("bob list = %d", list.Status)
	}
	if items, _ := dataOf(list)["items"].([]any); len(items) != 0 {
		t.Errorf("bob sees %d of alice's transactions", len(items))
	}
	search := e.call(bob, http.MethodGet, "/api/v1/transactions/search?q=Rahasia", nil, "")
	if items, _ := dataOf(search)["items"].([]any); len(items) != 0 {
		t.Errorf("bob's search found alice's transaction")
	}

	// Alice's data must be untouched by Bob's attempts.
	still := e.call(alice, http.MethodGet, "/api/v1/transactions/"+txID, nil, "")
	if still.Status != http.StatusOK {
		t.Fatalf("alice lost access to own tx: %d", still.Status)
	}
	if status := dataOf(still)["transaction"].(map[string]any)["status"]; status != "active" {
		t.Errorf("alice tx status = %v after bob's void attempt, want active", status)
	}
}

// ---------------------------------------------------------------------------
// Exit 8: a retry must not create a second transaction.
// ---------------------------------------------------------------------------

func TestIdempotentRetryCreatesOneTransaction(t *testing.T) {
	e := newEnv(t)
	user := e.newUser()
	acc := e.defaultAccount(user)
	key := uuid.NewString()

	first := e.createExpense(user, acc, "Retry kopi", 18000, key)
	second := e.createExpense(user, acc, "Retry kopi", 18000, key)
	if first.Status != http.StatusCreated || second.Status != http.StatusCreated {
		t.Fatalf("statuses = %d, %d; want 201, 201", first.Status, second.Status)
	}
	id1 := dataOf(first)["transaction"].(map[string]any)["id"]
	id2 := dataOf(second)["transaction"].(map[string]any)["id"]
	if id1 != id2 {
		t.Fatalf("replay returned a different transaction: %v vs %v", id1, id2)
	}

	list := e.call(user, http.MethodGet, "/api/v1/transactions?limit=50", nil, "")
	items, _ := dataOf(list)["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("transactions after retry = %d, want 1", len(items))
	}
}

func TestIdempotencyKeyReuseWithDifferentBodyConflicts(t *testing.T) {
	e := newEnv(t)
	user := e.newUser()
	acc := e.defaultAccount(user)
	key := uuid.NewString()

	if r := e.createExpense(user, acc, "Asli", 10000, key); r.Status != http.StatusCreated {
		t.Fatalf("first = %d", r.Status)
	}
	r := e.createExpense(user, acc, "Beda", 99000, key)
	if r.Status != http.StatusConflict {
		t.Fatalf("status = %d, want 409", r.Status)
	}
	if code := r.Body["error"].(map[string]any)["code"]; code != "IDEMPOTENCY_MISMATCH" {
		t.Fatalf("code = %v, want IDEMPOTENCY_MISMATCH", code)
	}
}

// ---------------------------------------------------------------------------
// Exit 7 & 16: over-budget warns but never blocks the transaction.
// ---------------------------------------------------------------------------

func TestOverBudgetWarnsWithoutBlocking(t *testing.T) {
	e := newEnv(t)
	user := e.newUser()
	acc := e.defaultAccount(user)

	if r := e.call(user, http.MethodPost, "/api/v1/budgets", map[string]any{
		"period": "this_month", "overallLimit": 100000,
	}, ""); r.Status != http.StatusCreated {
		t.Fatalf("budget = %d: %s", r.Status, r.Raw)
	}

	r := e.createExpense(user, acc, "Belanja besar", 150000, uuid.NewString())
	if r.Status != http.StatusCreated {
		t.Fatalf("over-budget transaction was rejected: %d %s", r.Status, r.Raw)
	}
	warnings, _ := dataOf(r)["warnings"].([]any)
	if len(warnings) == 0 {
		t.Fatal("expected a budget warning")
	}
	// BUDGET_WARNING (near limit) and BUDGET_EXCEEDED (past limit) are both
	// warnings; either is acceptable, blocking is not.
	warnType := warnings[0].(map[string]any)["type"]
	if warnType != "BUDGET_WARNING" && warnType != "BUDGET_EXCEEDED" {
		t.Fatalf("warning type = %v, want BUDGET_WARNING or BUDGET_EXCEEDED", warnType)
	}
}

// ---------------------------------------------------------------------------
// Exit 14 & 15: budget changes need approval and a current baseVersion.
// ---------------------------------------------------------------------------

func TestBudgetChangeRequiresApproval(t *testing.T) {
	e := newEnv(t)
	user := e.newUser()

	budget := e.call(user, http.MethodPost, "/api/v1/budgets", map[string]any{
		"period": "this_month", "overallLimit": 1000000,
	}, "")
	budgetID := dataOf(budget)["id"].(string)

	proposal := e.call(user, http.MethodPost, "/api/v1/budget-proposals", map[string]any{
		"budgetId": budgetID,
		"changes":  map[string]any{"overallLimit": 3000000},
	}, "")
	if proposal.Status != http.StatusCreated {
		t.Fatalf("proposal = %d: %s", proposal.Status, proposal.Raw)
	}
	proposalID := dataOf(proposal)["id"].(string)
	baseVersion := dataOf(proposal)["baseVersion"]

	// Still pending: the live budget must not have changed.
	current := e.call(user, http.MethodGet, "/api/v1/budgets/"+budgetID, nil, "")
	if limit := dataOf(current)["overallLimit"]; limit != float64(1000000) {
		t.Fatalf("limit changed before approval: %v", limit)
	}

	// A stale baseVersion must be refused.
	stale := e.call(user, http.MethodPost, "/api/v1/budget-proposals/"+proposalID+"/approve",
		map[string]any{"baseVersion": 99}, uuid.NewString())
	if stale.Status != http.StatusConflict {
		t.Fatalf("stale approve = %d, want 409 (body: %s)", stale.Status, stale.Raw)
	}
	if code := stale.Body["error"].(map[string]any)["code"]; code != "BUDGET_STALE" {
		t.Fatalf("stale code = %v, want BUDGET_STALE", code)
	}

	// Correct approval applies the change and writes an audit row.
	ok := e.call(user, http.MethodPost, "/api/v1/budget-proposals/"+proposalID+"/approve",
		map[string]any{"baseVersion": baseVersion}, uuid.NewString())
	if ok.Status != http.StatusOK {
		t.Fatalf("approve = %d: %s", ok.Status, ok.Raw)
	}
	if limit := dataOf(ok)["overallLimit"]; limit != float64(3000000) {
		t.Fatalf("limit after approval = %v, want 3000000", limit)
	}

	var audits int
	if err := e.pool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM audit_logs WHERE user_id = (
			SELECT user_id FROM budgets WHERE id = $1) AND entity_type = 'budget'`,
		budgetID).Scan(&audits); err != nil {
		t.Fatalf("audit query: %v", err)
	}
	if audits == 0 {
		t.Fatal("budget change left no audit row")
	}
}

// ---------------------------------------------------------------------------
// Exit 5-7: hybrid recording. An ambiguous message becomes a draft that does
// not touch balances; the clarification reply completes it.
// ---------------------------------------------------------------------------

func (e *testEnv) balance(token, accountID string) float64 {
	e.t.Helper()
	for _, a := range listOf(e.call(token, http.MethodGet, "/api/v1/accounts", nil, "")) {
		acc := a.(map[string]any)
		if acc["id"] == accountID {
			return acc["balance"].(float64)
		}
	}
	e.t.Fatalf("account %s not found", accountID)
	return 0
}

func TestAmbiguousInputDraftDoesNotChangeBalance(t *testing.T) {
	e := newEnv(t)
	user := e.newUser()
	acc := e.defaultAccount(user)

	before := e.balance(user, acc)

	first := e.call(user, http.MethodPost, "/api/v1/assistant/messages",
		map[string]any{"text": "catat 25rb", "source": "text"}, "")
	if first.Status != http.StatusOK {
		t.Fatalf("assistant = %d: %s", first.Status, first.Raw)
	}
	card := dataOf(first)["card"].(map[string]any)
	if card["type"] != "clarification" {
		t.Fatalf("card = %v, want clarification", card["type"])
	}
	if after := e.balance(user, acc); after != before {
		t.Fatalf("draft changed balance: %v -> %v", before, after)
	}

	convID := dataOf(first)["conversationId"].(string)
	reply := e.call(user, http.MethodPost, "/api/v1/assistant/messages",
		map[string]any{"conversationId": convID, "text": "kopi susu", "source": "text"}, "")
	if reply.Status != http.StatusOK {
		t.Fatalf("reply = %d: %s", reply.Status, reply.Raw)
	}
	if got := dataOf(reply)["card"].(map[string]any)["type"]; got != "transaction_result" {
		t.Fatalf("reply card = %v, want transaction_result", got)
	}
	if after := e.balance(user, acc); after != before-25000 {
		t.Fatalf("balance after completion = %v, want %v", after, before-25000)
	}
}

// Exit 10 (historical answer matches DB) and 11 (largest = DB query).
func TestLargestTransactionMatchesDatabase(t *testing.T) {
	e := newEnv(t)
	user := e.newUser()
	acc := e.defaultAccount(user)

	amounts := []int64{12000, 480000, 75000}
	for _, a := range amounts {
		if r := e.createExpense(user, acc, "Item", a, uuid.NewString()); r.Status != http.StatusCreated {
			t.Fatalf("create %d = %d", a, r.Status)
		}
	}

	r := e.call(user, http.MethodGet, "/api/v1/analytics/largest-transactions?period=this_month&type=expense&limit=1", nil, "")
	items, _ := dataOf(r)["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("largest returned %d items", len(items))
	}
	if got := items[0].(map[string]any)["amount"]; got != float64(480000) {
		t.Fatalf("largest amount = %v, want 480000", got)
	}
}

// Exit 12: unusual spending states when data is insufficient.
func TestUnusualSpendingStatesLimitedData(t *testing.T) {
	e := newEnv(t)
	user := e.newUser()
	r := e.call(user, http.MethodGet, "/api/v1/analytics/unusual-spending?period=this_month", nil, "")
	if r.Status != http.StatusOK {
		t.Fatalf("status = %d", r.Status)
	}
	if dataOf(r)["sufficientData"] != false {
		t.Fatal("new user must be reported as sufficientData=false")
	}
	if lim, _ := dataOf(r)["limitations"].([]any); len(lim) == 0 {
		t.Fatal("limitations must explain why data is insufficient")
	}
}

// ---------------------------------------------------------------------------
// Exit 1: a verified JWT identity reaches the API, and nothing else does.
// Runs the router in AUTH_MODE=jwt against a locally served JWKS, so the path
// is proven without a live Supabase project.
// ---------------------------------------------------------------------------

const (
	jwtIssuer = "https://local.supabase.test/auth/v1"
	jwtAud    = "authenticated"
)

func TestJWTModeAcceptsSignedTokenAndRejectsOthers(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	jwks := testjwks.Serve(t, &priv.PublicKey)

	verifier, err := auth.NewJWTVerifier(context.Background(), jwks.URL, jwtIssuer, jwtAud)
	if err != nil {
		t.Fatalf("new jwt verifier: %v", err)
	}
	e := newEnvWithVerifier(t, config.AuthModeJWT, func(*pgxpool.Pool) auth.Verifier { return verifier })
	if verifier.Mode() != "jwt" {
		t.Fatalf("mode = %q, want jwt", verifier.Mode())
	}

	userID := e.newUserID()
	otherID := e.newUserID()

	valid := testjwks.Sign(t, testjwks.Options{
		Signer: priv, Method: jwt.SigningMethodRS256, Issuer: jwtIssuer,
		Audience: jwtAud, Subject: userID.String(), ExpiresAt: time.Now().Add(time.Hour),
	})

	good := e.call(valid, http.MethodGet, "/api/v1/me", nil, "")
	if good.Status != http.StatusOK {
		t.Fatalf("valid jwt = %d: %s", good.Status, good.Raw)
	}
	if got := dataOf(good)["id"]; got != userID.String() {
		t.Fatalf("identity = %v, want %v", got, userID.String())
	}
	// The identity from the token decides whose data is returned.
	account := e.call(valid, http.MethodGet, "/api/v1/accounts", nil, "")
	if account.Status != http.StatusOK {
		t.Fatalf("accounts = %d", account.Status)
	}

	// A dev token must not work on a jwt-mode instance.
	if r := e.call(stubTokenPrefix+userID.String(), http.MethodGet, "/api/v1/me", nil, ""); r.Status != http.StatusUnauthorized {
		t.Fatalf("dev token in jwt mode = %d, want 401", r.Status)
	}

	// A valid token for another user must not be reused as this user's.
	foreign := testjwks.Sign(t, testjwks.Options{
		Signer: priv, Method: jwt.SigningMethodRS256, Issuer: jwtIssuer,
		Audience: jwtAud, Subject: otherID.String(), ExpiresAt: time.Now().Add(time.Hour),
	})
	foreignMe := e.call(foreign, http.MethodGet, "/api/v1/me", nil, "")
	if got := dataOf(foreignMe)["id"]; got != otherID.String() {
		t.Fatalf("foreign identity = %v, want %v", got, otherID.String())
	}

	// Tampered and untrusted tokens are rejected before touching the database.
	for name, token := range map[string]string{
		"unknown key": testjwks.Sign(t, testjwks.Options{
			Signer: mustOtherKey(t), Method: jwt.SigningMethodRS256,
			Issuer: jwtIssuer, Audience: jwtAud, Subject: userID.String(),
			ExpiresAt: time.Now().Add(time.Hour),
		}),
		"wrong issuer": testjwks.Sign(t, testjwks.Options{
			Signer: priv, Method: jwt.SigningMethodRS256, Issuer: "https://evil.test",
			Audience: jwtAud, Subject: userID.String(), ExpiresAt: time.Now().Add(time.Hour),
		}),
		"expired": testjwks.Sign(t, testjwks.Options{
			Signer: priv, Method: jwt.SigningMethodRS256, Issuer: jwtIssuer,
			Audience: jwtAud, Subject: userID.String(), ExpiresAt: time.Now().Add(-time.Hour),
		}),
		"garbage": "aaaa.bbbb.cccc",
	} {
		if r := e.call(token, http.MethodGet, "/api/v1/me", nil, ""); r.Status != http.StatusUnauthorized {
			t.Errorf("%s: status = %d, want 401 (body: %s)", name, r.Status, r.Raw)
		}
	}
}

// mustOtherKey returns a second RSA key, i.e. one the JWKS does not publish.
func mustOtherKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	other, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate other key: %v", err)
	}
	return other
}

// PRD §22.3 / plan.md §12.4: one sentence with several items becomes several
// transactions, recorded together or not at all.
func TestMultiTransactionSentenceCreatesTwoRows(t *testing.T) {
	e := newEnv(t)
	user := e.newUser()
	acc := e.defaultAccount(user)
	before := e.balance(user, acc)

	r := e.call(user, http.MethodPost, "/api/v1/assistant/messages", map[string]any{
		"text": "beli makan 25 ribu dan kopi 18 ribu", "source": "text",
	}, "")
	if r.Status != http.StatusOK {
		t.Fatalf("assistant = %d: %s", r.Status, r.Raw)
	}
	card := dataOf(r)["card"].(map[string]any)
	if card["type"] != "transaction_result" {
		t.Fatalf("card type = %v, want transaction_result", card["type"])
	}
	items, _ := card["items"].([]any)
	if len(items) != 2 {
		t.Fatalf("items = %d, want 2 (body: %s)", len(items), r.Raw)
	}
	wantAmounts := []float64{25000, 18000}
	for i, item := range items {
		if got := item.(map[string]any)["amount"]; got != wantAmounts[i] {
			t.Errorf("items[%d].amount = %v, want %v", i, got, wantAmounts[i])
		}
	}

	// Both rows landed, and the balance moved by their sum only.
	if after := e.balance(user, acc); after != before-43000 {
		t.Fatalf("balance = %v, want %v", after, before-43000)
	}
	list := e.call(user, http.MethodGet, "/api/v1/transactions?limit=10", nil, "")
	if rows, _ := dataOf(list)["items"].([]any); len(rows) != 2 {
		t.Fatalf("stored rows = %d, want 2", len(rows))
	}
}

// A sentence where one item lacks an amount must not be split: it stays a
// single transaction so nothing is dropped.
func TestMultiTransactionFallsBackWhenAmountMissing(t *testing.T) {
	e := newEnv(t)
	user := e.newUser()
	acc := e.defaultAccount(user)
	before := e.balance(user, acc)

	r := e.call(user, http.MethodPost, "/api/v1/assistant/messages", map[string]any{
		"text": "beli makan 25rb dan kopi", "source": "text",
	}, "")
	if r.Status != http.StatusOK {
		t.Fatalf("assistant = %d: %s", r.Status, r.Raw)
	}
	card := dataOf(r)["card"].(map[string]any)
	if card["type"] != "transaction_result" {
		t.Fatalf("card type = %v, want transaction_result", card["type"])
	}
	if items, _ := card["items"].([]any); len(items) != 0 {
		t.Fatalf("items = %d, want the single path (no items)", len(items))
	}
	if after := e.balance(user, acc); after != before-25000 {
		t.Fatalf("balance = %v, want %v", after, before-25000)
	}
}

// PRD §22.3 / ISSUE-13: "ubah kopi jadi 15rb" edits the existing row through
// the versioned update path instead of creating a new transaction.
func TestChatCorrectionUpdatesTransactionAndAudits(t *testing.T) {
	e := newEnv(t)
	user := e.newUser()
	acc := e.defaultAccount(user)

	// A budget must exist so the sentence "ubah budget jadi 2jt" keeps going to
	// the proposal handler rather than the correction handler.
	if r := e.call(user, http.MethodPost, "/api/v1/budgets", map[string]any{
		"period": "this_month", "overallLimit": 1000000,
	}, ""); r.Status != http.StatusCreated {
		t.Fatalf("budget = %d", r.Status)
	}

	created := e.call(user, http.MethodPost, "/api/v1/assistant/messages", map[string]any{
		"text": "catat kopi susu 18rb", "source": "text",
	}, "")
	if created.Status != http.StatusOK {
		t.Fatalf("create = %d: %s", created.Status, created.Raw)
	}
	createdCard := dataOf(created)["card"].(map[string]any)
	txID := createdCard["transaction"].(map[string]any)["id"].(string)
	afterCreate := e.balance(user, acc)

	corrected := e.call(user, http.MethodPost, "/api/v1/assistant/messages", map[string]any{
		"text": "ubah kopi jadi 15rb", "source": "text",
	}, "")
	if corrected.Status != http.StatusOK {
		t.Fatalf("correct = %d: %s", corrected.Status, corrected.Raw)
	}
	card := dataOf(corrected)["card"].(map[string]any)
	if card["type"] != "transaction_result" {
		t.Fatalf("card type = %v, want transaction_result (body: %s)", card["type"], corrected.Raw)
	}
	transaction := card["transaction"].(map[string]any)
	if transaction["id"] != txID {
		t.Fatalf("correction created a new row: %v vs %v", transaction["id"], txID)
	}
	if transaction["amount"] != float64(15000) {
		t.Fatalf("amount = %v, want 15000", transaction["amount"])
	}
	if transaction["version"].(float64) < 2 {
		t.Fatalf("version = %v, want >= 2 after an edit", transaction["version"])
	}

	// Still exactly one row, and the balance moved by the difference only.
	list := e.call(user, http.MethodGet, "/api/v1/transactions?limit=10", nil, "")
	if rows, _ := dataOf(list)["items"].([]any); len(rows) != 1 {
		t.Fatalf("rows = %d, want 1", len(rows))
	}
	// Lowering an expense from 18000 to 15000 leaves 3000 more in the account.
	if after := e.balance(user, acc); after != afterCreate+3000 {
		t.Fatalf("balance = %v, want %v", after, afterCreate+3000)
	}

	// The audit trail records the edit.
	var updates int
	if err := e.pool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM audit_logs
		 WHERE entity_type = 'transaction' AND entity_id = $1 AND action = 'update'`,
		txID).Scan(&updates); err != nil {
		t.Fatalf("audit query: %v", err)
	}
	if updates == 0 {
		t.Fatal("correction left no audit row")
	}
}

// A correction with no matching transaction must clarify, not invent a row.
func TestChatCorrectionWithoutMatchClarifies(t *testing.T) {
	e := newEnv(t)
	user := e.newUser()
	acc := e.defaultAccount(user)
	before := e.balance(user, acc)

	r := e.call(user, http.MethodPost, "/api/v1/assistant/messages", map[string]any{
		"text": "ubah sepatu jadi 500rb", "source": "text",
	}, "")
	if r.Status != http.StatusOK {
		t.Fatalf("correct = %d: %s", r.Status, r.Raw)
	}
	card := dataOf(r)["card"].(map[string]any)
	if card["type"] != "clarification" {
		t.Fatalf("card type = %v, want clarification (body: %s)", card["type"], r.Raw)
	}
	list := e.call(user, http.MethodGet, "/api/v1/transactions?limit=10", nil, "")
	if rows, _ := dataOf(list)["items"].([]any); len(rows) != 0 {
		t.Fatalf("rows = %d, want 0", len(rows))
	}
	if after := e.balance(user, acc); after != before {
		t.Fatalf("balance changed: %v -> %v", before, after)
	}
}

// "ubah budget jadi 2jt" must stay a budget proposal (regression guard for the
// correction detector stealing budget sentences).
func TestBudgetCorrectionStillCreatesProposal(t *testing.T) {
	e := newEnv(t)
	user := e.newUser()

	if r := e.call(user, http.MethodPost, "/api/v1/budgets", map[string]any{
		"period": "this_month", "overallLimit": 1000000,
	}, ""); r.Status != http.StatusCreated {
		t.Fatalf("budget = %d", r.Status)
	}

	r := e.call(user, http.MethodPost, "/api/v1/assistant/messages", map[string]any{
		"text": "ubah budget jadi 2jt", "source": "text",
	}, "")
	if r.Status != http.StatusOK {
		t.Fatalf("assistant = %d: %s", r.Status, r.Raw)
	}
	card := dataOf(r)["card"].(map[string]any)
	if card["type"] != "budget_proposal" {
		t.Fatalf("card type = %v, want budget_proposal (body: %s)", card["type"], r.Raw)
	}
}
