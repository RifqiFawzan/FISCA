// Package period resolves reporting periods in the user's timezone.
//
// Every range returned is half-open: [Start, End). Start/End are UTC instants;
// StartDate/EndDate are the calendar dates in the user's location, which is what
// budgets store (plan.md §5.3).
package period

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Range is a resolved reporting period.
type Range struct {
	Start     time.Time
	End       time.Time
	StartDate time.Time // midnight in Loc
	EndDate   time.Time // midnight in Loc, exclusive
	Key       string    // e.g. "2026-11"
	Label     string    // e.g. "November 2026"
	Bucket    string    // day | week | month
}

// Location loads a timezone, falling back to Asia/Jakarta when invalid.
func Location(name string) *time.Location {
	if name == "" {
		name = "Asia/Jakarta"
	}
	loc, err := time.LoadLocation(name)
	if err != nil {
		return time.FixedZone("WIB", 7*60*60)
	}
	return loc
}

// CurrentMonth returns the month containing now in loc.
func CurrentMonth(loc *time.Location, now time.Time) Range {
	local := now.In(loc)
	start := time.Date(local.Year(), local.Month(), 1, 0, 0, 0, 0, loc)
	return monthRange(start, loc)
}

// Parse resolves a period query. Supported values:
//
//	"" | "this_month" | "bulan_ini" | "current_month"
//	"last_month" | "bulan_lalu"
//	"7d" | "30d" | "90d" | "1y"
//	"Nm" (rolling N months, e.g. "3m" for "3 bulan terakhir")
//	"YYYY-MM"
//	"YYYY-MM-DD" (single day)
func Parse(raw string, loc *time.Location, now time.Time) (Range, error) {
	raw = strings.ToLower(strings.TrimSpace(raw))
	local := now.In(loc)

	switch raw {
	case "", "this_month", "bulan_ini", "current_month":
		return CurrentMonth(loc, now), nil
	case "last_month", "bulan_lalu":
		return Shift(CurrentMonth(loc, now), -1, loc), nil
	case "this_year", "tahun_ini":
		start := time.Date(local.Year(), time.January, 1, 0, 0, 0, 0, loc)
		end := start.AddDate(1, 0, 0)
		return fullRange(start, end, loc, "Tahun "+fmt.Sprint(local.Year()), "tahun"), nil
	case "last_year", "tahun_lalu":
		start := time.Date(local.Year()-1, time.January, 1, 0, 0, 0, 0, loc)
		end := start.AddDate(1, 0, 0)
		return fullRange(start, end, loc, "Tahun "+fmt.Sprint(local.Year()-1), "tahun"), nil
	case "7d":
		return dayRange(local.AddDate(0, 0, -6), local, loc), nil
	case "30d":
		return dayRange(local.AddDate(0, 0, -29), local, loc), nil
	case "90d":
		return dayRange(local.AddDate(0, 0, -89), local, loc), nil
	case "1y":
		start := local.AddDate(-1, 0, 0)
		return fullRange(start, local, loc, "12 bulan terakhir", "month"), nil
	}

	// Rolling months: "3m" = the last three months including today. The window
	// ends tomorrow so today stays inside the range (half-open [start, end)).
	if strings.HasSuffix(raw, "m") {
		months, err := strconv.Atoi(strings.TrimSuffix(raw, "m"))
		if err == nil && months >= 1 && months <= 60 {
			today := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, loc)
			start := today.AddDate(0, -months, 0)
			end := today.AddDate(0, 0, 1)
			bucket := "day"
			if months >= 3 {
				bucket = "month"
			}
			return fullRange(start, end, loc, fmt.Sprintf("%d bulan terakhir", months), bucket), nil
		}
	}

	if len(raw) == 7 {
		t, err := time.ParseInLocation("2006-01", raw, loc)
		if err != nil {
			return Range{}, fmt.Errorf("period tidak valid: %q", raw)
		}
		return monthRange(t, loc), nil
	}
	if len(raw) == 10 {
		t, err := time.ParseInLocation("2006-01-02", raw, loc)
		if err != nil {
			return Range{}, fmt.Errorf("period tidak valid: %q", raw)
		}
		end := t.AddDate(0, 0, 1)
		return fullRange(t, end, loc,
			fmt.Sprintf("%d %s %d", t.Day(), monthName(t.Month()), t.Year()), "day"), nil
	}
	return Range{}, fmt.Errorf("period tidak valid: %q", raw)
}

// Shift moves a monthly range by n months.
func Shift(r Range, n int, loc *time.Location) Range {
	start := r.StartDate.In(loc).AddDate(0, n, 0)
	start = time.Date(start.Year(), start.Month(), 1, 0, 0, 0, 0, loc)
	return monthRange(start, loc)
}

// MonthOf returns the monthly range containing t.
func MonthOf(t time.Time, loc *time.Location) Range {
	local := t.In(loc)
	start := time.Date(local.Year(), local.Month(), 1, 0, 0, 0, 0, loc)
	return monthRange(start, loc)
}

func monthRange(start time.Time, loc *time.Location) Range {
	end := time.Date(start.Year(), start.Month()+1, 1, 0, 0, 0, 0, loc)
	return fullRange(start, end, loc,
		fmt.Sprintf("%s %d", monthName(start.Month()), start.Year()), "month")
}

func dayRange(start, endLocal time.Time, loc *time.Location) Range {
	start = time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, loc)
	end := time.Date(endLocal.Year(), endLocal.Month(), endLocal.Day(), 0, 0, 0, 0, loc).AddDate(0, 0, 1)
	return fullRange(start, end, loc, "Periode terpilih", "day")
}

func fullRange(start, end time.Time, loc *time.Location, label, bucket string) Range {
	return Range{
		Start:     start.UTC(),
		End:       end.UTC(),
		StartDate: start.In(loc),
		EndDate:   end.In(loc),
		Key:       start.In(loc).Format("2006-01"),
		Label:     label,
		Bucket:    bucket,
	}
}

// Date converts a range boundary to the pgtype.Date value used by budgets.
func Date(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

func monthName(m time.Month) string {
	names := [...]string{
		"Januari", "Februari", "Maret", "April", "Mei", "Juni",
		"Juli", "Agustus", "September", "Oktober", "November", "Desember",
	}
	if m < time.January || m > time.December {
		return ""
	}
	return names[m-1]
}
