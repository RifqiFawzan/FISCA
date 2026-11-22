package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/fisca-app/fisca/apps/api/internal/infrastructure/database/generated"
	"github.com/fisca-app/fisca/apps/api/internal/period"
)

// budgetWarnings computes over-budget warnings at write time. Creating a
// transaction is never rejected because of a budget (plan.md §8.9).
func (s *Service) budgetWarnings(
	ctx context.Context,
	q *generated.Queries,
	userID uuid.UUID,
	categoryID pgtype.UUID,
	occurredAt time.Time,
) ([]Warning, error) {
	tz, err := s.UserTimezone(ctx, q, userID)
	if err != nil {
		return nil, err
	}
	r := period.MonthOf(occurredAt, period.Location(tz))

	budget, err := q.GetBudgetForPeriod(ctx, generated.GetBudgetForPeriodParams{
		UserID:      pgUUIDValue(userID),
		PeriodStart: dateValue(r.StartDate),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return []Warning{}, nil
	}
	if err != nil {
		return nil, mapDBError(err)
	}

	spend, err := q.GetBudgetSpend(ctx, generated.GetBudgetSpendParams{
		BudgetID: budget.ID,
		UserID:   pgUUIDValue(userID),
	})
	if err != nil {
		return nil, mapDBError(err)
	}

	threshold := s.Cfg.BudgetWarningThreshold
	warnings := make([]Warning, 0, 2)
	if warning, ok := buildBudgetWarning("overall", budget.ID, pgtype.UUID{}, budget.OverallLimit, spend.Spent, threshold, "Budget keseluruhan"); ok {
		warnings = append(warnings, warning)
	}

	if categoryID.Valid {
		rows, err := q.GetCategoryBudgetSpend(ctx, generated.GetCategoryBudgetSpendParams{
			BudgetID: budget.ID,
			UserID:   pgUUIDValue(userID),
		})
		if err != nil {
			return nil, mapDBError(err)
		}
		for _, row := range rows {
			if row.CategoryID != categoryID {
				continue
			}
			if warning, ok := buildBudgetWarning("category", budget.ID, categoryID, row.LimitAmount, row.Spent, threshold, "Budget "+row.CategoryName); ok {
				warnings = append(warnings, warning)
			}
			break
		}
	}
	return warnings, nil
}

func buildBudgetWarning(scope string, budgetID, categoryID pgtype.UUID, limit, spent int64, threshold float64, label string) (Warning, bool) {
	if limit <= 0 {
		return Warning{}, false
	}
	ratio := float64(spent) / float64(limit)
	budgetIDStr := uuidString(budgetID)

	if spent > limit {
		message := fmt.Sprintf("%s terlampaui sebesar Rp%s", label, formatRupiahPlain(spent-limit))
		if scope == "category" {
			message = fmt.Sprintf("%s melebihi anggaran sebesar Rp%s", label, formatRupiahPlain(spent-limit))
		}
		return Warning{
			Type:           "BUDGET_EXCEEDED",
			Message:        message,
			BudgetID:       budgetIDStr,
			CategoryID:     uuidString(categoryID),
			ExceededAmount: spent - limit,
		}, true
	}
	if ratio >= threshold {
		message := fmt.Sprintf("%s sudah terpakai %d%%", label, int(ratio*100))
		return Warning{
			Type:       "BUDGET_WARNING",
			Message:    message,
			BudgetID:   budgetIDStr,
			CategoryID: uuidString(categoryID),
		}, true
	}
	return Warning{}, false
}

func formatRupiahPlain(amount int64) string {
	negative := amount < 0
	if negative {
		amount = -amount
	}
	digits := fmt.Sprint(amount)
	var out []byte
	for i, d := range []byte(digits) {
		if i > 0 && (len(digits)-i)%3 == 0 {
			out = append(out, '.')
		}
		out = append(out, d)
	}
	if negative {
		return "-" + string(out)
	}
	return string(out)
}
