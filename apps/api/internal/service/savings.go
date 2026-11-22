package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/fisca-app/fisca/apps/api/internal/infrastructure/database/generated"
	"github.com/fisca-app/fisca/apps/api/internal/period"
)

// SavingsGoalDTO is one goal with its progress derived from the linked
// account's ledger balance. There is no stored balance: the goal is a target,
// not a wallet, so the money is counted exactly once (PRD O-17).
type SavingsGoalDTO struct {
	ID              string             `json:"id"`
	Name            string             `json:"name"`
	TargetAmount    int64              `json:"targetAmount"`
	Deadline        string             `json:"deadline"`
	AccountID       *string            `json:"accountId"`
	AccountName     *string            `json:"accountName"`
	Saved           int64              `json:"saved"`
	Remaining       int64              `json:"remaining"`
	Percent         float64            `json:"percent"`
	Status          string             `json:"status"`
	MonthsLeft      int                `json:"monthsLeft"`
	RequiredMonthly int64              `json:"requiredMonthly"`
	Projection      *SavingsProjection `json:"projection,omitempty"`
	Version         int32              `json:"version"`
	CreatedAt       string             `json:"createdAt"`
	UpdatedAt       string             `json:"updatedAt"`
}

// SavingsProjection answers "what if I save X per month" (PRD §10.4).
type SavingsProjection struct {
	MonthlyContribution  int64 `json:"monthlyContribution"`
	MonthsToTarget       int   `json:"monthsToTarget"`
	ProjectedOnDeadline  int64 `json:"projectedOnDeadline"`
	AchievableByDeadline bool  `json:"achievableByDeadline"`
}

// PurchaseAdvisorDTO is a deterministic affordability simulation (PRD §10.6).
// Every number comes from the ledger; nothing is invented.
type PurchaseAdvisorDTO struct {
	Price                  int64    `json:"price"`
	Currency               string   `json:"currency"`
	Available              int64    `json:"available"`
	ReservedSavings        int64    `json:"reservedSavings"`
	FreeToSpend            int64    `json:"freeToSpend"`
	RemainingBudget        int64    `json:"remainingBudget"`
	AverageMonthlyIncome   int64    `json:"averageMonthlyIncome"`
	AverageMonthlyExpense  int64    `json:"averageMonthlyExpense"`
	MonthsWithData         int      `json:"monthsWithData"`
	Verdict                string   `json:"verdict"`
	ProjectedAfterPurchase int64    `json:"projectedAfterPurchase"`
	Assumptions            []string `json:"assumptions"`
	Limitations            []string `json:"limitations"`
}

// CreateSavingsGoalInput is the POST /savings-goals body.
type CreateSavingsGoalInput struct {
	Name         string `json:"name"`
	TargetAmount int64  `json:"targetAmount"`
	Deadline     string `json:"deadline"`
	AccountID    string `json:"accountId"`
}

// UpdateSavingsGoalInput is the PATCH /savings-goals/{id} body. A nil pointer
// means "keep the current value"; the account can be re-pointed but not
// unlinked, because progress would stop being measurable.
type UpdateSavingsGoalInput struct {
	Version      int32   `json:"version"`
	Name         *string `json:"name"`
	TargetAmount *int64  `json:"targetAmount"`
	Deadline     *string `json:"deadline"`
	AccountID    *string `json:"accountId"`
}

const savingsDateLayout = "2006-01-02"

// CreateSavingsGoal stores a new goal. The account must belong to the user:
// progress is read from it on every request.
func (s *Service) CreateSavingsGoal(
	ctx context.Context,
	q *generated.Queries,
	userID uuid.UUID,
	in CreateSavingsGoalInput,
	wc WriteContext,
) (SavingsGoalDTO, error) {
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" {
		return SavingsGoalDTO{}, Invalid("name", "Nama target wajib diisi")
	}
	if len([]rune(in.Name)) > 120 {
		return SavingsGoalDTO{}, Invalid("name", "Nama target maksimal 120 karakter")
	}
	if in.TargetAmount <= 0 {
		return SavingsGoalDTO{}, Invalid("targetAmount", "Nominal target harus lebih dari 0")
	}
	deadline, err := s.parseDeadline(in.Deadline)
	if err != nil {
		return SavingsGoalDTO{}, err
	}
	accountID, err := parseUUID(in.AccountID)
	if err != nil {
		return SavingsGoalDTO{}, Invalid("accountId", "Akun tidak valid")
	}
	accounts, err := s.accountIndex(ctx, q, userID)
	if err != nil {
		return SavingsGoalDTO{}, err
	}
	if _, ok := accounts[uuidString(pgUUIDValue(accountID))]; !ok {
		return SavingsGoalDTO{}, Invalid("accountId", "Akun tidak ditemukan untuk pengguna ini")
	}

	goal, err := q.CreateSavingsGoal(ctx, generated.CreateSavingsGoalParams{
		UserID:       pgUUIDValue(userID),
		Name:         in.Name,
		TargetAmount: in.TargetAmount,
		Deadline:     dateValue(deadline),
		AccountID:    pgUUIDValue(accountID),
	})
	if err != nil {
		return SavingsGoalDTO{}, mapDBError(err)
	}

	dto := s.savingsDTO(goal, accounts, s.nowInUserZone(ctx, q, userID), nil)
	if err := s.audit(ctx, q, userID, wc, "savings_goal", goal.ID, "create", nil, dto); err != nil {
		return SavingsGoalDTO{}, err
	}
	return dto, nil
}

// ListSavingsGoals returns every active goal with live progress.
func (s *Service) ListSavingsGoals(
	ctx context.Context,
	q *generated.Queries,
	userID uuid.UUID,
) ([]SavingsGoalDTO, error) {
	accounts, err := s.accountIndex(ctx, q, userID)
	if err != nil {
		return nil, err
	}
	goals, err := q.ListSavingsGoalsByUser(ctx, pgUUIDValue(userID))
	if err != nil {
		return nil, mapDBError(err)
	}
	now := s.nowInUserZone(ctx, q, userID)
	out := make([]SavingsGoalDTO, 0, len(goals))
	for _, goal := range goals {
		out = append(out, s.savingsDTO(goal, accounts, now, nil))
	}
	return out, nil
}

// GetSavingsGoal returns one goal. Passing a positive monthlyContribution adds
// a projection so the UI can simulate a different saving rate (PRD §10.4).
func (s *Service) GetSavingsGoal(
	ctx context.Context,
	q *generated.Queries,
	userID uuid.UUID,
	id string,
	monthly int64,
) (SavingsGoalDTO, error) {
	goal, err := parseUUID(id)
	if err != nil {
		return SavingsGoalDTO{}, err
	}
	accounts, err := s.accountIndex(ctx, q, userID)
	if err != nil {
		return SavingsGoalDTO{}, err
	}
	row, err := q.GetSavingsGoalForUser(ctx, generated.GetSavingsGoalForUserParams{
		ID:     pgUUIDValue(goal),
		UserID: pgUUIDValue(userID),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return SavingsGoalDTO{}, NotFound("Target tabungan")
	}
	if err != nil {
		return SavingsGoalDTO{}, mapDBError(err)
	}

	var projection *SavingsProjection
	if monthly > 0 {
		p := savingsProjection(s.savingsDTO(row, accounts, s.nowInUserZone(ctx, q, userID), nil), monthly)
		projection = &p
	}
	return s.savingsDTO(row, accounts, s.nowInUserZone(ctx, q, userID), projection), nil
}

// UpdateSavingsGoal renames, re-targets, re-deadlines or re-points a goal.
func (s *Service) UpdateSavingsGoal(
	ctx context.Context,
	q *generated.Queries,
	userID uuid.UUID,
	id string,
	in UpdateSavingsGoalInput,
	wc WriteContext,
) (SavingsGoalDTO, error) {
	goalID, err := parseUUID(id)
	if err != nil {
		return SavingsGoalDTO{}, err
	}
	if in.Version <= 0 {
		return SavingsGoalDTO{}, Invalid("version", "Version wajib dikirim")
	}

	before, err := q.GetSavingsGoalForUser(ctx, generated.GetSavingsGoalForUserParams{
		ID:     pgUUIDValue(goalID),
		UserID: pgUUIDValue(userID),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return SavingsGoalDTO{}, NotFound("Target tabungan")
	}
	if err != nil {
		return SavingsGoalDTO{}, mapDBError(err)
	}

	merged := generated.UpdateSavingsGoalParams{
		Name:            before.Name,
		TargetAmount:    before.TargetAmount,
		Deadline:        before.Deadline,
		AccountID:       before.AccountID,
		ID:              before.ID,
		UserID:          pgUUIDValue(userID),
		ExpectedVersion: in.Version,
	}
	if in.Name != nil {
		name := strings.TrimSpace(*in.Name)
		if name == "" {
			return SavingsGoalDTO{}, Invalid("name", "Nama target wajib diisi")
		}
		if len([]rune(name)) > 120 {
			return SavingsGoalDTO{}, Invalid("name", "Nama target maksimal 120 karakter")
		}
		merged.Name = name
	}
	if in.TargetAmount != nil {
		if *in.TargetAmount <= 0 {
			return SavingsGoalDTO{}, Invalid("targetAmount", "Nominal target harus lebih dari 0")
		}
		merged.TargetAmount = *in.TargetAmount
	}
	if in.Deadline != nil {
		deadline, err := s.parseDeadline(*in.Deadline)
		if err != nil {
			return SavingsGoalDTO{}, err
		}
		merged.Deadline = dateValue(deadline)
	}
	if in.AccountID != nil {
		accountID, err := parseUUID(*in.AccountID)
		if err != nil {
			return SavingsGoalDTO{}, Invalid("accountId", "Akun tidak valid")
		}
		accounts, err := s.accountIndex(ctx, q, userID)
		if err != nil {
			return SavingsGoalDTO{}, err
		}
		if _, ok := accounts[uuidString(pgUUIDValue(accountID))]; !ok {
			return SavingsGoalDTO{}, Invalid("accountId", "Akun tidak ditemukan untuk pengguna ini")
		}
		merged.AccountID = pgUUIDValue(accountID)
	}

	updated, err := q.UpdateSavingsGoal(ctx, merged)
	if errors.Is(err, pgx.ErrNoRows) {
		return SavingsGoalDTO{}, Conflict("Target tabungan sudah berubah, muat ulang data", map[string]any{
			"latestVersion": before.Version,
		})
	}
	if err != nil {
		return SavingsGoalDTO{}, mapDBError(err)
	}

	accounts, err := s.accountIndex(ctx, q, userID)
	if err != nil {
		return SavingsGoalDTO{}, err
	}
	after := s.savingsDTO(updated, accounts, s.nowInUserZone(ctx, q, userID), nil)
	if err := s.audit(ctx, q, userID, wc, "savings_goal", updated.ID, "update", before, updated); err != nil {
		return SavingsGoalDTO{}, err
	}
	return after, nil
}

// ArchiveSavingsGoal hides a goal without touching any money.
func (s *Service) ArchiveSavingsGoal(
	ctx context.Context,
	q *generated.Queries,
	userID uuid.UUID,
	id string,
	wc WriteContext,
) error {
	goalID, err := parseUUID(id)
	if err != nil {
		return err
	}
	before, err := q.GetSavingsGoalForUser(ctx, generated.GetSavingsGoalForUserParams{
		ID:     pgUUIDValue(goalID),
		UserID: pgUUIDValue(userID),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return NotFound("Target tabungan")
	}
	if err != nil {
		return mapDBError(err)
	}
	if err := q.ArchiveSavingsGoal(ctx, generated.ArchiveSavingsGoalParams{
		ID:     before.ID,
		UserID: pgUUIDValue(userID),
	}); err != nil {
		return mapDBError(err)
	}
	return s.audit(ctx, q, userID, wc, "savings_goal", before.ID, "archive", before, nil)
}

// PurchaseAdvisor simulates a purchase against money that actually exists
// (PRD §10.6). It deliberately reports what it could not verify instead of
// guessing: obligations are not modelled in FISCA, so they are stated as a
// limitation rather than assumed to be zero.
func (s *Service) PurchaseAdvisor(
	ctx context.Context,
	q *generated.Queries,
	userID uuid.UUID,
	price int64,
) (PurchaseAdvisorDTO, error) {
	if price <= 0 {
		return PurchaseAdvisorDTO{}, Invalid("price", "Harga harus lebih dari 0")
	}

	accounts, err := s.ListAccounts(ctx, q, userID)
	if err != nil {
		return PurchaseAdvisorDTO{}, err
	}
	var available int64
	for _, account := range accounts {
		available += account.Balance
	}

	// Reserved savings: the shortfall of every goal is spoken for, so the same
	// rupiah is never offered twice.
	accountsByID := make(map[string]AccountDTO, len(accounts))
	for _, account := range accounts {
		accountsByID[account.ID] = account
	}
	goals, err := q.ListSavingsGoalsByUser(ctx, pgUUIDValue(userID))
	if err != nil {
		return PurchaseAdvisorDTO{}, mapDBError(err)
	}
	var reserved int64
	for _, goal := range goals {
		saved := int64(0)
		if account, ok := accountsByID[uuidString(goal.AccountID)]; ok {
			saved = account.Balance
		}
		if shortfall := goal.TargetAmount - saved; shortfall > 0 {
			reserved += shortfall
		}
	}

	loc := s.location(ctx, q, userID)
	window, err := period.Parse("3m", loc, s.Now())
	if err != nil {
		return PurchaseAdvisorDTO{}, err
	}
	totals, err := q.GetPeriodTotals(ctx, generated.GetPeriodTotalsParams{
		UserID: pgUUIDValue(userID),
		FromTs: timeStampValue(window.Start),
		ToTs:   timeStampValue(window.End),
	})
	if err != nil {
		return PurchaseAdvisorDTO{}, mapDBError(err)
	}
	months, err := q.CountActiveMonths(ctx, generated.CountActiveMonthsParams{
		UserID: pgUUIDValue(userID),
		FromTs: timeStampValue(window.Start),
		ToTs:   timeStampValue(window.End),
	})
	if err != nil {
		return PurchaseAdvisorDTO{}, mapDBError(err)
	}

	var avgIncome, avgExpense int64
	if months > 0 {
		avgIncome = totals.Income / int64(months)
		avgExpense = totals.Expense / int64(months)
	}

	var remainingBudget int64
	budgetRange, err := s.resolvePeriod(ctx, q, userID, "this_month")
	if err == nil {
		if budget, budgetErr := s.budgetForPeriod(ctx, q, userID, budgetRange); budgetErr == nil && budget != nil {
			remainingBudget = budget.Remaining
		}
	}

	free := available - reserved
	verdict := "not_affordable"
	switch {
	case price <= free:
		verdict = "affordable"
	case months >= 2 && avgIncome > avgExpense && free+3*(avgIncome-avgExpense) >= price:
		verdict = "tight"
	}

	assumptions := []string{
		"Saldo dihitung dari seluruh akun aktif lewat ledger, bukan kolom saldo tersimpan.",
		"Kekurangan target tabungan diperuntukkan dulu, sehingga uangnya tidak ditawarkan dua kali.",
		"Sisa budget tidak dijumlahkan ke dana bebas karena uangnya sudah termasuk dalam saldo.",
	}
	limitations := []string{
		"Kewajiban dan utang belum dicatat di FISCA, jadi cicilan tidak ikut diperhitungkan.",
		"Hasil berupa simulasi informasional, bukan jaminan kemampuan finansial (PRD §10.6–10.7).",
	}
	if months < 2 {
		limitations = append(limitations,
			fmt.Sprintf("Riwayat transaksi baru %d bulan, rata-rata bulanan belum stabil.", months))
	}
	if avgIncome == 0 {
		limitations = append(limitations, "Belum ada pemasukan tercatat dalam 90 hari terakhir.")
	}
	if remainingBudget == 0 {
		limitations = append(limitations, "Belum ada sisa budget periode berjalan.")
	}

	return PurchaseAdvisorDTO{
		Price:                  price,
		Currency:               "IDR",
		Available:              available,
		ReservedSavings:        reserved,
		FreeToSpend:            free,
		RemainingBudget:        remainingBudget,
		AverageMonthlyIncome:   avgIncome,
		AverageMonthlyExpense:  avgExpense,
		MonthsWithData:         int(months),
		Verdict:                verdict,
		ProjectedAfterPurchase: available - price,
		Assumptions:            assumptions,
		Limitations:            limitations,
	}, nil
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func (s *Service) accountIndex(
	ctx context.Context,
	q *generated.Queries,
	userID uuid.UUID,
) (map[string]AccountDTO, error) {
	accounts, err := s.ListAccounts(ctx, q, userID)
	if err != nil {
		return nil, err
	}
	index := make(map[string]AccountDTO, len(accounts))
	for _, account := range accounts {
		index[account.ID] = account
	}
	return index, nil
}

// nowInUserZone keeps deadline arithmetic in the user's timezone.
func (s *Service) nowInUserZone(ctx context.Context, q *generated.Queries, userID uuid.UUID) time.Time {
	return s.Now().In(s.location(ctx, q, userID))
}

// parseDeadline accepts YYYY-MM-DD and refuses a date that has already passed:
// a goal created in the past would be born overdue.
func (s *Service) parseDeadline(raw string) (time.Time, error) {
	parsed, err := time.ParseInLocation(savingsDateLayout, strings.TrimSpace(raw), time.UTC)
	if err != nil {
		return time.Time{}, Invalid("deadline", "Tenggat harus berformat YYYY-MM-DD")
	}
	today := s.Now().UTC()
	if !parsed.After(time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.UTC)) {
		return time.Time{}, Invalid("deadline", "Tenggat harus setelah hari ini")
	}
	return parsed, nil
}

// savingsDTO projects a stored goal onto its live account balance.
func (s *Service) savingsDTO(
	goal generated.SavingsGoal,
	accounts map[string]AccountDTO,
	now time.Time,
	projection *SavingsProjection,
) SavingsGoalDTO {
	var saved int64
	var accountID, accountName *string
	if goal.AccountID.Valid {
		id := uuidString(goal.AccountID)
		accountID = &id
		if account, ok := accounts[id]; ok {
			accountName = &account.Name
			// An overdrawn pot has not saved anything.
			if account.Balance > 0 {
				saved = account.Balance
			}
		}
	}

	remaining := goal.TargetAmount - saved
	if remaining < 0 {
		remaining = 0
	}
	percent := 0.0
	if goal.TargetAmount > 0 {
		percent = float64(saved) / float64(goal.TargetAmount)
	}

	deadline := goal.Deadline.Time
	monthsLeft := monthsUntil(now, deadline)
	var required int64
	if monthsLeft > 0 {
		required = int64(math.Ceil(float64(remaining) / float64(monthsLeft)))
	} else {
		required = remaining
	}

	return SavingsGoalDTO{
		ID:              uuidString(goal.ID),
		Name:            goal.Name,
		TargetAmount:    goal.TargetAmount,
		Deadline:        goal.Deadline.Time.Format(savingsDateLayout),
		AccountID:       accountID,
		AccountName:     accountName,
		Saved:           saved,
		Remaining:       remaining,
		Percent:         percent,
		Status:          savingsStatus(saved, goal.TargetAmount, deadline, goal.CreatedAt.Time, now),
		MonthsLeft:      monthsLeft,
		RequiredMonthly: required,
		Projection:      projection,
		Version:         goal.Version,
		CreatedAt:       timestamp(goal.CreatedAt).Format(time.RFC3339),
		UpdatedAt:       timestamp(goal.UpdatedAt).Format(time.RFC3339),
	}
}

// savingsStatus is a simple, explainable rule: a goal is on track when its
// progress keeps pace with the time already spent. The first tenth of the
// window is a grace period, so a goal created a minute ago is not judged as
// behind. It is informational, not financial advice (PRD §10.7).
func savingsStatus(saved, target int64, deadline, created, now time.Time) string {
	if saved >= target {
		return "achieved"
	}
	if !deadline.After(now) {
		return "overdue"
	}
	span := deadline.Sub(created)
	if span <= 0 {
		return "at_risk"
	}
	elapsed := now.Sub(created)
	if elapsed < 0 {
		elapsed = 0
	}
	timeFraction := float64(elapsed) / float64(span)
	if timeFraction < 0.1 {
		return "on_track"
	}
	if float64(saved)/float64(target) >= timeFraction {
		return "on_track"
	}
	return "at_risk"
}

// savingsProjection answers "what if I save monthly per month until the
// deadline". monthsToTarget is the raw duration at that rate, with no time
// limit: a rupiah per month honestly reports how absurd it is.
func savingsProjection(goal SavingsGoalDTO, monthly int64) SavingsProjection {
	monthsToTarget := 0
	if monthly > 0 {
		monthsToTarget = int(math.Ceil(float64(goal.Remaining) / float64(monthly)))
	}
	projected := goal.Saved + monthly*int64(maxInt(goal.MonthsLeft, 0))
	return SavingsProjection{
		MonthlyContribution:  monthly,
		MonthsToTarget:       monthsToTarget,
		ProjectedOnDeadline:  projected,
		AchievableByDeadline: projected >= goal.TargetAmount,
	}
}

// monthsUntil counts whole months left until the deadline, never below zero,
// and never below one while the deadline is still ahead.
func monthsUntil(now, deadline time.Time) int {
	if !deadline.After(now) {
		return 0
	}
	months := (deadline.Year()-now.Year())*12 + int(deadline.Month()-now.Month())
	if deadline.Day() < now.Day() {
		months--
	}
	if months < 1 {
		return 1
	}
	return months
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
