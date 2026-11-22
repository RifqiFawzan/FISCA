package service

import (
	"testing"
	"time"
)

func TestParseRelativeDate(t *testing.T) {
	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		t.Fatalf("load location: %v", err)
	}
	// 8 Nov 2026, 10:00 WIB.
	now := time.Date(2026, time.November, 8, 10, 0, 0, 0, loc)

	cases := []struct {
		name string
		text string
		want string // YYYY-MM-DD in loc, empty means nil
	}{
		{"kemarin", "catat makan siang kemarin 25rb", "2026-11-07"},
		{"kemarin lusa", "beli kopi kemarin lusa 18rb", "2026-11-06"},
		{"hari ini", "catat kopi hari ini 18rb", "2026-11-08"},
		{"tgl 15 earlier this month", "bayar listrik tgl 5 200rb", "2026-11-05"},
		{"tgl in the future rolls back a month", "bayar kos tgl 20 1jt", "2026-10-20"},
		{"minggu lalu", "catat makan minggu lalu 25rb", "2026-11-01"},
		{"tadi pagi", "catat kopi tadi pagi 18rb", "2026-11-08"},
		{"bulan lalu", "catat makan bulan lalu 25rb", "2026-10-08"},
		{"tgl with month name", "bayar listrik tgl 15 oktober 200rb", "2026-10-15"},
		{"month name in the future rolls back a year", "bayar tgl 5 desember 100rb", "2025-12-05"},
		{"no date", "catat makan siang 25rb", ""},
		{"invalid day", "bayar tgl 45 10rb", ""},
		{"day overflows the month", "bayar tgl 31 10rb", ""},
		{"day missing in named month", "bayar tgl 31 februari 10rb", ""},
		{"unknown month name", "bayar tgl 15 romawi 10rb", ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := parseRelativeDate(tc.text, now, loc)
			if tc.want == "" {
				if got != nil {
					t.Fatalf("got %s, want nil", got.In(loc).Format("2006-01-02"))
				}
				return
			}
			if got == nil {
				t.Fatalf("got nil, want %s", tc.want)
			}
			if date := got.In(loc).Format("2006-01-02"); date != tc.want {
				t.Fatalf("got %s, want %s", date, tc.want)
			}
			if h := got.In(loc).Hour(); h != 12 {
				t.Fatalf("hour = %d, want 12 (noon avoids timezone day shifts)", h)
			}
		})
	}
}

// Day words must not survive into the stored description.
func TestCleanDescriptionDropsDateWords(t *testing.T) {
	cases := map[string]string{
		"catat makan siang kemarin 25rb":     "Makan siang",
		"beli kopi kemarin lusa 18rb":        "Beli kopi",
		"bayar listrik tgl 5 200rb":          "Bayar listrik",
		"catat makan minggu lalu 25rb":       "Makan",
		"catat kopi tadi pagi 18rb":          "Kopi",
		"catat makan bulan lalu 25rb":        "Makan",
		"bayar listrik tgl 15 oktober 200rb": "Bayar listrik",
	}
	for in, want := range cases {
		got := cleanDescription(stripDateWords(in))
		if got != want {
			t.Errorf("cleanDescription(stripDateWords(%q)) = %q, want %q", in, got, want)
		}
	}
}

// Regression: the heuristic fills Description with the raw sentence, so the
// day word must be removed from an already-filled description too.
func TestHeuristicDescriptionIsCleanedOfDayWord(t *testing.T) {
	now := time.Date(2026, time.November, 8, 10, 0, 0, 0, time.UTC)
	intent := heuristicIntent("catat makan siang kemarin 25rb", now)
	got := cleanDescription(stripDateWords(intent.Description))
	if got != "Makan siang" {
		t.Fatalf("description = %q, want %q", got, "Makan siang")
	}
}

// The intent description can arrive half-eaten by the amount stripper
// ("tgl 15 oktober" → "tgl oktober"); the raw sentence must win then, while a
// clean model-written description is kept as-is (ISSUE-14).
func TestDescriptionForDatedText(t *testing.T) {
	cases := []struct {
		raw      string
		fallback string
		want     string
	}{
		{"bayar listrik tgl 15 oktober 200rb", "Bayar listrik tgl oktober", "Bayar listrik"},
		{"catat kopi tadi pagi 18rb", "Kopi tadi pagi", "Kopi"},
		{"catat makan siang kemarin 25rb", "Makan siang kemarin", "Makan siang"},
		{"beli sesuatu kemarin 25rb", "Belanja kebutuhan", "Belanja kebutuhan"},
		{"catat coba tes tadi pagi 7rb", "Tes tadi pagi", "Tes"},
		{"catat makan minggu lalu 25rb", "", "Makan"},
	}
	for _, tc := range cases {
		if got := descriptionForDatedText(tc.raw, tc.fallback); got != tc.want {
			t.Errorf("descriptionForDatedText(%q, %q) = %q, want %q",
				tc.raw, tc.fallback, got, tc.want)
		}
	}
}
