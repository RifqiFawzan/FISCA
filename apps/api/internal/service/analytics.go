package service

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/fisca-app/fisca/apps/api/internal/infrastructure/database/generated"
	"github.com/fisca-app/fisca/apps/api/internal/period"
)

// PeriodDTO describes the resolved reporting window.
type PeriodDTO struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Start string `json:"start"`
	End   string `json:"end"`
}

// MonthlyTotalDTO is one point of the income/expense trend.
type MonthlyTotalDTO struct {
	Month   string `json:"month"`
	Income  int64  `json:"income"`
	Expense int64  `json:"expense"`
}

// DashboardDTO is the single round-trip payload for Home.
type DashboardDTO struct {
	Period             PeriodDTO          `json:"period"`
	Summary            SummaryDTO         `json:"summary"`
	Accounts           []AccountDTO       `json:"accounts"`
	RecentTransactions []TransactionDTO   `json:"recentTransactions"`
	TopCategories      []CategorySpendDTO `json:"topCategories"`
	MonthlyTrend       []MonthlyTotalDTO  `json:"monthlyTrend"`
	Budget             *BudgetDTO         `json:"budget"`
}

// FinancialSummaryDTO is the reports header payload.
type FinancialSummaryDTO struct {
	Period       PeriodDTO          `json:"period"`
	Summary      SummaryDTO         `json:"summary"`
	Previous     SummaryDTO         `json:"previous"`
	Delta        SummaryDTO         `json:"delta"`
	Categories   []CategorySpendDTO `json:"categories"`
	MonthlyTrend []MonthlyTotalDTO  `json:"monthlyTrend"`
}

// LargestTransactionsDTO lists the biggest expenses (or a chosen type).
type LargestTransactionsDTO struct {
	Period PeriodDTO        `json:"period"`
	Items  []TransactionDTO `json:"items"`
}

// UnusualFinding is one category flagged as unusual.
type UnusualFinding struct {
	CategoryID     *string `json:"categoryId"`
	CategoryName   string  `json:"categoryName"`
	CurrentAmount  int64   `json:"currentAmount"`
	BaselineMedian int64   `json:"baselineMedian"`
	Deviation      float64 `json:"deviation"`
	Message        string  `json:"message"`
}

// UnusualSpendingDTO always states whether the data is sufficient.
type UnusualSpendingDTO struct {
	Period         PeriodDTO        `json:"period"`
	SufficientData bool             `json:"sufficientData"`
	Method         string           `json:"method"`
	MonthsAnalyzed int              `json:"monthsAnalyzed"`
	Findings       []UnusualFinding `json:"findings"`
	Limitations    []string         `json:"limitations"`
}

// CategoryDeltaDTO is one row of the period comparison.
type CategoryDeltaDTO struct {
	CategoryID *string `json:"categoryId"`
	Name       string  `json:"name"`
	Current    int64   `json:"current"`
	Previous   int64   `json:"previous"`
	Delta      int64   `json:"delta"`
	Percent    float64 `json:"percent"`
}

// SpendingComparisonDTO compares the selected period with the one before it.
type SpendingComparisonDTO struct {
	Period     PeriodDTO          `json:"period"`
	Previous   PeriodDTO          `json:"previousPeriod"`
	Current    SummaryDTO         `json:"current"`
	Prior      SummaryDTO         `json:"previous"`
	Delta      SummaryDTO         `json:"delta"`
	Categories []CategoryDeltaDTO `json:"categories"`
}

// CashFlowPointDTO is one bucket of the cash-flow chart.
type CashFlowPointDTO struct {
	BucketStart string `json:"bucketStart"`
	Income      int64  `json:"income"`
	Expense     int64  `json:"expense"`
	Net         int64  `json:"net"`
}

// CashFlowDTO is the chart payload.
type CashFlowDTO struct {
	Period PeriodDTO          `json:"period"`
	Bucket string             `json:"bucket"`
	Points []CashFlowPointDTO `json:"points"`
}

// Dashboard builds everything the Home screen needs in one request.
func (s *Service) Dashboard(ctx context.Context, q *generated.Queries, userID uuid.UUID, periodRaw string) (DashboardDTO, error) {
	r, err := s.resolvePeriod(ctx, q, userID, periodRaw)
	if err != nil {
		return DashboardDTO{}, err
	}

	totals, err := q.GetPeriodTotals(ctx, generated.GetPeriodTotalsParams{
		UserID: pgUUIDValue(userID),
		FromTs: timeStampValue(r.Start),
		ToTs:   timeStampValue(r.End),
	})
	if err != nil {
		return DashboardDTO{}, mapDBError(err)
	}

	accounts, err := s.ListAccounts(ctx, q, userID)
	if err != nil {
		return DashboardDTO{}, err
	}

	recent, err := q.GetRecentTransactions(ctx, generated.GetRecentTransactionsParams{
		UserID: pgUUIDValue(userID),
		Limit:  5,
	})
	if err != nil {
		return DashboardDTO{}, mapDBError(err)
	}
	cache := newTxNameCache(ctx, q, pgUUIDValue(userID))
	recentDTO := make([]TransactionDTO, 0, len(recent))
	for _, row := range recent {
		recentDTO = append(recentDTO, transactionDTO(row, cache))
	}

	categories, err := s.categoryBreakdown(ctx, q, userID, r)
	if err != nil {
		return DashboardDTO{}, err
	}
	if len(categories) > 6 {
		categories = categories[:6]
	}

	trend, err := s.monthlyTrend(ctx, q, userID, period.Shift(r, -5, s.location(ctx, q, userID)), r)
	if err != nil {
		return DashboardDTO{}, err
	}

	budget, err := s.budgetForPeriod(ctx, q, userID, r)
	if err != nil {
		return DashboardDTO{}, err
	}

	return DashboardDTO{
		Period:             periodDTO(r),
		Summary:            SummaryDTO{Income: totals.Income, Expense: totals.Expense, Net: totals.Net},
		Accounts:           accounts,
		RecentTransactions: recentDTO,
		TopCategories:      categories,
		MonthlyTrend:       trend,
		Budget:             budget,
	}, nil
}

// FinancialSummary aggregates the selected period plus the previous one.
func (s *Service) FinancialSummary(ctx context.Context, q *generated.Queries, userID uuid.UUID, periodRaw string) (FinancialSummaryDTO, error) {
	r, err := s.resolvePeriod(ctx, q, userID, periodRaw)
	if err != nil {
		return FinancialSummaryDTO{}, err
	}
	prev := period.Shift(r, -1, s.location(ctx, q, userID))

	currentTotals, err := totalsFor(ctx, q, userID, r)
	if err != nil {
		return FinancialSummaryDTO{}, err
	}
	prevTotals, err := totalsFor(ctx, q, userID, prev)
	if err != nil {
		return FinancialSummaryDTO{}, err
	}
	categories, err := s.categoryBreakdown(ctx, q, userID, r)
	if err != nil {
		return FinancialSummaryDTO{}, err
	}
	trend, err := s.monthlyTrend(ctx, q, userID, period.Shift(r, -5, s.location(ctx, q, userID)), r)
	if err != nil {
		return FinancialSummaryDTO{}, err
	}

	return FinancialSummaryDTO{
		Period:   periodDTO(r),
		Summary:  currentTotals,
		Previous: prevTotals,
		Delta: SummaryDTO{
			Income:  currentTotals.Income - prevTotals.Income,
			Expense: currentTotals.Expense - prevTotals.Expense,
			Net:     currentTotals.Net - prevTotals.Net,
		},
		Categories:   categories,
		MonthlyTrend: trend,
	}, nil
}

// LargestTransactions returns the biggest transactions in the period.
func (s *Service) LargestTransactions(ctx context.Context, q *generated.Queries, userID uuid.UUID, periodRaw, txType, categoryID string, limit int) (LargestTransactionsDTO, error) {
	r, err := s.resolvePeriod(ctx, q, userID, periodRaw)
	if err != nil {
		return LargestTransactionsDTO{}, err
	}
	if limit <= 0 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}
	params := generated.GetLargestTransactionsParams{
		UserID:   pgUUIDValue(userID),
		FromTs:   timeStampValue(r.Start),
		ToTs:     timeStampValue(r.End),
		PageSize: int32(limit),
	}
	if t := strings.TrimSpace(txType); t != "" {
		if t != "income" && t != "expense" && t != "transfer" {
			return LargestTransactionsDTO{}, Invalid("type", "Tipe transaksi tidak valid")
		}
		params.Type = &t
	}
	if categoryID != "" {
		id, err := parseUUID(categoryID)
		if err != nil {
			return LargestTransactionsDTO{}, err
		}
		params.CategoryID = pgUUIDValue(id)
	}
	rows, err := q.GetLargestTransactions(ctx, params)
	if err != nil {
		return LargestTransactionsDTO{}, mapDBError(err)
	}
	cache := newTxNameCache(ctx, q, pgUUIDValue(userID))
	items := make([]TransactionDTO, 0, len(rows))
	for _, row := range rows {
		items = append(items, transactionDTO(row, cache))
	}
	return LargestTransactionsDTO{Period: periodDTO(r), Items: items}, nil
}

// UnusualSpending flags categories whose current spend is far above their
// median monthly spend. It reports insufficient data instead of guessing.
func (s *Service) UnusualSpending(ctx context.Context, q *generated.Queries, userID uuid.UUID, periodRaw, categoryID string) (UnusualSpendingDTO, error) {
	r, err := s.resolvePeriod(ctx, q, userID, periodRaw)
	if err != nil {
		return UnusualSpendingDTO{}, err
	}
	loc := s.location(ctx, q, userID)
	historyStart := period.Shift(r, -6, loc)

	params := generated.GetMonthlyCategoryTotalsParams{
		UserID: pgUUIDValue(userID),
		FromTs: timeStampValue(historyStart.Start),
		ToTs:   timeStampValue(r.End),
	}
	if categoryID != "" {
		id, err := parseUUID(categoryID)
		if err != nil {
			return UnusualSpendingDTO{}, err
		}
		params.CategoryID = pgUUIDValue(id)
	}
	rows, err := q.GetMonthlyCategoryTotals(ctx, params)
	if err != nil {
		return UnusualSpendingDTO{}, mapDBError(err)
	}

	currentKey := r.StartDate.Format("2006-01")
	type agg struct {
		name    string
		current int64
		prior   map[string]int64
	}
	byCategory := map[string]*agg{}
	months := map[string]struct{}{}
	for _, row := range rows {
		key := uuidString(row.CategoryID)
		monthKey := row.Month.Time.Format("2006-01")
		months[monthKey] = struct{}{}
		entry, ok := byCategory[key]
		if !ok {
			entry = &agg{name: row.CategoryName, prior: map[string]int64{}}
			byCategory[key] = entry
		}
		if monthKey == currentKey {
			entry.current = row.Total
		} else {
			entry.prior[monthKey] = row.Total
		}
	}

	findings := make([]UnusualFinding, 0)
	for key, entry := range byCategory {
		if entry.current <= 0 || len(entry.prior) < 3 {
			continue
		}
		baseline := make([]int64, 0, len(entry.prior))
		for _, total := range entry.prior {
			baseline = append(baseline, total)
		}
		median := medianInt64(baseline)
		if median <= 0 {
			continue
		}
		deviation := medianAbsoluteDeviation(baseline, median)
		scale := 1.4826 * deviation
		if scale == 0 {
			scale = float64(median) * 0.2
		}
		threshold := float64(median) + 3*scale
		if float64(entry.current) <= threshold {
			continue
		}
		var catPtr *string
		if key != "" {
			v := key
			catPtr = &v
		}
		findings = append(findings, UnusualFinding{
			CategoryID:     catPtr,
			CategoryName:   entry.name,
			CurrentAmount:  entry.current,
			BaselineMedian: median,
			Deviation:      float64(entry.current-median) / float64(median),
			Message:        "Pengeluaran " + entry.name + " jauh di atas kebiasaan bulananmu.",
		})
	}
	sort.Slice(findings, func(i, j int) bool { return findings[i].CurrentAmount > findings[j].CurrentAmount })
	if findings == nil {
		findings = []UnusualFinding{}
	}

	sufficient := len(months) >= 4
	limitations := []string{}
	if !sufficient {
		limitations = append(limitations, "Butuh minimal 4 bulan data pengeluaran untuk mendeteksi pola.")
	}

	return UnusualSpendingDTO{
		Period:         periodDTO(r),
		SufficientData: sufficient,
		Method:         "median_mad",
		MonthsAnalyzed: len(months),
		Findings:       findings,
		Limitations:    limitations,
	}, nil
}

// SpendingComparison compares the period with the previous period.
func (s *Service) SpendingComparison(ctx context.Context, q *generated.Queries, userID uuid.UUID, periodRaw string) (SpendingComparisonDTO, error) {
	r, err := s.resolvePeriod(ctx, q, userID, periodRaw)
	if err != nil {
		return SpendingComparisonDTO{}, err
	}
	loc := s.location(ctx, q, userID)
	prev := period.Shift(r, -1, loc)

	currentTotals, err := totalsFor(ctx, q, userID, r)
	if err != nil {
		return SpendingComparisonDTO{}, err
	}
	prevTotals, err := totalsFor(ctx, q, userID, prev)
	if err != nil {
		return SpendingComparisonDTO{}, err
	}
	currentCats, err := q.GetSpendingByCategory(ctx, generated.GetSpendingByCategoryParams{
		UserID: pgUUIDValue(userID), FromTs: timeStampValue(r.Start), ToTs: timeStampValue(r.End),
	})
	if err != nil {
		return SpendingComparisonDTO{}, mapDBError(err)
	}
	prevCats, err := q.GetSpendingByCategory(ctx, generated.GetSpendingByCategoryParams{
		UserID: pgUUIDValue(userID), FromTs: timeStampValue(prev.Start), ToTs: timeStampValue(prev.End),
	})
	if err != nil {
		return SpendingComparisonDTO{}, mapDBError(err)
	}

	type pair struct {
		name     string
		current  int64
		previous int64
	}
	merged := map[string]*pair{}
	for _, row := range prevCats {
		key := uuidString(row.CategoryID)
		merged[key] = &pair{name: row.CategoryName, previous: row.Total}
	}
	for _, row := range currentCats {
		key := uuidString(row.CategoryID)
		entry, ok := merged[key]
		if !ok {
			entry = &pair{name: row.CategoryName}
			merged[key] = entry
		}
		entry.current = row.Total
	}
	categories := make([]CategoryDeltaDTO, 0, len(merged))
	for key, entry := range merged {
		var catPtr *string
		if key != "" {
			v := key
			catPtr = &v
		}
		percent := 0.0
		if entry.previous > 0 {
			percent = float64(entry.current-entry.previous) / float64(entry.previous)
		}
		categories = append(categories, CategoryDeltaDTO{
			CategoryID: catPtr,
			Name:       entry.name,
			Current:    entry.current,
			Previous:   entry.previous,
			Delta:      entry.current - entry.previous,
			Percent:    percent,
		})
	}
	sort.Slice(categories, func(i, j int) bool { return categories[i].Current > categories[j].Current })

	return SpendingComparisonDTO{
		Period:   periodDTO(r),
		Previous: periodDTO(prev),
		Current:  currentTotals,
		Prior:    prevTotals,
		Delta: SummaryDTO{
			Income:  currentTotals.Income - prevTotals.Income,
			Expense: currentTotals.Expense - prevTotals.Expense,
			Net:     currentTotals.Net - prevTotals.Net,
		},
		Categories: categories,
	}, nil
}

// CashFlow returns income/expense buckets.
func (s *Service) CashFlow(ctx context.Context, q *generated.Queries, userID uuid.UUID, periodRaw, bucket string) (CashFlowDTO, error) {
	r, err := s.resolvePeriod(ctx, q, userID, periodRaw)
	if err != nil {
		return CashFlowDTO{}, err
	}
	bucket = strings.TrimSpace(bucket)
	if bucket == "" {
		bucket = "day"
	}
	switch bucket {
	case "day", "week", "month":
	default:
		return CashFlowDTO{}, Invalid("bucket", "Bucket harus day, week, atau month")
	}
	rows, err := q.GetCashFlowBuckets(ctx, generated.GetCashFlowBucketsParams{
		Bucket: bucket,
		UserID: pgUUIDValue(userID),
		FromTs: timeStampValue(r.Start),
		ToTs:   timeStampValue(r.End),
	})
	if err != nil {
		return CashFlowDTO{}, mapDBError(err)
	}
	points := make([]CashFlowPointDTO, 0, len(rows))
	for _, row := range rows {
		points = append(points, CashFlowPointDTO{
			BucketStart: timestamp(row.BucketStart).Format(time.RFC3339),
			Income:      row.Income,
			Expense:     row.Expense,
			Net:         row.Income - row.Expense,
		})
	}
	return CashFlowDTO{Period: periodDTO(r), Bucket: bucket, Points: points}, nil
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func (s *Service) resolvePeriod(ctx context.Context, q *generated.Queries, userID uuid.UUID, raw string) (period.Range, error) {
	tz, err := s.UserTimezone(ctx, q, userID)
	if err != nil {
		return period.Range{}, err
	}
	r, err := period.Parse(raw, period.Location(tz), s.Now())
	if err != nil {
		return period.Range{}, Invalid("period", err.Error())
	}
	return r, nil
}

func totalsFor(ctx context.Context, q *generated.Queries, userID uuid.UUID, r period.Range) (SummaryDTO, error) {
	row, err := q.GetPeriodTotals(ctx, generated.GetPeriodTotalsParams{
		UserID: pgUUIDValue(userID),
		FromTs: timeStampValue(r.Start),
		ToTs:   timeStampValue(r.End),
	})
	if err != nil {
		return SummaryDTO{}, mapDBError(err)
	}
	return SummaryDTO{Income: row.Income, Expense: row.Expense, Net: row.Net}, nil
}

func (s *Service) categoryBreakdown(ctx context.Context, q *generated.Queries, userID uuid.UUID, r period.Range) ([]CategorySpendDTO, error) {
	rows, err := q.GetSpendingByCategory(ctx, generated.GetSpendingByCategoryParams{
		UserID: pgUUIDValue(userID),
		FromTs: timeStampValue(r.Start),
		ToTs:   timeStampValue(r.End),
	})
	if err != nil {
		return nil, mapDBError(err)
	}
	var total int64
	for _, row := range rows {
		total += row.Total
	}
	out := make([]CategorySpendDTO, 0, len(rows))
	for _, row := range rows {
		percent := 0.0
		if total > 0 {
			percent = float64(row.Total) / float64(total)
		}
		out = append(out, CategorySpendDTO{
			CategoryID:       uuidPtr(row.CategoryID),
			Name:             row.CategoryName,
			Color:            row.Color,
			Total:            row.Total,
			TransactionCount: row.TransactionCount,
			Percent:          percent,
		})
	}
	return out, nil
}

func (s *Service) monthlyTrend(ctx context.Context, q *generated.Queries, userID uuid.UUID, start, end period.Range) ([]MonthlyTotalDTO, error) {
	rows, err := q.GetMonthlyTotals(ctx, generated.GetMonthlyTotalsParams{
		UserID: pgUUIDValue(userID),
		FromTs: timeStampValue(start.Start),
		ToTs:   timeStampValue(end.End),
	})
	if err != nil {
		return nil, mapDBError(err)
	}
	out := make([]MonthlyTotalDTO, 0, len(rows))
	for _, row := range rows {
		out = append(out, MonthlyTotalDTO{
			Month:   row.Month.Time.Format("2006-01"),
			Income:  row.Income,
			Expense: row.Expense,
		})
	}
	return out, nil
}

func periodDTO(r period.Range) PeriodDTO {
	return PeriodDTO{
		Key:   r.Key,
		Label: r.Label,
		Start: r.Start.Format(time.RFC3339),
		End:   r.End.Format(time.RFC3339),
	}
}

func medianInt64(values []int64) int64 {
	if len(values) == 0 {
		return 0
	}
	sorted := append([]int64(nil), values...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	mid := len(sorted) / 2
	if len(sorted)%2 == 1 {
		return sorted[mid]
	}
	return (sorted[mid-1] + sorted[mid]) / 2
}

func medianAbsoluteDeviation(values []int64, median int64) float64 {
	if len(values) == 0 {
		return 0
	}
	deviations := make([]int64, 0, len(values))
	for _, v := range values {
		d := v - median
		if d < 0 {
			d = -d
		}
		deviations = append(deviations, d)
	}
	return float64(medianInt64(deviations))
}

// location resolves the user's timezone for period calculations.
func (s *Service) location(ctx context.Context, q *generated.Queries, userID uuid.UUID) *time.Location {
	tz, err := s.UserTimezone(ctx, q, userID)
	if err != nil {
		return period.Location("Asia/Jakarta")
	}
	return period.Location(tz)
}
