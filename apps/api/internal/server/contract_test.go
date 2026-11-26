package server_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

// The contract under test. Keeping it in docs/api/openapi.yaml means the same
// document drives the TypeScript types (npm run gen:api) and this test, so a
// DTO change cannot slip past either side (plan.md §12.3, ISSUE-19).
const contractPath = "../../../../docs/api/openapi.yaml"

type oasSchema struct {
	Ref                  string               `yaml:"$ref"`
	Type                 string               `yaml:"type"`
	Format               string               `yaml:"format"`
	Properties           map[string]oasSchema `yaml:"properties"`
	Required             []string             `yaml:"required"`
	Items                *oasSchema           `yaml:"items"`
	Enum                 []string             `yaml:"enum"`
	Nullable             bool                 `yaml:"nullable"`
	AdditionalProperties *bool                `yaml:"additionalProperties"`
	OneOf                []oasSchema          `yaml:"oneOf"`
}

type oasMediaType struct {
	Schema oasSchema `yaml:"schema"`
}

type oasResponse struct {
	Ref         string                  `yaml:"$ref"`
	Description string                  `yaml:"description"`
	Content     map[string]oasMediaType `yaml:"content"`
}

type oasOperation struct {
	Responses map[string]oasResponse `yaml:"responses"`
}

type oasPathItem struct {
	Parameters []map[string]any `yaml:"parameters"`
	Get        *oasOperation    `yaml:"get"`
	Post       *oasOperation    `yaml:"post"`
	Patch      *oasOperation    `yaml:"patch"`
	Delete     *oasOperation    `yaml:"delete"`
}

func (p *oasPathItem) operation(method string) *oasOperation {
	switch strings.ToUpper(method) {
	case http.MethodGet:
		return p.Get
	case http.MethodPost:
		return p.Post
	case http.MethodPatch:
		return p.Patch
	case http.MethodDelete:
		return p.Delete
	}
	return nil
}

func (p *oasPathItem) operations() map[string]*oasOperation {
	out := map[string]*oasOperation{}
	for _, method := range []string{"get", "post", "patch", "delete"} {
		if op := p.operation(method); op != nil {
			out[method] = op
		}
	}
	return out
}

type oasDocument struct {
	Paths      map[string]oasPathItem `yaml:"paths"`
	Components struct {
		Schemas   map[string]oasSchema   `yaml:"schemas"`
		Responses map[string]oasResponse `yaml:"responses"`
	} `yaml:"components"`
}

func (d *oasDocument) schemaAt(ref string) (*oasSchema, bool) {
	prefix := "#/components/schemas/"
	if !strings.HasPrefix(ref, prefix) {
		return nil, false
	}
	s, ok := d.Components.Schemas[strings.TrimPrefix(ref, prefix)]
	return &s, ok
}

func (d *oasDocument) responseAt(ref string) (*oasResponse, bool) {
	prefix := "#/components/responses/"
	if !strings.HasPrefix(ref, prefix) {
		return nil, false
	}
	r, ok := d.Components.Responses[strings.TrimPrefix(ref, prefix)]
	return &r, ok
}

func (o *oasOperation) response(d *oasDocument, status string) (*oasResponse, bool) {
	resp, ok := o.Responses[status]
	if !ok {
		return nil, false
	}
	if resp.Ref != "" {
		return d.responseAt(resp.Ref)
	}
	return &resp, true
}

func (o *oasOperation) statuses() []string {
	out := make([]string, 0, len(o.Responses))
	for status := range o.Responses {
		out = append(out, status)
	}
	return out
}

// ---------------------------------------------------------------------------
// Minimal JSON-Schema validator for the subset OpenAPI 3.0 uses here:
// $ref, oneOf, type, properties, required, items, enum, nullable,
// additionalProperties. Anything undocumented in a response is a contract
// break, which is what makes DTO drift fail this test.
// ---------------------------------------------------------------------------

func typeName(value any) string {
	switch value.(type) {
	case nil:
		return "null"
	case string:
		return "string"
	case bool:
		return "boolean"
	case float64:
		return "number"
	case []any:
		return "array"
	case map[string]any:
		return "object"
	}
	return fmt.Sprintf("%T", value)
}

func (d *oasDocument) validate(schema *oasSchema, value any, path string, errs *[]string) {
	if schema == nil {
		return
	}
	if schema.Ref != "" {
		resolved, ok := d.schemaAt(schema.Ref)
		if !ok {
			*errs = append(*errs, path+": unresolved $ref "+schema.Ref)
			return
		}
		d.validate(resolved, value, path, errs)
		return
	}
	if len(schema.OneOf) > 0 {
		var failures []string
		for i := range schema.OneOf {
			var branch []string
			d.validate(&schema.OneOf[i], value, path, &branch)
			if len(branch) == 0 {
				return
			}
			failures = append(failures, strings.Join(branch, "; "))
		}
		*errs = append(*errs, path+": value matches no oneOf branch ("+strings.Join(failures, " | ")+")")
		return
	}

	if value == nil {
		// `type: "null"` (oneOf branch) and `nullable: true` both accept null.
		if schema.Nullable || schema.Type == "" || schema.Type == "null" {
			return
		}
		*errs = append(*errs, path+": null is not allowed for type "+schema.Type)
		return
	}
	if schema.Type == "null" {
		*errs = append(*errs, path+": expected null, got "+typeName(value))
		return
	}

	switch schema.Type {
	case "object":
		object, ok := value.(map[string]any)
		if !ok {
			*errs = append(*errs, path+": expected object, got "+typeName(value))
			return
		}
		for _, required := range schema.Required {
			if _, present := object[required]; !present {
				*errs = append(*errs, path+"."+required+": required property is missing")
			}
		}
		for key, item := range object {
			property, documented := schema.Properties[key]
			if !documented {
				if schema.AdditionalProperties != nil && !*schema.AdditionalProperties {
					*errs = append(*errs, path+"."+key+": property is not in the contract")
				}
				continue
			}
			d.validate(&property, item, path+"."+key, errs)
		}
	case "array":
		items, ok := value.([]any)
		if !ok {
			*errs = append(*errs, path+": expected array, got "+typeName(value))
			return
		}
		if schema.Items != nil {
			for i, item := range items {
				d.validate(schema.Items, item, fmt.Sprintf("%s[%d]", path, i), errs)
			}
		}
	case "string":
		if _, ok := value.(string); !ok {
			*errs = append(*errs, path+": expected string, got "+typeName(value))
		}
	case "integer":
		number, ok := value.(float64)
		if !ok || number != math.Trunc(number) {
			*errs = append(*errs, path+": expected integer, got "+typeName(value))
		}
	case "number":
		if _, ok := value.(float64); !ok {
			*errs = append(*errs, path+": expected number, got "+typeName(value))
		}
	case "boolean":
		if _, ok := value.(bool); !ok {
			*errs = append(*errs, path+": expected boolean, got "+typeName(value))
		}
	}

	if len(schema.Enum) > 0 {
		text := fmt.Sprint(value)
		for _, allowed := range schema.Enum {
			if allowed == text {
				return
			}
		}
		*errs = append(*errs, path+": "+text+" is not one of "+strings.Join(schema.Enum, "|"))
	}
}

func loadContract(t *testing.T) *oasDocument {
	t.Helper()
	raw, err := os.ReadFile(filepath.Clean(contractPath))
	if err != nil {
		t.Fatalf("read contract: %v", err)
	}
	var doc oasDocument
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("parse contract: %v", err)
	}
	if len(doc.Paths) == 0 {
		t.Fatal("contract declares no paths")
	}
	return &doc
}

// TestContractMatchesLiveAPI drives every documented endpoint and validates the
// actual response against docs/api/openapi.yaml. A DTO change that is not
// reflected in the contract fails here.
func TestContractMatchesLiveAPI(t *testing.T) {
	doc := loadContract(t)
	env := newEnv(t)
	token := env.newUser()

	exercised := map[string]bool{}
	collectCardTypes := map[string]bool{}
	collectErrorCodes := map[string]bool{}

	// call performs one request, asserts the status is documented, then
	// validates the body against the schema for that status.
	call := func(method, template, path string, body any, idempotencyKey string, wantStatus int) map[string]any {
		t.Helper()
		key := strings.ToLower(method) + " " + template
		exercised[key] = true

		item, ok := doc.Paths[template]
		if !ok {
			t.Fatalf("contract has no path %q", template)
		}
		op := item.operation(method)
		if op == nil {
			t.Fatalf("contract has no %s %q", method, template)
		}

		response := env.call(token, method, path, body, idempotencyKey)
		if response.Status != wantStatus {
			t.Fatalf("%s %s: status = %d, want %d (body: %s)", method, template, response.Status, wantStatus, response.Raw)
		}
		resp, ok := op.response(doc, strconv.Itoa(wantStatus))
		if !ok {
			t.Fatalf("%s %s: status %d is not documented; documented: %v",
				method, template, wantStatus, op.statuses())
		}
		media, ok := resp.Content["application/json"]
		if !ok {
			t.Fatalf("%s %s [%d]: no application/json response documented", method, template, wantStatus)
		}
		var errs []string
		doc.validate(&media.Schema, response.Body, template, &errs)
		if len(errs) > 0 {
			t.Fatalf("%s %s [%d] does not match the contract:\n  %s",
				method, template, wantStatus, strings.Join(errs, "\n  "))
		}

		if wantStatus >= 400 {
			if envelope, ok := response.Body["error"].(map[string]any); ok {
				if code, ok := envelope["code"].(string); ok {
					collectErrorCodes[code] = true
				}
			}
		}
		if data, ok := response.Body["data"].(map[string]any); ok {
			if card, ok := data["card"].(map[string]any); ok {
				if cardType, ok := card["type"].(string); ok {
					collectCardTypes[cardType] = true
				}
			}
		}
		return response.Body
	}

	// --- system + identity ---------------------------------------------------
	call(http.MethodGet, "/health", "/health", nil, "", http.StatusOK)

	// No token at all: proves the 401 envelope and the UNAUTHORIZED code.
	unauthorized := rawCall(env, "", http.MethodGet, "/api/v1/me", nil)
	if unauthorized.Status != http.StatusUnauthorized {
		t.Fatalf("me without token: status = %d, want 401 (body: %s)", unauthorized.Status, unauthorized.Raw)
	}
	validateStatus(t, doc, "/api/v1/me", http.MethodGet, unauthorized.Status, unauthorized.Body, collectErrorCodes)
	exercised[strings.ToLower(http.MethodGet)+" /api/v1/me"] = true

	me := call(http.MethodGet, "/api/v1/me", "/api/v1/me", nil, "", http.StatusOK)
	_ = me
	call(http.MethodPatch, "/api/v1/me", "/api/v1/me",
		map[string]any{"displayName": "Contract Tester"}, "", http.StatusOK)

	// --- categories ----------------------------------------------------------
	call(http.MethodGet, "/api/v1/categories", "/api/v1/categories", nil, "", http.StatusOK)
	createdCategory := call(http.MethodPost, "/api/v1/categories", "/api/v1/categories",
		map[string]any{"name": "Contract Kategori", "kind": "expense"}, "", http.StatusCreated)
	categoryID := dataID(t, createdCategory)
	call(http.MethodPatch, "/api/v1/categories/{id}", "/api/v1/categories/"+categoryID,
		map[string]any{"name": "Contract Kategori Baru"}, "", http.StatusOK)
	call(http.MethodDelete, "/api/v1/categories/{id}", "/api/v1/categories/"+categoryID, nil, "", http.StatusOK)

	// --- accounts ------------------------------------------------------------
	call(http.MethodGet, "/api/v1/accounts", "/api/v1/accounts", nil, "", http.StatusOK)
	bank := call(http.MethodPost, "/api/v1/accounts", "/api/v1/accounts",
		map[string]any{"name": "Contract Bank", "type": "bank", "openingBalance": 100000}, "", http.StatusCreated)
	bankID := dataID(t, bank)
	call(http.MethodPatch, "/api/v1/accounts/{id}", "/api/v1/accounts/"+bankID,
		map[string]any{"name": "Contract Bank Utama"}, "", http.StatusOK)
	call(http.MethodPost, "/api/v1/accounts/{id}/default", "/api/v1/accounts/"+bankID+"/default", nil, "", http.StatusOK)

	archive := call(http.MethodPost, "/api/v1/accounts", "/api/v1/accounts",
		map[string]any{"name": "Contract Arsip", "type": "cash"}, "", http.StatusCreated)
	call(http.MethodDelete, "/api/v1/accounts/{id}", "/api/v1/accounts/"+dataID(t, archive), nil, "", http.StatusOK)

	// --- transactions --------------------------------------------------------
	created := call(http.MethodPost, "/api/v1/transactions", "/api/v1/transactions", map[string]any{
		"type": "expense", "amount": 25000, "description": "Contract kopi",
		"accountId": bankID, "source": "manual",
	}, "contract-tx-1", http.StatusCreated)
	transactionID := created["data"].(map[string]any)["transaction"].(map[string]any)["id"].(string)

	call(http.MethodGet, "/api/v1/transactions", "/api/v1/transactions?limit=5", nil, "", http.StatusOK)
	call(http.MethodGet, "/api/v1/transactions/search", "/api/v1/transactions/search?q=kopi", nil, "", http.StatusOK)
	call(http.MethodGet, "/api/v1/transactions/{id}", "/api/v1/transactions/"+transactionID, nil, "", http.StatusOK)
	call(http.MethodPatch, "/api/v1/transactions/{id}", "/api/v1/transactions/"+transactionID,
		map[string]any{"version": 1, "description": "Contract kopi diubah"}, "", http.StatusOK)

	// --- analytics -----------------------------------------------------------
	call(http.MethodGet, "/api/v1/analytics/dashboard", "/api/v1/analytics/dashboard?period=this_month", nil, "", http.StatusOK)
	call(http.MethodGet, "/api/v1/analytics/financial-summary", "/api/v1/analytics/financial-summary?period=this_month", nil, "", http.StatusOK)
	call(http.MethodGet, "/api/v1/analytics/largest-transactions", "/api/v1/analytics/largest-transactions?period=this_month", nil, "", http.StatusOK)
	call(http.MethodGet, "/api/v1/analytics/unusual-spending", "/api/v1/analytics/unusual-spending?period=this_month", nil, "", http.StatusOK)
	call(http.MethodGet, "/api/v1/analytics/spending-comparison", "/api/v1/analytics/spending-comparison?period=this_month", nil, "", http.StatusOK)
	call(http.MethodGet, "/api/v1/analytics/cash-flow", "/api/v1/analytics/cash-flow?period=this_month&bucket=day", nil, "", http.StatusOK)

	// --- budgets -------------------------------------------------------------
	call(http.MethodPost, "/api/v1/budgets", "/api/v1/budgets",
		map[string]any{"period": "this_month", "overallLimit": 1000000}, "", http.StatusCreated)
	budgets := call(http.MethodGet, "/api/v1/budgets", "/api/v1/budgets", nil, "", http.StatusOK)
	budgetID := budgets["data"].([]any)[0].(map[string]any)["id"].(string)
	budgetVersion := budgets["data"].([]any)[0].(map[string]any)["version"].(float64)

	call(http.MethodGet, "/api/v1/budgets/{id}", "/api/v1/budgets/"+budgetID, nil, "", http.StatusOK)
	call(http.MethodPatch, "/api/v1/budgets/{id}", "/api/v1/budgets/"+budgetID,
		map[string]any{"version": int(budgetVersion), "overallLimit": 1500000}, "", http.StatusOK)
	call(http.MethodGet, "/api/v1/budgets/{id}/status", "/api/v1/budgets/"+budgetID+"/status", nil, "", http.StatusOK)

	// --- budget proposals ----------------------------------------------------
	proposal := call(http.MethodPost, "/api/v1/budget-proposals", "/api/v1/budget-proposals",
		map[string]any{"budgetId": budgetID, "changes": map[string]any{"overallLimit": 2000000}}, "", http.StatusCreated)
	proposalID := dataID(t, proposal)
	proposalVersion := proposal["data"].(map[string]any)["baseVersion"].(float64)

	call(http.MethodGet, "/api/v1/budget-proposals", "/api/v1/budget-proposals", nil, "", http.StatusOK)
	call(http.MethodGet, "/api/v1/budget-proposals/{id}", "/api/v1/budget-proposals/"+proposalID, nil, "", http.StatusOK)
	call(http.MethodPatch, "/api/v1/budget-proposals/{id}", "/api/v1/budget-proposals/"+proposalID,
		map[string]any{"changes": map[string]any{"overallLimit": 2500000}, "rationale": "contract"}, "", http.StatusOK)
	call(http.MethodPost, "/api/v1/budget-proposals/{id}/approve", "/api/v1/budget-proposals/"+proposalID+"/approve",
		map[string]any{"baseVersion": int(proposalVersion)}, "contract-approve-1", http.StatusOK)

	second := call(http.MethodPost, "/api/v1/budget-proposals", "/api/v1/budget-proposals",
		map[string]any{"budgetId": budgetID, "changes": map[string]any{"overallLimit": 2100000}}, "", http.StatusCreated)
	call(http.MethodPost, "/api/v1/budget-proposals/{id}/reject", "/api/v1/budget-proposals/"+dataID(t, second)+"/reject",
		nil, "", http.StatusOK)

	// --- savings goals + purchase advisor (PRD §10.4, §10.6) ---------------
	createdGoal := call(http.MethodPost, "/api/v1/savings-goals", "/api/v1/savings-goals", map[string]any{
		"name": "Contract Liburan", "targetAmount": 5000000,
		"deadline": futureDeadline(90), "accountId": bankID,
	}, "", http.StatusCreated)
	goalID := dataID(t, createdGoal)

	call(http.MethodGet, "/api/v1/savings-goals", "/api/v1/savings-goals", nil, "", http.StatusOK)
	// `monthly` turns the detail into a projection simulation.
	call(http.MethodGet, "/api/v1/savings-goals/{id}", "/api/v1/savings-goals/"+goalID+"?monthly=500000", nil, "", http.StatusOK)
	call(http.MethodPatch, "/api/v1/savings-goals/{id}", "/api/v1/savings-goals/"+goalID,
		map[string]any{"version": 1, "targetAmount": 6000000}, "", http.StatusOK)
	call(http.MethodGet, "/api/v1/purchase-advisor", "/api/v1/purchase-advisor?price=1500000", nil, "", http.StatusOK)
	call(http.MethodDelete, "/api/v1/savings-goals/{id}", "/api/v1/savings-goals/"+goalID, nil, "", http.StatusOK)

	// --- assistant (drives every reachable card type) ------------------------
	assistantMessages := []string{
		"catat kopi uji 15rb",                          // transaction_result
		"catat 25rb",                                   // clarification
		"berapa pengeluaran bulan ini",                 // financial_summary
		"cari transaksi kopi",                          // historical_search
		"transaksi paling besar bulan ini",             // largest_transaction
		"apakah ada pengeluaran tidak biasa bulan ini", // unusual_spending
		"cek budget bulan ini",                         // budget_status
		"ubah budget jadi 3jt",                         // budget_proposal
		"halo apa kabar",                               // recommendation
		"cek saldo",                                    // account_balance
		"hapus transaksi kopi uji",                     // delete_confirmation (kandidat; penghapusan baru jalan lewat tombol)
	}
	var conversationID string
	for i, message := range assistantMessages {
		body := call(http.MethodPost, "/api/v1/assistant/messages", "/api/v1/assistant/messages",
			map[string]any{"text": message, "source": "text"}, "", http.StatusOK)
		if i == 0 {
			conversationID = body["data"].(map[string]any)["conversationId"].(string)
		}
	}

	call(http.MethodGet, "/api/v1/assistant/conversations", "/api/v1/assistant/conversations", nil, "", http.StatusOK)
	call(http.MethodGet, "/api/v1/assistant/conversations/{id}", "/api/v1/assistant/conversations/"+conversationID, nil, "", http.StatusOK)
	call(http.MethodDelete, "/api/v1/assistant/conversations/{id}", "/api/v1/assistant/conversations/"+conversationID, nil, "", http.StatusOK)

	// --- speech (provider absent → documented error statuses) ---------------
	speechWithoutFile := rawCall(env, token, http.MethodPost, "/api/v1/speech/transcriptions", nil)
	if speechWithoutFile.Status != http.StatusUnprocessableEntity {
		t.Fatalf("speech without file: status = %d, want 422 (body: %s)", speechWithoutFile.Status, speechWithoutFile.Raw)
	}
	validateStatus(t, doc, "/api/v1/speech/transcriptions", http.MethodPost, speechWithoutFile.Status, speechWithoutFile.Body, collectErrorCodes)
	exercised[strings.ToLower(http.MethodPost)+" /api/v1/speech/transcriptions"] = true

	multipartBody, contentType := multipartAudio(t)
	speech := multipartCall(env, token, "/api/v1/speech/transcriptions", multipartBody, contentType)
	if speech.Status != http.StatusServiceUnavailable {
		t.Fatalf("speech with file: status = %d, want 503 without OPENAI_API_KEY (body: %s)", speech.Status, speech.Raw)
	}
	validateStatus(t, doc, "/api/v1/speech/transcriptions", http.MethodPost, speech.Status, speech.Body, collectErrorCodes)

	// --- void ----------------------------------------------------------------
	call(http.MethodPost, "/api/v1/transactions/{id}/void", "/api/v1/transactions/"+transactionID+"/void",
		map[string]any{"reason": "contract void"}, "contract-void-1", http.StatusOK)

	// --- coverage: every documented operation must have been exercised -------
	var missing []string
	for template, item := range doc.Paths {
		for method := range item.operations() {
			if !exercised[method+" "+template] {
				missing = append(missing, strings.ToUpper(method)+" "+template)
			}
		}
	}
	if len(missing) > 0 {
		t.Errorf("contract declares %d endpoints this test never called:\n  %s",
			len(missing), strings.Join(sorted(missing), "\n  "))
	}

	// --- card type contract (plan.md §7.3) ----------------------------------
	cardEnum := doc.Components.Schemas["AssistantCardType"].Enum
	if len(cardEnum) == 0 {
		t.Fatal("contract has no AssistantCardType enum")
	}
	known := map[string]bool{}
	for _, cardType := range cardEnum {
		known[cardType] = true
	}
	for cardType := range collectCardTypes {
		if !known[cardType] {
			t.Errorf("assistant produced card type %q which is not in the contract enum", cardType)
		}
	}
	// `error` is emitted when a service rule rejects an otherwise valid request;
	// it needs a rejection the deterministic path can trigger, so it stays
	// documented-but-unexercised until then.
	delete(known, "error")
	for cardType := range known {
		if !collectCardTypes[cardType] {
			t.Errorf("contract documents card type %q but the test never observed it", cardType)
		}
	}

	// --- error code contract (plan.md §7.1) ---------------------------------
	errorEnum := doc.Components.Schemas["ErrorCode"].Enum
	knownCodes := map[string]bool{}
	for _, code := range errorEnum {
		knownCodes[code] = true
	}
	for code := range collectErrorCodes {
		if !knownCodes[code] {
			t.Errorf("API returned error code %q which is not in the contract enum", code)
		}
	}
	for _, code := range []string{"UNAUTHORIZED", "VALIDATION_ERROR"} {
		if !collectErrorCodes[code] {
			t.Errorf("contract documents error code %q but the test never observed it", code)
		}
	}

	t.Logf("contract: %d paths, %d card types observed, %d error codes observed",
		len(doc.Paths), len(collectCardTypes), len(collectErrorCodes))
}

// validateStatus checks one response body against the contract without going
// through the `call` helper (used for requests with a non-JSON body).
func validateStatus(t *testing.T, doc *oasDocument, template, method string, status int, body any, codes map[string]bool) {
	t.Helper()
	item, ok := doc.Paths[template]
	if !ok {
		t.Fatalf("contract has no path %q", template)
	}
	op := item.operation(method)
	if op == nil {
		t.Fatalf("contract has no %s %q", method, template)
	}
	resp, ok := op.response(doc, strconv.Itoa(status))
	if !ok {
		t.Fatalf("%s %s: status %d is not documented; documented: %v", method, template, status, op.statuses())
	}
	media, ok := resp.Content["application/json"]
	if !ok {
		t.Fatalf("%s %s [%d]: no application/json response documented", method, template, status)
	}
	var errs []string
	doc.validate(&media.Schema, body, template, &errs)
	if len(errs) > 0 {
		t.Fatalf("%s %s [%d] does not match the contract:\n  %s",
			method, template, status, strings.Join(errs, "\n  "))
	}
	if envelope, ok := body.(map[string]any)["error"].(map[string]any); ok {
		if code, ok := envelope["code"].(string); ok {
			codes[code] = true
		}
	}
}

// rawCall sends a request with a raw body (or none) through the real router.
func rawCall(e *testEnv, token, method, path string, body io.Reader) response {
	request := httptest.NewRequest(method, path, body)
	request.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()
	e.router.ServeHTTP(recorder, request)
	return decode(recorder)
}

// multipartCall sends a multipart/form-data request through the real router.
func multipartCall(e *testEnv, token, path string, body *bytes.Buffer, contentType string) response {
	request := httptest.NewRequest(http.MethodPost, path, body)
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Content-Type", contentType)
	recorder := httptest.NewRecorder()
	e.router.ServeHTTP(recorder, request)
	return decode(recorder)
}

func decode(recorder *httptest.ResponseRecorder) response {
	out := response{Status: recorder.Code, Raw: recorder.Body.String()}
	if out.Raw != "" {
		_ = json.Unmarshal(recorder.Body.Bytes(), &out.Body)
	}
	return out
}

// futureDeadline returns a YYYY-MM-DD date n days from now in UTC, which is the
// basis `parseDeadline` compares against.
func futureDeadline(days int) string {
	return time.Now().UTC().AddDate(0, 0, days).Format("2006-01-02")
}

func dataID(t *testing.T, body map[string]any) string {
	t.Helper()
	data, ok := body["data"].(map[string]any)
	if !ok {
		t.Fatalf("response has no data object: %s", body)
	}
	id, ok := data["id"].(string)
	if !ok {
		t.Fatalf("data has no id: %s", data)
	}
	return id
}

func multipartAudio(t *testing.T) (*bytes.Buffer, string) {
	t.Helper()
	buffer := &bytes.Buffer{}
	writer := multipart.NewWriter(buffer)
	part, err := writer.CreateFormFile("audio", "contract.webm")
	if err != nil {
		t.Fatalf("multipart: %v", err)
	}
	if _, err := part.Write([]byte("dummy audio")); err != nil {
		t.Fatalf("multipart write: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("multipart close: %v", err)
	}
	return buffer, writer.FormDataContentType()
}

func sorted(values []string) []string {
	out := append([]string(nil), values...)
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			if out[j] < out[i] {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out
}
