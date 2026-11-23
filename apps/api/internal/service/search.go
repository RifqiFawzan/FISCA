package service

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/fisca-app/fisca/apps/api/internal/infrastructure/database/generated"
)

// SearchFilters is the structured meaning of a free-form query. The assistant
// and the UI both use it, so the mapping stays in one place (plan.md §7.2).
type SearchFilters struct {
	Period       string `json:"period,omitempty"`
	Type         string `json:"type,omitempty"`
	CategoryID   string `json:"categoryId,omitempty"`
	CategoryName string `json:"categoryName,omitempty"`
	MinAmount    *int64 `json:"minAmount,omitempty"`
	MaxAmount    *int64 `json:"maxAmount,omitempty"`
	Text         string `json:"text,omitempty"`
	Sort         string `json:"sort,omitempty"`
	Interpreted  string `json:"interpreted"`
}

// SearchResult is the parsed filter plus the matching page.
type SearchResult struct {
	Filters SearchFilters    `json:"filters"`
	Items   []TransactionDTO `json:"items"`
	Next    string           `json:"nextCursor,omitempty"`
}

// amountBody matches one or more Indonesian number+unit groups in a row, so a
// compound amount ("2 juta 500 ribu") is captured as a whole.
const amountBody = `(?:\d+(?:\.\d{3})*(?:,\d+)?\s*(?:juta|jt|ribu|rb|k)?\s*)+`

var (
	aboveAmountRe = regexp.MustCompile(`(?:di\s*atas|lebih\s*dari|diatas|minimal|min)\s*(?:rp\.?\s*)?` + amountBody)
	belowAmountRe = regexp.MustCompile(`(?:di\s*bawah|kurang\s*dari|dibawah|maksimal|max)\s*(?:rp\.?\s*)?` + amountBody)

	// amountTokenRe matches a single number with an optional unit. Thousands
	// are dot-separated ("1.200.000"), decimals comma-separated ("1,2").
	amountTokenRe = regexp.MustCompile(`(\d+(?:\.\d{3})*(?:,\d+)?)\s*(juta|jt|ribu|rb|k)?`)
)

var searchStopwords = map[string]struct{}{
	"transaksi": {}, "pengeluaran": {}, "pemasukan": {}, "yang": {}, "dan": {},
	"di": {}, "ke": {}, "untuk": {}, "dari": {}, "saya": {}, "aku": {}, "ku": {},
	"buat": {}, "beli": {}, "bayar": {}, "dengan": {}, "ada": {}, "apa": {},
	"aja": {}, "saja": {}, "semua": {}, "total": {}, "berapa": {}, "kategori": {},
	"bulan": {}, "minggu": {}, "tahun": {}, "hari": {}, "ini": {}, "lalu": {},
	"terakhir": {}, "paling": {}, "besar": {}, "kecil": {}, "lebih": {},
	"kurang": {}, "atas": {}, "bawah": {}, "rp": {}, "uang": {}, "keluar": {},
	"masuk": {}, "dapat": {}, "cari": {}, "tampilkan": {}, "lihat": {}, "daftar": {},
}

var monthWords = map[string]string{
	"januari": "01", "februari": "02", "maret": "03", "april": "04",
	"mei": "05", "juni": "06", "juli": "07", "agustus": "08",
	"september": "09", "oktober": "10", "november": "11", "desember": "12",
}

// ParseSearch turns an Indonesian free-form query into structured filters.
// It is deliberately deterministic; AI is never trusted to invent amounts.
func (s *Service) ParseSearch(ctx context.Context, q *generated.Queries, userID uuid.UUID, raw string) (SearchFilters, error) {
	text := strings.ToLower(strings.TrimSpace(raw))
	if text == "" {
		return SearchFilters{Interpreted: "Semua transaksi"}, nil
	}
	filters := SearchFilters{}
	notes := []string{}

	// --- period -------------------------------------------------------------
	if periodKey, label := detectPeriodPhrase(text, s.Now().Year()); periodKey != "" {
		filters.Period = periodKey
		notes = append(notes, label)
	}

	// --- amounts ------------------------------------------------------------
	if phrase := aboveAmountRe.FindString(text); phrase != "" {
		if amount, ok := parseAmountFromText(phrase); ok {
			filters.MinAmount = &amount
			notes = append(notes, "≥ Rp"+formatRupiahPlain(amount))
		}
	}
	if phrase := belowAmountRe.FindString(text); phrase != "" {
		if amount, ok := parseAmountFromText(phrase); ok {
			filters.MaxAmount = &amount
			notes = append(notes, "≤ Rp"+formatRupiahPlain(amount))
		}
	}
	if strings.Contains(text, "termahal") || strings.Contains(text, "terbesar") || strings.Contains(text, "paling besar") {
		filters.Sort = "amount_desc"
		notes = append(notes, "urut nominal terbesar")
	}

	// --- type ---------------------------------------------------------------
	switch {
	case strings.Contains(text, "pemasukan"), strings.Contains(text, "pendapatan"),
		strings.Contains(text, "income"), strings.Contains(text, "gaji"):
		filters.Type = "income"
		notes = append(notes, "pemasukan")
	case strings.Contains(text, "pengeluaran"), strings.Contains(text, "belanja"),
		strings.Contains(text, "jajan"), strings.Contains(text, "expense"):
		filters.Type = "expense"
		notes = append(notes, "pengeluaran")
	case strings.Contains(text, "transfer"):
		filters.Type = "transfer"
		notes = append(notes, "transfer")
	}

	// --- category -----------------------------------------------------------
	categories, err := q.ListCategoriesByUser(ctx, generated.ListCategoriesByUserParams{
		UserID: pgUUIDValue(userID),
	})
	if err != nil {
		return filters, mapDBError(err)
	}
	for _, category := range categories {
		name := strings.ToLower(category.Name)
		if stringMatchesQuery(name, text) {
			filters.CategoryID = uuidString(category.ID)
			filters.CategoryName = category.Name
			notes = append(notes, "kategori "+category.Name)
			break
		}
	}

	// --- leftover keyword ---------------------------------------------------
	filters.Text = leftoverText(text, filters, categories)
	if len(notes) == 0 {
		filters.Interpreted = "Pencarian transaksi"
	} else {
		filters.Interpreted = "Filter: " + strings.Join(notes, ", ")
	}
	return filters, nil
}

// SearchTransactions parses then executes a natural-language search.
func (s *Service) SearchTransactions(ctx context.Context, q *generated.Queries, userID uuid.UUID, raw string, limit int) (SearchResult, error) {
	filters, err := s.ParseSearch(ctx, q, userID, raw)
	if err != nil {
		return SearchResult{}, err
	}
	if filters.Sort == "amount_desc" {
		largest, err := s.LargestTransactions(ctx, q, userID, filters.Period, filters.Type, filters.CategoryID, limit)
		if err != nil {
			return SearchResult{}, err
		}
		return SearchResult{Filters: filters, Items: largest.Items}, nil
	}

	page, err := s.ListTransactions(ctx, q, userID, ListTransactionsInput{
		Type:       filters.Type,
		CategoryID: filters.CategoryID,
		Q:          filters.Text,
		Period:     filters.Period,
		Limit:      limit,
		MinAmount:  filters.MinAmount,
		MaxAmount:  filters.MaxAmount,
	})
	if err != nil {
		return SearchResult{}, err
	}

	// A pure category-name query ("gaji") used to filter on category_id
	// alone, so an uncategorized row whose description mentions the word
	// never showed up. Union a text search on the same keyword: category
	// members plus any row that mentions it, deduplicated newest-first.
	// Both queries share Type/Period/amounts and the same cursor, so the
	// merged page keeps paging coherent.
	if filters.CategoryID != "" && filters.Text == "" && limit > 0 {
		hadMore := page.NextCursor != ""
		alt, err := s.ListTransactions(ctx, q, userID, ListTransactionsInput{
			Type:      filters.Type,
			Q:         filters.CategoryName,
			Period:    filters.Period,
			Limit:     limit,
			MinAmount: filters.MinAmount,
			MaxAmount: filters.MaxAmount,
		})
		if err != nil {
			return SearchResult{}, err
		}
		if alt.NextCursor != "" {
			hadMore = true
		}
		page.Items = mergeTransactionsNewestFirst(page.Items, alt.Items, limit)
		page.NextCursor = ""
		if hadMore && len(page.Items) > 0 {
			last := page.Items[len(page.Items)-1]
			if ts, err := time.Parse(time.RFC3339, last.OccurredAt); err == nil {
				page.NextCursor = encodeCursor(ts, last.ID)
			}
		}
	}
	return SearchResult{Filters: filters, Items: page.Items, Next: page.NextCursor}, nil
}

// mergeTransactionsNewestFirst unions two newest-first transaction lists
// (same ORDER BY as ListTransactions), dropping duplicate ids and keeping
// at most limit rows so cursor paging stays coherent.
func mergeTransactionsNewestFirst(a, b []TransactionDTO, limit int) []TransactionDTO {
	seen := make(map[string]struct{}, len(a)+len(b))
	merged := make([]TransactionDTO, 0, len(a)+len(b))
	for _, list := range [][]TransactionDTO{a, b} {
		for _, tx := range list {
			if _, ok := seen[tx.ID]; ok {
				continue
			}
			seen[tx.ID] = struct{}{}
			merged = append(merged, tx)
		}
	}
	sort.Slice(merged, func(i, j int) bool {
		ti, _ := time.Parse(time.RFC3339, merged[i].OccurredAt)
		tj, _ := time.Parse(time.RFC3339, merged[j].OccurredAt)
		if !ti.Equal(tj) {
			return ti.After(tj)
		}
		return merged[i].ID > merged[j].ID
	})
	if len(merged) > limit {
		merged = merged[:limit]
	}
	return merged
}

func stringMatchesQuery(name, text string) bool {
	normalized := strings.NewReplacer("&", " ", "  ", " ").Replace(name)
	for _, token := range strings.Fields(normalized) {
		if len([]rune(token)) < 4 {
			continue
		}
		if strings.Contains(text, token) {
			return true
		}
	}
	return strings.Contains(text, strings.ToLower(name))
}

// detectPeriodPhrase maps Indonesian time words to the API period vocabulary.
func detectPeriodPhrase(text string, year int) (periodKey, label string) {
	lower := strings.ToLower(text)
	phrases := []struct {
		needle string
		period string
		label  string
	}{
		{"hari ini", "today", "hari ini"},
		{"bulan lalu", "last_month", "bulan lalu"},
		{"bulan kemarin", "last_month", "bulan lalu"},
		{"bulan ini", "this_month", "bulan ini"},
		{"minggu ini", "7d", "7 hari terakhir"},
		{"minggu lalu", "7d", "7 hari terakhir"},
		{"7 hari", "7d", "7 hari terakhir"},
		{"30 hari", "30d", "30 hari terakhir"},
		{"90 hari", "90d", "90 hari terakhir"},
		{"tahun ini", "this_year", "tahun ini"},
		{"tahun lalu", "last_year", "tahun lalu"},
	}
	for _, phrase := range phrases {
		if strings.Contains(lower, phrase.needle) {
			return phrase.period, phrase.label
		}
	}
	// Rolling windows: "3 bulan terakhir", "selama tiga bulan terakhir".
	if months, windowLabel := parseRollingMonths(lower); months > 0 {
		return strconv.Itoa(months) + "m", windowLabel
	}
	for word, month := range monthWords {
		if regexp.MustCompile(`\b` + word + `\b`).MatchString(lower) {
			return strconv.Itoa(year) + "-" + month, word
		}
	}
	return "", ""
}

// rollingMonthsRe captures the count in "N bulan terakhir".
var rollingMonthsRe = regexp.MustCompile(
	`(dua\s+belas|sebulan|\d+|satu|dua|tiga|empat|lima|enam|tujuh|delapan|sembilan|sepuluh|sebelas)\s*bulan\s+terakhir`)

// numberWords maps Indonesian number words used in period phrases.
var numberWords = map[string]int{
	"sebulan": 1, "satu": 1, "dua": 2, "tiga": 3, "empat": 4, "lima": 5,
	"enam": 6, "tujuh": 7, "delapan": 8, "sembilan": 9, "sepuluh": 10,
	"sebelas": 11, "dua belas": 12,
}

// parseRollingMonths returns the month count for "… bulan terakhir" phrases,
// or 0 when the text names none. The caller maps it to the "Nm" period key.
func parseRollingMonths(lower string) (int, string) {
	match := rollingMonthsRe.FindStringSubmatch(lower)
	if match == nil {
		return 0, ""
	}
	raw := strings.Join(strings.Fields(match[1]), " ")
	months := numberWords[raw]
	if months == 0 {
		if parsed, err := strconv.Atoi(raw); err == nil {
			months = parsed
		}
	}
	if months < 1 || months > 60 {
		return 0, ""
	}
	return months, fmt.Sprintf("%d bulan terakhir", months)
}

func leftoverText(text string, filters SearchFilters, categories []generated.Category) string {
	text = aboveAmountRe.ReplaceAllString(text, " ")
	text = belowAmountRe.ReplaceAllString(text, " ")
	for _, phrase := range []string{
		"bulan lalu", "bulan kemarin", "bulan ini", "minggu ini", "minggu lalu",
		"hari ini",
		"7 hari", "30 hari", "90 hari", "tahun ini", "tahun lalu",
		"paling besar", "termahal", "terbesar",
	} {
		text = strings.ReplaceAll(text, phrase, " ")
	}
	words := strings.FieldsFunc(text, func(r rune) bool {
		return r == ' ' || r == ',' || r == '.' || r == '?' || r == '!'
	})
	kept := make([]string, 0, len(words))
	for _, word := range words {
		if _, stop := searchStopwords[word]; stop {
			continue
		}
		if len([]rune(word)) < 3 {
			continue
		}
		kept = append(kept, word)
	}
	joined := strings.Join(kept, " ")
	// A category name already narrowed the search; strip it from the keyword
	// so we do not accidentally require the literal text too.
	if filters.CategoryID != "" || filters.CategoryName != "" {
		for _, category := range categories {
			joined = strings.ReplaceAll(joined, strings.ToLower(category.Name), "")
		}
	}
	joined = strings.TrimSpace(joined)
	if len([]rune(joined)) > 100 {
		joined = string([]rune(joined)[:100])
	}
	return joined
}

func parseRupiahNumber(raw, suffix string) (int64, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, false
	}
	multiplier := int64(1)
	switch strings.ToLower(suffix) {
	case "rb", "ribu", "k":
		multiplier = 1_000
	case "jt", "juta":
		multiplier = 1_000_000
	}

	// Indonesian notation: "." separates thousands, "," separates decimals.
	// For a decimal suffix amount ("1,5jt") keep one decimal place.
	cleaned := strings.ReplaceAll(raw, ".", "")
	cleaned = strings.ReplaceAll(cleaned, ",", ".")
	if strings.Contains(cleaned, ".") {
		value, err := strconv.ParseFloat(cleaned, 64)
		if err != nil {
			return 0, false
		}
		amount := int64(value * float64(multiplier))
		if amount <= 0 {
			return 0, false
		}
		return amount, true
	}
	value, err := strconv.ParseInt(cleaned, 10, 64)
	if err != nil || value <= 0 {
		return 0, false
	}
	return value * multiplier, true
}

// parseAmountFromText reads every monetary token in a sentence and sums the
// ones that carry a unit, so "2 juta 500 ribu" becomes 2500000 (plan.md §12.4).
// Without any unit the last plain number wins. Date numbers are removed first:
// "bayar tgl 5 200rb" must be 200000, not 5 (ISSUE-17).
func parseAmountFromText(text string) (int64, bool) {
	cleaned := dayOfMonthRe.ReplaceAllString(strings.ToLower(text), " ")
	matches := amountTokenRe.FindAllStringSubmatch(cleaned, -1)

	var withUnit, plain int64
	var hasUnit, hasPlain bool
	for _, match := range matches {
		value, ok := parseRupiahNumber(match[1], match[2])
		if !ok {
			continue
		}
		if match[2] != "" {
			withUnit += value
			hasUnit = true
			continue
		}
		plain = value
		hasPlain = true
	}
	switch {
	case hasUnit:
		return withUnit, true
	case hasPlain:
		return plain, true
	}
	return 0, false
}
