package service

import (
	"testing"
	"time"

	"github.com/fisca-app/fisca/apps/api/internal/infrastructure/database/generated"
	"github.com/fisca-app/fisca/apps/api/internal/period"
)

func TestParseRupiahNumber(t *testing.T) {
	cases := []struct {
		raw    string
		suffix string
		want   int64
		ok     bool
	}{
		{"25", "rb", 25000, true},
		{"25", "ribu", 25000, true},
		{"25", "k", 25000, true},
		{"1,5", "jt", 1500000, true},
		{"2", "juta", 2000000, true},
		{"10.000", "", 10000, true},
		{"25000", "", 25000, true},
		{"", "", 0, false},
		{"0", "", 0, false},
		{"abc", "", 0, false},
	}
	for _, tc := range cases {
		got, ok := parseRupiahNumber(tc.raw, tc.suffix)
		if ok != tc.ok || got != tc.want {
			t.Errorf("parseRupiahNumber(%q, %q) = (%d, %v), want (%d, %v)",
				tc.raw, tc.suffix, got, ok, tc.want, tc.ok)
		}
	}
}

func TestDetectPeriodPhrase(t *testing.T) {
	cases := []struct {
		text string
		want string
	}{
		{"pengeluaran bulan ini", "this_month"},
		{"bulan lalu berapa", "last_month"},
		{"7 hari terakhir", "7d"},
		{"30 hari", "30d"},
		{"tahun ini", "this_year"},
		{"tahun lalu", "last_year"},
		{"belanja maret", "2026-03"},
		{"pengeluaran selama tiga bulan terakhir", "3m"},
		{"3 bulan terakhir", "3m"},
		{"6 bulan terakhir", "6m"},
		{"dua belas bulan terakhir", "12m"},
		{"tidak ada periode", ""},
	}
	for _, tc := range cases {
		got, _ := detectPeriodPhrase(tc.text, 2026)
		if got != tc.want {
			t.Errorf("detectPeriodPhrase(%q) = %q, want %q", tc.text, got, tc.want)
		}
	}
}

// Every period key the phrase detector produces must be accepted by
// period.Parse, otherwise the search would fail after parsing (ISSUE-14).
func TestDetectedPeriodParses(t *testing.T) {
	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		t.Fatalf("load location: %v", err)
	}
	now := time.Date(2026, time.November, 8, 10, 0, 0, 0, loc)

	texts := []string{
		"pengeluaran bulan ini", "bulan lalu berapa", "7 hari terakhir",
		"30 hari", "90 hari", "tahun ini", "belanja maret",
		"pengeluaran selama tiga bulan terakhir", "dua bulan terakhir",
	}
	for _, text := range texts {
		key, label := detectPeriodPhrase(text, 2026)
		if key == "" {
			t.Errorf("detectPeriodPhrase(%q) produced no key", text)
			continue
		}
		r, err := period.Parse(key, loc, now)
		if err != nil {
			t.Errorf("period.Parse(%q) from %q: %v", key, text, err)
			continue
		}
		if r.Label == "" || r.Bucket == "" {
			t.Errorf("Parse(%q) = label %q bucket %q, both required", key, r.Label, r.Bucket)
		}
		if label == "" {
			t.Errorf("detectPeriodPhrase(%q) produced an empty label", text)
		}
	}
}

// The leftover keyword must not contain the words we already turned into
// structured filters, otherwise the SQL text filter finds nothing.
func TestLeftoverTextStripsConsumedTokens(t *testing.T) {
	filters := SearchFilters{Period: "this_month", Type: "expense", Text: "10rb"}
	got := leftoverText("pengeluaran bulan ini di atas 10rb", filters, nil)
	if got != "" {
		t.Errorf("leftoverText = %q, want empty", got)
	}

	got = leftoverText("cari transaksi kopi", SearchFilters{}, nil)
	if got != "kopi" {
		t.Errorf("leftoverText = %q, want %q", got, "kopi")
	}
}

func TestLeftoverTextRemovesCategoryName(t *testing.T) {
	categories := []generated.Category{{Name: "Transportasi"}}
	filters := SearchFilters{CategoryName: "Transportasi"}
	if got := leftoverText("transaksi transportasi", filters, categories); got != "" {
		t.Errorf("leftoverText = %q, want empty after category stripped", got)
	}
}

func TestCleanDescription(t *testing.T) {
	cases := []struct{ in, want string }{
		{"catat makan siang 25rb", "Makan siang"},
		{"tolong catat kopi 18.000", "Kopi"},
		{"beli bensin 50rb", "Beli bensin"},
		// "tadi" stays here: stripDateWords removes "tadi pagi" as a unit,
		// while dropping only "tadi" would leave "pagi" in the description.
		{"catat coba tes tadi pagi 7rb", "Tes tadi pagi"},
	}
	for _, tc := range cases {
		if got := cleanDescription(tc.in); got != tc.want {
			t.Errorf("cleanDescription(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// The fallback parser is what runs when no OPENAI_API_KEY is configured, so its
// intent mapping is product behaviour, not a test helper.
func TestHeuristicIntent(t *testing.T) {
	now := time.Date(2026, time.November, 8, 10, 0, 0, 0, time.UTC)

	cases := []struct {
		text     string
		kind     string
		typeName string
		amount   int64
	}{
		{"catat makan siang 25rb", "create_transaction", "expense", 25000},
		{"terima gaji 5jt", "create_transaction", "income", 5000000},
		{"berapa pengeluaran bulan ini", "spending_summary", "expense", 0},
		{"transaksi paling besar bulan ini", "largest_transactions", "expense", 0},
		{"cari transaksi kopi", "search", "expense", 0},
		{"cek budget bulan ini", "budget_status", "expense", 0},
		{"ubah budget jadi 4jt", "proposal", "expense", 4000000},
	}
	for _, tc := range cases {
		got := heuristicIntent(tc.text, now)
		if got.Kind != tc.kind {
			t.Errorf("heuristicIntent(%q).Kind = %q, want %q", tc.text, got.Kind, tc.kind)
		}
		if got.Type != tc.typeName {
			t.Errorf("heuristicIntent(%q).Type = %q, want %q", tc.text, got.Type, tc.typeName)
		}
		if got.Amount != tc.amount {
			t.Errorf("heuristicIntent(%q).Amount = %d, want %d", tc.text, got.Amount, tc.amount)
		}
	}
}

func TestStringMatchesQuery(t *testing.T) {
	if !stringMatchesQuery("transportasi", "cari transaksi transportasi") {
		t.Error("expected category name to match its own mention")
	}
	if stringMatchesQuery("kesehatan", "cari transaksi transportasi") {
		t.Error("unrelated category must not match")
	}
}
