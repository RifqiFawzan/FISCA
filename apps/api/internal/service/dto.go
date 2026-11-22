package service

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/fisca-app/fisca/apps/api/internal/infrastructure/database/generated"
)

// UserDTO is the profile returned by GET /me.
type UserDTO struct {
	ID           string  `json:"id"`
	Email        string  `json:"email"`
	DisplayName  string  `json:"displayName"`
	Timezone     string  `json:"timezone"`
	Locale       string  `json:"locale"`
	BaseCurrency string  `json:"baseCurrency"`
	OnboardedAt  *string `json:"onboardedAt"`
	CreatedAt    string  `json:"createdAt"`
}

// AccountDTO includes the ledger-derived balance (never stored).
type AccountDTO struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Type           string `json:"type"`
	OpeningBalance int64  `json:"openingBalance"`
	Balance        int64  `json:"balance"`
	IsDefault      bool   `json:"isDefault"`
	CreatedAt      string `json:"createdAt"`
}

// CategoryDTO is a user-defined income/expense category.
type CategoryDTO struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	Kind      string  `json:"kind"`
	Icon      *string `json:"icon"`
	Color     *string `json:"color"`
	IsDefault bool    `json:"isDefault"`
	CreatedAt string  `json:"createdAt"`
}

// TransactionDTO is the stable shape used by every screen.
type TransactionDTO struct {
	ID              string  `json:"id"`
	Type            string  `json:"type"`
	Amount          int64   `json:"amount"`
	Currency        string  `json:"currency"`
	Description     string  `json:"description"`
	CategoryID      *string `json:"categoryId"`
	CategoryName    *string `json:"categoryName"`
	CategoryColor   *string `json:"categoryColor"`
	CategoryIcon    *string `json:"categoryIcon"`
	AccountID       *string `json:"accountId"`
	AccountName     *string `json:"accountName"`
	FromAccountID   *string `json:"fromAccountId"`
	FromAccountName *string `json:"fromAccountName"`
	ToAccountID     *string `json:"toAccountId"`
	ToAccountName   *string `json:"toAccountName"`
	PaymentMethod   *string `json:"paymentMethod"`
	MerchantName    *string `json:"merchantName"`
	OccurredAt      string  `json:"occurredAt"`
	Notes           *string `json:"notes"`
	Source          string  `json:"source"`
	Status          string  `json:"status"`
	VoidedAt        *string `json:"voidedAt"`
	VoidReason      *string `json:"voidReason"`
	Version         int32   `json:"version"`
	CreatedAt       string  `json:"createdAt"`
	UpdatedAt       string  `json:"updatedAt"`
}

// AuditDTO is one audit trail row.
type AuditDTO struct {
	ID        int64           `json:"id"`
	Actor     string          `json:"actor"`
	Action    string          `json:"action"`
	Before    json.RawMessage `json:"before"`
	After     json.RawMessage `json:"after"`
	CreatedAt string          `json:"createdAt"`
}

// TransactionDetail is the detail payload with its audit trail.
type TransactionDetail struct {
	Transaction TransactionDTO `json:"transaction"`
	Audit       []AuditDTO     `json:"audit"`
}

// SummaryDTO is the income/expense/net triple.
type SummaryDTO struct {
	Income  int64 `json:"income"`
	Expense int64 `json:"expense"`
	Net     int64 `json:"net"`
}

// CategorySpendDTO is one slice of the expense breakdown.
type CategorySpendDTO struct {
	CategoryID       *string `json:"categoryId"`
	Name             string  `json:"name"`
	Color            string  `json:"color"`
	Total            int64   `json:"total"`
	TransactionCount int64   `json:"transactionCount"`
	Percent          float64 `json:"percent"`
}

// BudgetCategoryStatus is the per-category budget progress.
type BudgetCategoryStatus struct {
	CategoryID string  `json:"categoryId"`
	Name       string  `json:"name"`
	Limit      int64   `json:"limit"`
	Spent      int64   `json:"spent"`
	Remaining  int64   `json:"remaining"`
	Percent    float64 `json:"percent"`
	Status     string  `json:"status"`
}

// BudgetDTO carries a budget plus its live, computed status.
type BudgetDTO struct {
	ID           string                 `json:"id"`
	PeriodStart  string                 `json:"periodStart"`
	PeriodEnd    string                 `json:"periodEnd"`
	OverallLimit int64                  `json:"overallLimit"`
	Currency     string                 `json:"currency"`
	Status       string                 `json:"status"`
	Version      int32                  `json:"version"`
	Spent        int64                  `json:"spent"`
	Remaining    int64                  `json:"remaining"`
	Percent      float64                `json:"percent"`
	Unallocated  int64                  `json:"unallocated"`
	Categories   []BudgetCategoryStatus `json:"categories"`
	CreatedAt    string                 `json:"createdAt"`
	UpdatedAt    string                 `json:"updatedAt"`
}

// BudgetProposalDTO is a pending or decided budget change.
type BudgetProposalDTO struct {
	ID          string          `json:"id"`
	BudgetID    string          `json:"budgetId"`
	BaseVersion int32           `json:"baseVersion"`
	Status      string          `json:"status"`
	Changes     json.RawMessage `json:"changes"`
	Rationale   *string         `json:"rationale"`
	Impact      json.RawMessage `json:"impact"`
	ExpiresAt   string          `json:"expiresAt"`
	DecidedAt   *string         `json:"decidedAt"`
	CreatedAt   string          `json:"createdAt"`
}

// Warning is attached to a successful transaction create (plan.md §8.9).
type Warning struct {
	Type           string `json:"type"`
	Message        string `json:"message"`
	BudgetID       string `json:"budgetId,omitempty"`
	CategoryID     string `json:"categoryId,omitempty"`
	ExceededAmount int64  `json:"exceededAmount,omitempty"`
}

// ---------------------------------------------------------------------------
// mappers
// ---------------------------------------------------------------------------

func userDTO(u generated.User) UserDTO {
	return UserDTO{
		ID:           uuidString(u.ID),
		Email:        u.Email,
		DisplayName:  u.DisplayName,
		Timezone:     u.Timezone,
		Locale:       u.Locale,
		BaseCurrency: u.BaseCurrency,
		OnboardedAt:  timePtr(u.OnboardedAt),
		CreatedAt:    timestamp(u.CreatedAt).Format(time.RFC3339),
	}
}

func accountDTO(a generated.Account, balance int64) AccountDTO {
	return AccountDTO{
		ID:             uuidString(a.ID),
		Name:           a.Name,
		Type:           a.Type,
		OpeningBalance: a.OpeningBalance,
		Balance:        balance,
		IsDefault:      a.IsDefault,
		CreatedAt:      timestamp(a.CreatedAt).Format(time.RFC3339),
	}
}

func categoryDTO(c generated.Category) CategoryDTO {
	return CategoryDTO{
		ID:        uuidString(c.ID),
		Name:      c.Name,
		Kind:      c.Kind,
		Icon:      c.Icon,
		Color:     c.Color,
		IsDefault: c.IsDefault,
		CreatedAt: timestamp(c.CreatedAt).Format(time.RFC3339),
	}
}

func transactionDTO(t generated.Transaction, names *txNameCache) TransactionDTO {
	dto := TransactionDTO{
		ID:            uuidString(t.ID),
		Type:          t.Type,
		Amount:        t.Amount,
		Currency:      t.Currency,
		Description:   t.Description,
		CategoryID:    uuidPtr(t.CategoryID),
		AccountID:     uuidPtr(t.AccountID),
		FromAccountID: uuidPtr(t.FromAccountID),
		ToAccountID:   uuidPtr(t.ToAccountID),
		PaymentMethod: t.PaymentMethod,
		MerchantName:  t.MerchantName,
		OccurredAt:    timestamp(t.OccurredAt).Format(time.RFC3339),
		Notes:         t.Notes,
		Source:        t.Source,
		Status:        t.Status,
		VoidedAt:      timePtr(t.VoidedAt),
		VoidReason:    t.VoidReason,
		Version:       t.Version,
		CreatedAt:     timestamp(t.CreatedAt).Format(time.RFC3339),
		UpdatedAt:     timestamp(t.UpdatedAt).Format(time.RFC3339),
	}
	if names != nil {
		if name, ok := names.account(t.AccountID); ok {
			dto.AccountName = &name
		}
		if name, ok := names.account(t.FromAccountID); ok {
			dto.FromAccountName = &name
		}
		if name, ok := names.account(t.ToAccountID); ok {
			dto.ToAccountName = &name
		}
		if c, ok := names.category(t.CategoryID); ok {
			dto.CategoryName = &c.Name
			dto.CategoryColor = c.Color
			dto.CategoryIcon = c.Icon
		}
	}
	return dto
}

func auditDTO(a generated.AuditLog) AuditDTO {
	return AuditDTO{
		ID:        a.ID,
		Actor:     a.Actor,
		Action:    a.Action,
		Before:    json.RawMessage(a.Before),
		After:     json.RawMessage(a.After),
		CreatedAt: timestamp(a.CreatedAt).Format(time.RFC3339),
	}
}

func budgetDTO(b generated.Budget, spent int64, categories []BudgetCategoryStatus, unallocated int64) BudgetDTO {
	remaining := b.OverallLimit - spent
	percent := 0.0
	if b.OverallLimit > 0 {
		percent = float64(spent) / float64(b.OverallLimit)
	}
	if categories == nil {
		categories = []BudgetCategoryStatus{}
	}
	return BudgetDTO{
		ID:           uuidString(b.ID),
		PeriodStart:  dateString(b.PeriodStart),
		PeriodEnd:    dateString(b.PeriodEnd),
		OverallLimit: b.OverallLimit,
		Currency:     b.Currency,
		Status:       b.Status,
		Version:      b.Version,
		Spent:        spent,
		Remaining:    remaining,
		Percent:      percent,
		Unallocated:  unallocated,
		Categories:   categories,
		CreatedAt:    timestamp(b.CreatedAt).Format(time.RFC3339),
		UpdatedAt:    timestamp(b.UpdatedAt).Format(time.RFC3339),
	}
}

func proposalDTO(p generated.BudgetProposal) BudgetProposalDTO {
	return BudgetProposalDTO{
		ID:          uuidString(p.ID),
		BudgetID:    uuidString(p.BudgetID),
		BaseVersion: p.BaseVersion,
		Status:      p.Status,
		Changes:     json.RawMessage(p.Changes),
		Rationale:   p.Rationale,
		Impact:      json.RawMessage(p.Impact),
		ExpiresAt:   timestamp(p.ExpiresAt).Format(time.RFC3339),
		DecidedAt:   timePtr(p.DecidedAt),
		CreatedAt:   timestamp(p.CreatedAt).Format(time.RFC3339),
	}
}

func timePtr(t pgtype.Timestamptz) *string {
	if !t.Valid {
		return nil
	}
	v := timestamp(t).Format(time.RFC3339)
	return &v
}

// txNameCache avoids N+1 lookups while mapping a page of transactions.
type txNameCache struct {
	ctx        context.Context
	q          *generated.Queries
	accounts   map[string]generated.Account
	categories map[string]generated.Category
	userID     pgtype.UUID
}

func newTxNameCache(ctx context.Context, q *generated.Queries, userID pgtype.UUID) *txNameCache {
	return &txNameCache{
		ctx:        ctx,
		q:          q,
		accounts:   map[string]generated.Account{},
		categories: map[string]generated.Category{},
		userID:     userID,
	}
}

func (c *txNameCache) account(id pgtype.UUID) (string, bool) {
	if !id.Valid {
		return "", false
	}
	key := uuidString(id)
	if a, ok := c.accounts[key]; ok {
		return a.Name, true
	}
	a, err := c.q.GetAccountForUser(c.ctx, generated.GetAccountForUserParams{ID: id, UserID: c.userID})
	if err != nil {
		return "", false
	}
	c.accounts[key] = a
	return a.Name, true
}

func (c *txNameCache) category(id pgtype.UUID) (generated.Category, bool) {
	if !id.Valid {
		return generated.Category{}, false
	}
	key := uuidString(id)
	if cat, ok := c.categories[key]; ok {
		return cat, true
	}
	cat, err := c.q.GetCategoryForUser(c.ctx, generated.GetCategoryForUserParams{ID: id, UserID: c.userID})
	if err != nil {
		return generated.Category{}, false
	}
	c.categories[key] = cat
	return cat, true
}
