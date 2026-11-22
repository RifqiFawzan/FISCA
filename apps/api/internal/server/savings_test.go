package server_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
)

// createSavingsGoal posts a goal linked to the given account.
func createSavingsGoal(t *testing.T, env *testEnv, token, name string, target int64, accountID string) map[string]any {
	t.Helper()
	response := env.call(token, http.MethodPost, "/api/v1/savings-goals", map[string]any{
		"name": name, "targetAmount": target,
		"deadline": futureDeadline(90), "accountId": accountID,
	}, "")
	if response.Status != http.StatusCreated {
		t.Fatalf("create goal: status = %d, want 201 (body: %s)", response.Status, response.Raw)
	}
	return dataOf(response)
}

// The goal is a target, not a wallet: progress must come from the linked
// account's ledger balance, and creating a goal must not create money
// (PRD O-17, "hindari double counting").
func TestSavingsGoalProgressFollowsLedgerWithoutDoubleCounting(t *testing.T) {
	env := newEnv(t)
	token := env.newUser()
	cash := env.defaultAccount(token)

	pot := env.call(token, http.MethodPost, "/api/v1/accounts", map[string]any{
		"name": "Tabungan", "type": "bank", "openingBalance": 400000,
	}, "")
	if pot.Status != http.StatusCreated {
		t.Fatalf("create pot: status = %d (body: %s)", pot.Status, pot.Raw)
	}
	potID := dataOf(pot)["id"].(string)

	// Snapshot before the goal exists: nothing about a target changes the money.
	before := env.call(token, http.MethodGet, "/api/v1/purchase-advisor?price=1", nil, "")
	availableBefore := dataOf(before)["available"].(float64)

	goal := createSavingsGoal(t, env, token, "Liburan", 1000000, potID)

	after := env.call(token, http.MethodGet, "/api/v1/purchase-advisor?price=1", nil, "")
	availableAfter := dataOf(after)["available"].(float64)
	if availableAfter != availableBefore {
		t.Fatalf("creating a goal changed available money: %v -> %v", availableBefore, availableAfter)
	}
	if availableAfter != 400000 {
		t.Fatalf("available = %v, want 400000", availableAfter)
	}
	if reserved := dataOf(after)["reservedSavings"].(float64); reserved != 600000 {
		t.Fatalf("reservedSavings = %v, want 600000 (target minus saved)", reserved)
	}

	// Moving money between accounts changes neither the total nor the progress.
	if r := env.call(token, http.MethodPost, "/api/v1/transactions", map[string]any{
		"type": "transfer", "amount": 100000, "description": "Setor tabungan",
		"fromAccountId": cash, "toAccountId": potID, "source": "manual",
	}, uuid.NewString()); r.Status != http.StatusCreated {
		t.Fatalf("transfer: status = %d (body: %s)", r.Status, r.Raw)
	}
	moved := env.call(token, http.MethodGet, "/api/v1/purchase-advisor?price=1", nil, "")
	if got := dataOf(moved)["available"].(float64); got != availableBefore {
		t.Fatalf("transfer changed available money: %v, want %v", got, availableBefore)
	}

	detail := env.call(token, http.MethodGet, "/api/v1/savings-goals/"+goal["id"].(string), nil, "")
	if detail.Status != http.StatusOK {
		t.Fatalf("get goal: status = %d", detail.Status)
	}
	data := dataOf(detail)
	if got := data["saved"].(float64); got != 500000 {
		t.Fatalf("saved = %v, want 500000 (ledger balance of the linked account)", got)
	}
	if got := data["remaining"].(float64); got != 500000 {
		t.Fatalf("remaining = %v, want 500000", got)
	}
	if got := data["accountName"]; got != "Tabungan" {
		t.Fatalf("accountName = %v, want Tabungan", got)
	}
	if got := data["status"]; got != "on_track" {
		t.Fatalf("status = %v, want on_track", got)
	}
	if got := data["monthsLeft"].(float64); got < 1 {
		t.Fatalf("monthsLeft = %v, want >= 1", got)
	}
}

// `monthly` turns the detail into a simulation (PRD §10.4).
func TestSavingsGoalProjectionSimulation(t *testing.T) {
	env := newEnv(t)
	token := env.newUser()
	pot := env.call(token, http.MethodPost, "/api/v1/accounts", map[string]any{
		"name": "Tabungan", "type": "bank", "openingBalance": 100000,
	}, "")
	goal := createSavingsGoal(t, env, token, "Laptop", 1000000, dataOf(pot)["id"].(string))

	// A rate that comfortably reaches the target before the deadline.
	rich := env.call(token, http.MethodGet, "/api/v1/savings-goals/"+goal["id"].(string)+"?monthly=1000000", nil, "")
	projection := dataOf(rich)["projection"].(map[string]any)
	if got := projection["monthlyContribution"].(float64); got != 1000000 {
		t.Fatalf("monthlyContribution = %v, want 1000000", got)
	}
	if got := projection["monthsToTarget"].(float64); got < 1 {
		t.Fatalf("monthsToTarget = %v, want >= 1", got)
	}
	if got := projection["achievableByDeadline"].(bool); !got {
		t.Fatal("a 1000000/month rate must be achievable for a 1000000 target")
	}

	// A rate far below what is needed honestly reports the duration it takes,
	// and that it will not make the deadline.
	poor := env.call(token, http.MethodGet, "/api/v1/savings-goals/"+goal["id"].(string)+"?monthly=1", nil, "")
	projection = dataOf(poor)["projection"].(map[string]any)
	if got := projection["monthsToTarget"].(float64); got != 900000 {
		t.Fatalf("monthsToTarget = %v, want 900000 (900000 remaining at 1/month)", got)
	}
	if got := projection["achievableByDeadline"].(bool); got {
		t.Fatal("1 rupiah per month cannot reach the target before the deadline")
	}
}

// Version conflicts and archiving behave like the rest of the API.
func TestSavingsGoalVersionConflictAndArchive(t *testing.T) {
	env := newEnv(t)
	token := env.newUser()
	account := env.defaultAccount(token)
	goal := createSavingsGoal(t, env, token, "Dana Darurat", 5000000, account)
	goalID := goal["id"].(string)

	patched := env.call(token, http.MethodPatch, "/api/v1/savings-goals/"+goalID,
		map[string]any{"version": 1, "targetAmount": 6000000}, "")
	if patched.Status != http.StatusOK {
		t.Fatalf("patch: status = %d (body: %s)", patched.Status, patched.Raw)
	}
	if got := dataOf(patched)["targetAmount"].(float64); got != 6000000 {
		t.Fatalf("targetAmount = %v, want 6000000", got)
	}
	if got := dataOf(patched)["version"].(float64); got != 2 {
		t.Fatalf("version = %v, want 2", got)
	}

	stale := env.call(token, http.MethodPatch, "/api/v1/savings-goals/"+goalID,
		map[string]any{"version": 1, "targetAmount": 7000000}, "")
	if stale.Status != http.StatusConflict {
		t.Fatalf("stale patch: status = %d, want 409 (body: %s)", stale.Status, stale.Raw)
	}

	archived := env.call(token, http.MethodDelete, "/api/v1/savings-goals/"+goalID, nil, "")
	if archived.Status != http.StatusOK {
		t.Fatalf("archive: status = %d (body: %s)", archived.Status, archived.Raw)
	}
	if got := env.call(token, http.MethodGet, "/api/v1/savings-goals/"+goalID, nil, ""); got.Status != http.StatusNotFound {
		t.Fatalf("archived goal: status = %d, want 404", got.Status)
	}
	list := env.call(token, http.MethodGet, "/api/v1/savings-goals", nil, "")
	if items, ok := list.Body["data"].([]any); ok && len(items) > 0 {
		t.Fatalf("archived goal still listed: %v", items)
	}
}

// Another user must not see, change or remove the goal (PRD §23 exit #19).
func TestSavingsGoalCrossUserIsolation(t *testing.T) {
	env := newEnv(t)
	owner := env.newUser()
	ownerAccount := env.defaultAccount(owner)
	goal := createSavingsGoal(t, env, owner, "Rahasia", 900000, ownerAccount)

	stranger := env.newUser()
	// (explicit calls below so each failure names the operation)
	if r := env.call(stranger, http.MethodGet, "/api/v1/savings-goals/"+goal["id"].(string), nil, ""); r.Status != http.StatusNotFound {
		t.Fatalf("stranger get: status = %d, want 404", r.Status)
	}
	if r := env.call(stranger, http.MethodPatch, "/api/v1/savings-goals/"+goal["id"].(string),
		map[string]any{"version": 1, "targetAmount": 1}, ""); r.Status != http.StatusNotFound {
		t.Fatalf("stranger patch: status = %d, want 404", r.Status)
	}
	if r := env.call(stranger, http.MethodDelete, "/api/v1/savings-goals/"+goal["id"].(string), nil, ""); r.Status != http.StatusNotFound {
		t.Fatalf("stranger delete: status = %d, want 404", r.Status)
	}

	// The owner's goal is untouched and the stranger sees an empty list.
	still := env.call(owner, http.MethodGet, "/api/v1/savings-goals/"+goal["id"].(string), nil, "")
	if still.Status != http.StatusOK {
		t.Fatalf("owner lost the goal: status = %d", still.Status)
	}
	if got := dataOf(still)["targetAmount"].(float64); got != 900000 {
		t.Fatalf("targetAmount = %v, want 900000", got)
	}
	strangerList := env.call(stranger, http.MethodGet, "/api/v1/savings-goals", nil, "")
	if items, ok := strangerList.Body["data"].([]any); ok && len(items) > 0 {
		t.Fatalf("stranger sees %d goals", len(items))
	}
}

// Input that would produce a meaningless target is rejected up front.
func TestSavingsGoalValidation(t *testing.T) {
	env := newEnv(t)
	owner := env.newUser()
	ownerAccount := env.defaultAccount(owner)
	stranger := env.newUser()

	cases := []struct {
		name string
		body map[string]any
	}{
		{"empty name", map[string]any{"name": "  ", "targetAmount": 100000, "deadline": futureDeadline(10), "accountId": ownerAccount}},
		{"zero target", map[string]any{"name": "A", "targetAmount": 0, "deadline": futureDeadline(10), "accountId": ownerAccount}},
		{"deadline in the past", map[string]any{"name": "A", "targetAmount": 100000, "deadline": "2020-01-01", "accountId": ownerAccount}},
		{"malformed deadline", map[string]any{"name": "A", "targetAmount": 100000, "deadline": "kemarin", "accountId": ownerAccount}},
		{"unknown account", map[string]any{"name": "A", "targetAmount": 100000, "deadline": futureDeadline(10), "accountId": uuid.NewString()}},
		{"missing account", map[string]any{"name": "A", "targetAmount": 100000, "deadline": futureDeadline(10)}},
		{"account of another user", map[string]any{"name": "A", "targetAmount": 100000, "deadline": futureDeadline(10), "accountId": env.defaultAccount(stranger)}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := env.call(owner, http.MethodPost, "/api/v1/savings-goals", tc.body, "")
			if r.Status != http.StatusUnprocessableEntity {
				t.Fatalf("status = %d, want 422 (body: %s)", r.Status, r.Raw)
			}
			if code := r.Body["error"].(map[string]any)["code"]; code != "VALIDATION_ERROR" {
				t.Fatalf("code = %v, want VALIDATION_ERROR", code)
			}
		})
	}
}

// Purchase affordability is computed from the ledger and always states what it
// could not verify (PRD §10.6–10.7, §26 guardrail "AI tidak boleh mengarang").
func TestPurchaseAdvisorUsesLedgerAndStatesLimits(t *testing.T) {
	env := newEnv(t)
	token := env.newUser()

	pot := env.call(token, http.MethodPost, "/api/v1/accounts", map[string]any{
		"name": "Tabungan", "type": "bank", "openingBalance": 400000,
	}, "")

	// Within the money that exists → affordable.
	cheap := env.call(token, http.MethodGet, "/api/v1/purchase-advisor?price=100000", nil, "")
	if cheap.Status != http.StatusOK {
		t.Fatalf("advisor: status = %d (body: %s)", cheap.Status, cheap.Raw)
	}
	data := dataOf(cheap)
	if got := data["available"].(float64); got != 400000 {
		t.Fatalf("available = %v, want 400000", got)
	}
	if got := data["freeToSpend"].(float64); got != 400000 {
		t.Fatalf("freeToSpend = %v, want 400000", got)
	}
	if got := data["verdict"]; got != "affordable" {
		t.Fatalf("verdict = %v, want affordable", got)
	}
	if got := data["projectedAfterPurchase"].(float64); got != 300000 {
		t.Fatalf("projectedAfterPurchase = %v, want 300000", got)
	}
	assertContains(t, data["limitations"], "Kewajiban")
	assertContains(t, data["assumptions"], "ledger")

	// Goals reserve their shortfall, so the same rupiah is not offered twice.
	if r := env.call(token, http.MethodPost, "/api/v1/savings-goals", map[string]any{
		"name": "Kamera", "targetAmount": 2000000,
		"deadline": futureDeadline(60), "accountId": dataOf(pot)["id"].(string),
	}, ""); r.Status != http.StatusCreated {
		t.Fatalf("goal: status = %d (body: %s)", r.Status, r.Raw)
	}
	reserved := env.call(token, http.MethodGet, "/api/v1/purchase-advisor?price=100000", nil, "")
	reservedData := dataOf(reserved)
	if got := reservedData["reservedSavings"].(float64); got != 1600000 {
		t.Fatalf("reservedSavings = %v, want 1600000", got)
	}
	if got := reservedData["freeToSpend"].(float64); got != -1200000 {
		t.Fatalf("freeToSpend = %v, want -1200000", got)
	}
	if got := reservedData["verdict"]; got != "not_affordable" {
		t.Fatalf("verdict = %v, want not_affordable", got)
	}
	assertContains(t, reservedData["limitations"], "cicilan")

	// Missing or nonsense price is a validation error, not a guess.
	if r := env.call(token, http.MethodGet, "/api/v1/purchase-advisor", nil, ""); r.Status != http.StatusUnprocessableEntity {
		t.Fatalf("missing price: status = %d, want 422", r.Status)
	}
	if r := env.call(token, http.MethodGet, "/api/v1/purchase-advisor?price=0", nil, ""); r.Status != http.StatusUnprocessableEntity {
		t.Fatalf("zero price: status = %d, want 422", r.Status)
	}
}

func assertContains(t *testing.T, value any, needle string) {
	t.Helper()
	items, ok := value.([]any)
	if !ok {
		t.Fatalf("expected a list of strings, got %T", value)
	}
	for _, item := range items {
		if text, ok := item.(string); ok && strings.Contains(text, needle) {
			return
		}
	}
	t.Fatalf("no entry containing %q in %v", needle, items)
}
