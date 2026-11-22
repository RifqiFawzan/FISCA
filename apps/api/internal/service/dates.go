package service

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

// dayOfMonthRe matches "tgl 15" and "tgl 15 Oktober". The month name is
// optional, so a date can point outside the current month (ISSUE-14).
var dayOfMonthRe = regexp.MustCompile(`\b(?:tgl|tanggal)\s*(\d{1,2})(?:\s+([a-z]{3,}))?\b`)

// tadiRe matches "tadi", "tadi pagi", "tadi malam": all earlier today.
var tadiRe = regexp.MustCompile(`\btadi(?:\s+(?:pagi|siang|sore|malam))?`)

// monthByName maps full and abbreviated Indonesian month names.
var monthByName = map[string]time.Month{
	"januari": time.January, "jan": time.January,
	"februari": time.February, "feb": time.February,
	"maret": time.March, "mar": time.March,
	"april": time.April, "apr": time.April,
	"mei":  time.May,
	"juni": time.June, "jun": time.June,
	"juli": time.July, "jul": time.July,
	"agustus": time.August, "agu": time.August, "agt": time.August,
	"september": time.September, "sep": time.September, "sept": time.September,
	"oktober": time.October, "okt": time.October,
	"november": time.November, "nov": time.November,
	"desember": time.December, "des": time.December,
}

// parseRelativeDate resolves Indonesian day references to noon of that day in
// loc. It returns nil when the text names no date, so callers keep the server
// default (now).
//
// Supported: "kemarin lusa" (-2), "kemarin" (-1), "hari ini"/"tadi …" (today),
// "bulan lalu" (same day last month), "minggu lalu" (-7), "tgl 15",
// "tgl 15 Oktober".
//
// Future dates roll back: a named month moves to the previous year, an
// unnamed day to the previous month, because a transaction cannot be recorded
// for a day that has not happened yet.
func parseRelativeDate(text string, now time.Time, loc *time.Location) *time.Time {
	lower := strings.ToLower(text)
	local := now.In(loc)
	day := time.Date(local.Year(), local.Month(), local.Day(), 12, 0, 0, 0, loc)

	var out time.Time
	switch {
	case strings.Contains(lower, "kemarin lusa"):
		out = day.AddDate(0, 0, -2)
	case strings.Contains(lower, "kemarin"):
		out = day.AddDate(0, 0, -1)
	case strings.Contains(lower, "hari ini"), tadiRe.MatchString(lower):
		out = day
	case strings.Contains(lower, "bulan lalu"):
		// Same day of the previous month; AddDate clamps short months.
		out = day.AddDate(0, -1, 0)
	case strings.Contains(lower, "minggu lalu"):
		out = day.AddDate(0, 0, -7)
	default:
		parsed, ok := parseExplicitDay(lower, local, day, loc)
		if !ok {
			return nil
		}
		out = parsed
	}
	return &out
}

// parseExplicitDay reads "tgl N" and "tgl N Month".
func parseExplicitDay(lower string, local, day time.Time, loc *time.Location) (time.Time, bool) {
	match := dayOfMonthRe.FindStringSubmatch(lower)
	if match == nil {
		return time.Time{}, false
	}
	n, err := strconv.Atoi(match[1])
	if err != nil || n < 1 || n > 31 {
		return time.Time{}, false
	}

	year, month := local.Year(), local.Month()
	namedMonth := match[2] != ""
	if namedMonth {
		parsed, ok := monthByName[match[2]]
		if !ok {
			return time.Time{}, false
		}
		month = parsed
	}

	out := time.Date(year, month, n, 12, 0, 0, 0, loc)
	if out.Month() != month || out.Day() != n {
		// e.g. "tgl 31 februari": the day does not exist in that month.
		return time.Time{}, false
	}
	if out.After(day) {
		if namedMonth {
			out = out.AddDate(-1, 0, 0) // most recent past occurrence
		} else {
			out = out.AddDate(0, -1, 0) // same day, previous month
		}
	}
	return out, true
}

// stripDateWords removes day references from a free-text description so
// "makan siang kemarin" is stored as "Makan siang".
func stripDateWords(text string) string {
	lower := strings.ToLower(text)
	for _, phrase := range []string{
		"kemarin lusa", "minggu lalu", "bulan lalu", "kemarin", "hari ini",
	} {
		lower = strings.ReplaceAll(lower, phrase, " ")
	}
	lower = tadiRe.ReplaceAllString(lower, " ")
	return dayOfMonthRe.ReplaceAllString(lower, " ")
}

// descriptionForDatedText keeps the description the intent produced when it is
// still usable, otherwise rebuilds it from the raw sentence.
//
// The heuristic strips bare numbers before dates are parsed, which can leave a
// half-eaten date phrase ("tgl 15 oktober" → "tgl oktober") that
// stripDateWords no longer recognises. In that case the raw sentence is the
// only reliable source (ISSUE-14).
func descriptionForDatedText(raw, fallback string) string {
	cleaned := strings.TrimSpace(cleanDescription(stripDateWords(fallback)))
	if cleaned == "" || strings.Contains(strings.ToLower(cleaned), "tgl") {
		return strings.TrimSpace(cleanDescription(stripDateWords(raw)))
	}
	return cleaned
}
