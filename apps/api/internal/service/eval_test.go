package service

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Eval set: docs/eval/ai-cases.jsonl (plan.md §12.4, PRD §22.3).
//
// The runner exercises only the deterministic local pipeline — correction
// detection, the heuristic intent parser, multi-transaction splitting, date and
// period parsing — so it needs no model key and no database. When
// OPENAI_API_KEY is set, this suite still measures the fallback path only.
const evalCasesPath = "../../../../docs/eval/ai-cases.jsonl"

// minEvalAccuracy is the initial go/no-go threshold for the deterministic
// parser (see docs/eval/hasil.md).
const minEvalAccuracy = 0.95

type evalCase struct {
	ID         string          `json:"id"`
	Kategori   string          `json:"kategori"`
	Input      string          `json:"input"`
	Ekspektasi json.RawMessage `json:"ekspektasi"`
	Catatan    string          `json:"catatan"`
}

// expectation only holds the fields a case asserts; absent fields are ignored.
type expectation struct {
	Kind       *string `json:"kind"`
	Type       *string `json:"type"`
	Amount     *int64  `json:"amount"`
	Count      *int    `json:"count"`
	OccurredAt *string `json:"occurredAt"`
	Period     *string `json:"period"`
	Clarify    *bool   `json:"clarify"`
}

type actualResult struct {
	kind       string
	txType     string
	amount     int64
	count      int
	occurredAt string
	period     string
	clarify    bool
}

// evaluate mirrors the order used by HandleMessage: correction first, then the
// heuristic intent, then the multi-transaction split inside handleCreate.
func evaluate(input string, now time.Time, loc *time.Location) actualResult {
	out := actualResult{}

	if corr := detectCorrection(input); corr.match {
		out.kind = "correction"
		if corr.amount != nil {
			out.amount = *corr.amount
		}
	} else {
		intent := heuristicIntent(input, now)
		out.kind = intent.Kind
		out.txType = intent.Type
		out.amount = intent.Amount

		if intent.Kind == "create_transaction" && intent.Type != "transfer" {
			if segments := parseMultiSegments(splitTransactionSegments(input), intent.Type, "chat"); segments != nil {
				out.kind = "multi_transaction"
				out.count = len(segments)
			}
		}
	}

	if date := parseRelativeDate(input, now, loc); date != nil {
		out.occurredAt = date.In(loc).Format("2006-01-02")
	}
	out.period, _ = detectPeriodPhrase(strings.ToLower(input), now.Year())

	// Mirrors the missing-field check in handleCreate.
	if out.kind == "create_transaction" {
		intent := heuristicIntent(input, now)
		description := intent.Description
		if parseRelativeDate(input, now, loc) != nil {
			description = descriptionForDatedText(input, description)
		}
		if description == "" {
			description = cleanDescription(stripDateWords(input))
		}
		out.clarify = out.amount <= 0 || description == "" || isGenericDescription(description)
	}
	return out
}

func loadEvalCases(t *testing.T) []evalCase {
	t.Helper()
	file, err := os.Open(filepath.Clean(evalCasesPath))
	if err != nil {
		t.Fatalf("open eval set: %v", err)
	}
	defer file.Close()

	var cases []evalCase
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 64*1024)
	line := 0
	for scanner.Scan() {
		line++
		text := strings.TrimSpace(scanner.Text())
		if text == "" {
			continue
		}
		var parsed evalCase
		if err := json.Unmarshal([]byte(text), &parsed); err != nil {
			t.Fatalf("line %d: invalid JSON: %v", line, err)
		}
		cases = append(cases, parsed)
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("read eval set: %v", err)
	}
	return cases
}

// TestEvalSetCoversPlannedCategories keeps the file aligned with plan.md §12.4.
func TestEvalSetCoversPlannedCategories(t *testing.T) {
	required := []string{
		"nominal", "tanggal", "multi-transaksi", "code-switching",
		"transfer-vs-expense", "pertanyaan-vs-perintah", "koreksi",
		"ambiguitas", "analitik", "budget",
	}
	cases := loadEvalCases(t)
	if len(cases) < 40 {
		t.Fatalf("eval set has %d cases, plan.md §12.4 requires at least 40", len(cases))
	}

	byCategory := map[string]int{}
	for _, c := range cases {
		byCategory[c.Kategori]++
		if strings.TrimSpace(c.Input) == "" {
			t.Errorf("%s: empty input", c.ID)
		}
		if strings.TrimSpace(c.Catatan) == "" {
			t.Errorf("%s: missing catatan", c.ID)
		}
		if len(c.Ekspektasi) == 0 || string(c.Ekspektasi) == "null" {
			t.Errorf("%s: missing ekspektasi", c.ID)
		}
	}
	for _, category := range required {
		if byCategory[category] < 3 {
			t.Errorf("category %q has %d cases, want at least 3", category, byCategory[category])
		}
	}
	t.Logf("eval set: %d cases across %d categories", len(cases), len(byCategory))
}

// TestEvalDeterministicAccuracy runs every case through the local parser and
// reports intent, entity, date, period, multi and clarification accuracy.
func TestEvalDeterministicAccuracy(t *testing.T) {
	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		t.Fatalf("load location: %v", err)
	}
	// Fixed reference date: all date expectations in the file assume it.
	now := time.Date(2026, time.November, 8, 10, 0, 0, 0, loc)

	cases := loadEvalCases(t)

	var checked, passed int
	metrics := map[string]*[2]int{} // name -> [ok, total]
	mismatches := []string{}

	track := func(metric string, ok bool) {
		slot, seen := metrics[metric]
		if !seen {
			slot = &[2]int{}
			metrics[metric] = slot
		}
		slot[1]++
		if ok {
			slot[0]++
		}
	}

	for _, c := range cases {
		var want expectation
		if err := json.Unmarshal(c.Ekspektasi, &want); err != nil {
			t.Fatalf("%s: invalid ekspektasi: %v", c.ID, err)
		}
		got := evaluate(c.Input, now, loc)

		var problems []string
		if want.Kind != nil {
			ok := got.kind == *want.Kind
			track("intent", ok)
			if !ok {
				problems = append(problems, fmt.Sprintf("kind=%s want=%s", got.kind, *want.Kind))
			}
		}
		if want.Type != nil {
			ok := got.txType == *want.Type
			track("type", ok)
			if !ok {
				problems = append(problems, fmt.Sprintf("type=%s want=%s", got.txType, *want.Type))
			}
		}
		if want.Amount != nil {
			ok := got.amount == *want.Amount
			track("nominal", ok)
			if !ok {
				problems = append(problems, fmt.Sprintf("amount=%d want=%d", got.amount, *want.Amount))
			}
		}
		if want.Count != nil {
			ok := got.count == *want.Count
			track("multi", ok)
			if !ok {
				problems = append(problems, fmt.Sprintf("count=%d want=%d", got.count, *want.Count))
			}
		}
		if want.OccurredAt != nil {
			ok := got.occurredAt == *want.OccurredAt
			track("tanggal", ok)
			if !ok {
				problems = append(problems, fmt.Sprintf("occurredAt=%s want=%s", got.occurredAt, *want.OccurredAt))
			}
		}
		if want.Period != nil {
			ok := got.period == *want.Period
			track("periode", ok)
			if !ok {
				problems = append(problems, fmt.Sprintf("period=%s want=%s", got.period, *want.Period))
			}
		}
		if want.Clarify != nil {
			ok := got.clarify == *want.Clarify
			track("klarifikasi", ok)
			if !ok {
				problems = append(problems, fmt.Sprintf("clarify=%v want=%v", got.clarify, *want.Clarify))
			}
		}

		checked++
		if len(problems) == 0 {
			passed++
			continue
		}
		mismatches = append(mismatches, fmt.Sprintf("%s [%s] %q: %s",
			c.ID, c.Kategori, c.Input, strings.Join(problems, "; ")))
	}

	// Report first: this is a measurement, not only a gate.
	t.Logf("eval deterministic accuracy: %d/%d cases (%.1f%%)", passed, checked,
		float64(passed)/float64(checked)*100)
	for _, metric := range []string{"intent", "type", "nominal", "multi", "tanggal", "periode", "klarifikasi"} {
		if slot, ok := metrics[metric]; ok {
			t.Logf("  %-12s %d/%d (%.1f%%)", metric, slot[0], slot[1],
				float64(slot[0])/float64(slot[1])*100)
		}
	}
	for _, line := range mismatches {
		t.Errorf("mismatch: %s", line)
	}

	if accuracy := float64(passed) / float64(checked); accuracy < minEvalAccuracy {
		t.Errorf("overall accuracy %.1f%% below the go/no-go threshold %.0f%%",
			accuracy*100, minEvalAccuracy*100)
	}
}
