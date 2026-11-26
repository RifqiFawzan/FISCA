package period

import (
	"testing"
	"time"
)

func jakarta(t *testing.T) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		t.Fatalf("load location: %v", err)
	}
	return loc
}

// now is 2026-11-08 10:00 WIB. Ranges must be computed in the user's zone, not
// in UTC, otherwise the first seven hours of every month land in the wrong one.
func fixedNow(t *testing.T) time.Time {
	t.Helper()
	return time.Date(2026, time.November, 8, 10, 0, 0, 0, jakarta(t))
}

func TestParseThisMonth(t *testing.T) {
	loc := jakarta(t)
	got, err := Parse("this_month", loc, fixedNow(t))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	wantStart := time.Date(2026, time.November, 1, 0, 0, 0, 0, loc)
	wantEnd := time.Date(2026, time.December, 1, 0, 0, 0, 0, loc)
	if !got.Start.Equal(wantStart) || !got.End.Equal(wantEnd) {
		t.Fatalf("range = [%s, %s), want [%s, %s)", got.Start, got.End, wantStart, wantEnd)
	}
	if got.Key != "2026-11" {
		t.Errorf("key = %q, want 2026-11", got.Key)
	}
	if got.Label != "November 2026" {
		t.Errorf("label = %q, want November 2026", got.Label)
	}
}

func TestParseLastMonth(t *testing.T) {
	loc := jakarta(t)
	got, err := Parse("last_month", loc, fixedNow(t))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	wantStart := time.Date(2026, time.October, 1, 0, 0, 0, 0, loc)
	wantEnd := time.Date(2026, time.November, 1, 0, 0, 0, 0, loc)
	if !got.Start.Equal(wantStart) || !got.End.Equal(wantEnd) {
		t.Fatalf("range = [%s, %s), want [%s, %s)", got.Start, got.End, wantStart, wantEnd)
	}
}

func TestParseRollingWindows(t *testing.T) {
	loc := jakarta(t)
	now := fixedNow(t)

	cases := []struct {
		raw      string
		wantDays int
	}{
		{"7d", 7},
		{"30d", 30},
		{"90d", 90},
		{"2d", 2},
		{"today", 1},
		{"hari_ini", 1},
	}
	for _, tc := range cases {
		t.Run(tc.raw, func(t *testing.T) {
			got, err := Parse(tc.raw, loc, now)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			days := int(got.End.Sub(got.Start).Hours() / 24)
			if days != tc.wantDays {
				t.Errorf("window = %d days, want %d", days, tc.wantDays)
			}
			if !got.Start.Before(now) || !got.End.After(now) {
				t.Errorf("window [%s, %s) must contain now %s", got.Start, got.End, now)
			}
		})
	}
}

func TestParseExplicitMonthAndDay(t *testing.T) {
	loc := jakarta(t)

	month, err := Parse("2026-02", loc, fixedNow(t))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if month.Key != "2026-02" {
		t.Errorf("key = %q, want 2026-02", month.Key)
	}
	if days := int(month.End.Sub(month.Start).Hours() / 24); days != 28 {
		t.Errorf("February 2026 window = %d days, want 28", days)
	}

	day, err := Parse("2026-11-08", loc, fixedNow(t))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if days := int(day.End.Sub(day.Start).Hours() / 24); days != 1 {
		t.Errorf("single-day window = %d days, want 1", days)
	}
}

func TestParseRejectsGarbage(t *testing.T) {
	loc := jakarta(t)
	for _, raw := range []string{"kemarin", "2026-13", "2026-11-31", "abc"} {
		if _, err := Parse(raw, loc, fixedNow(t)); err == nil {
			t.Errorf("Parse(%q) = nil error, want rejection", raw)
		}
	}
}

// End is exclusive everywhere: a transaction exactly at midnight on the first
// of next month belongs to the next period, never to this one.
func TestRangeIsHalfOpen(t *testing.T) {
	loc := jakarta(t)
	got, err := Parse("this_month", loc, fixedNow(t))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	atEnd := got.End
	if !atEnd.Equal(time.Date(2026, time.December, 1, 0, 0, 0, 0, loc)) {
		t.Fatalf("end = %s, want 2026-12-01T00:00:00+07:00", atEnd)
	}
}

// "3 bulan terakhir" and friends resolve to a rolling window that includes
// today (ISSUE-14).
func TestParseRollingMonths(t *testing.T) {
	loc := jakarta(t)
	now := fixedNow(t) // 8 Nov 2026, 10:00 WIB

	got, err := Parse("3m", loc, now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	wantStart := time.Date(2026, time.August, 8, 0, 0, 0, 0, loc)
	wantEnd := time.Date(2026, time.November, 9, 0, 0, 0, 0, loc)
	if !got.Start.In(loc).Equal(wantStart) {
		t.Errorf("start = %s, want %s", got.Start.In(loc), wantStart)
	}
	if !got.End.In(loc).Equal(wantEnd) {
		t.Errorf("end = %s, want %s", got.End.In(loc), wantEnd)
	}
	if got.Label != "3 bulan terakhir" {
		t.Errorf("label = %q, want %q", got.Label, "3 bulan terakhir")
	}
	if got.Bucket != "month" {
		t.Errorf("bucket = %q, want month", got.Bucket)
	}
	// The window must contain now: it is half-open [start, end).
	if now.Before(got.Start) || !now.Before(got.End) {
		t.Errorf("now %s outside [%s, %s)", now, got.Start, got.End)
	}

	// Short windows bucket by day, long ones by month.
	day, err := Parse("1m", loc, now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if day.Bucket != "day" {
		t.Errorf("1m bucket = %q, want day", day.Bucket)
	}

	for _, raw := range []string{"0m", "99m", "m", "abc"} {
		if _, err := Parse(raw, loc, now); err == nil {
			t.Errorf("Parse(%q) = nil error, want rejection", raw)
		}
	}
}
