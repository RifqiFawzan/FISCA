package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/fisca-app/fisca/apps/api/internal/infrastructure/database/generated"
	"github.com/fisca-app/fisca/apps/api/internal/period"
)

// AllocationInput is one category limit.
type AllocationInput struct {
	CategoryID  string `json:"categoryId"`
	LimitAmount int64  `json:"limitAmount"`
}

// CreateBudgetInput is the POST /budgets body.
type CreateBudgetInput struct {
	Period       string            `json:"period"`
	OverallLimit int64             `json:"overallLimit"`
	Allocations  []AllocationInput `json:"allocations"`
}

// UpdateBudgetInput is the PATCH /budgets/{id} body.
type UpdateBudgetInput struct {
	Version      int32              `json:"version"`
	OverallLimit *int64             `json:"overallLimit"`
	Allocations  *[]AllocationInput `json:"allocations"`
}

// BudgetChanges is the JSON stored on a proposal. A nil allocations pointer
// means "leave allocations unchanged"; an empty slice clears them.
type BudgetChanges struct {
	OverallLimit *int64             `json:"overallLimit,omitempty"`
	Allocations  *[]AllocationInput `json:"allocations,omitempty"`
}

// CreateProposalInput is the POST /budget-proposals body.
type CreateProposalInput struct {
	BudgetID  string        `json:"budgetId"`
	Changes   BudgetChanges `json:"changes"`
	Rationale *string       `json:"rationale"`
}

// UpdateProposalInput is the PATCH /budget-proposals/{id} body.
type UpdateProposalInput struct {
	Changes   *BudgetChanges `json:"changes"`
	Rationale *string        `json:"rationale"`
}

// ListBudgets returns active budgets with computed status.
func (s *Service) ListBudgets(ctx context.Context, q *generated.Queries, userID uuid.UUID) ([]BudgetDTO, error) {
	rows, err := q.ListBudgetsByUser(ctx, pgUUIDValue(userID))
	if err != nil {
		return nil, mapDBError(err)
	}
	out := make([]BudgetDTO, 0, len(rows))
	for _, row := range rows {
		dto, err := s.budgetStatus(ctx, q, userID, row)
		if err != nil {
			return nil, err
		}
		out = append(out, dto)
	}
	return out, nil
}

// GetBudget returns one budget with computed status.
func (s *Service) GetBudget(ctx context.Context, q *generated.Queries, userID uuid.UUID, id string) (BudgetDTO, error) {
	budgetID, err := parseUUID(id)
	if err != nil {
		return BudgetDTO{}, err
	}
	budget, err := q.GetBudgetForUser(ctx, generated.GetBudgetForUserParams{
		ID:     pgUUIDValue(budgetID),
		UserID: pgUUIDValue(userID),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return BudgetDTO{}, NotFound("Budget")
	}
	if err != nil {
		return BudgetDTO{}, mapDBError(err)
	}
	return s.budgetStatus(ctx, q, userID, budget)
}

// CreateBudget creates a monthly budget and its initial allocations.
func (s *Service) CreateBudget(ctx context.Context, q *generated.Queries, userID uuid.UUID, in CreateBudgetInput, wc WriteContext) (BudgetDTO, error) {
	if in.OverallLimit <= 0 {
		return BudgetDTO{}, Invalid("overallLimit", "Limit budget harus lebih dari 0")
	}
	r, err := s.resolvePeriod(ctx, q, userID, in.Period)
	if err != nil {
		return BudgetDTO{}, err
	}
	if err := s.validateAllocations(ctx, q, userID, in.Allocations, in.OverallLimit); err != nil {
		return BudgetDTO{}, err
	}

	budget, err := q.CreateBudget(ctx, generated.CreateBudgetParams{
		UserID:       pgUUIDValue(userID),
		PeriodStart:  dateValue(r.StartDate),
		PeriodEnd:    dateValue(r.EndDate),
		OverallLimit: in.OverallLimit,
	})
	if err != nil {
		return BudgetDTO{}, mapDBError(err)
	}
	if err := s.replaceAllocations(ctx, q, budget.ID, in.Allocations); err != nil {
		return BudgetDTO{}, err
	}
	if err := s.audit(ctx, q, userID, wc, "budget", budget.ID, "create", nil, budget); err != nil {
		return BudgetDTO{}, err
	}
	return s.budgetStatus(ctx, q, userID, budget)
}

// UpdateBudget edits a budget with optimistic locking.
func (s *Service) UpdateBudget(ctx context.Context, q *generated.Queries, userID uuid.UUID, id string, in UpdateBudgetInput, wc WriteContext) (BudgetDTO, error) {
	budgetID, err := parseUUID(id)
	if err != nil {
		return BudgetDTO{}, err
	}
	if in.Version <= 0 {
		return BudgetDTO{}, Invalid("version", "Version wajib dikirim")
	}
	before, err := q.GetBudgetForUser(ctx, generated.GetBudgetForUserParams{
		ID:     pgUUIDValue(budgetID),
		UserID: pgUUIDValue(userID),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return BudgetDTO{}, NotFound("Budget")
	}
	if err != nil {
		return BudgetDTO{}, mapDBError(err)
	}

	overall := before.OverallLimit
	if in.OverallLimit != nil {
		overall = *in.OverallLimit
	}
	if overall <= 0 {
		return BudgetDTO{}, Invalid("overallLimit", "Limit budget harus lebih dari 0")
	}

	allocations := in.Allocations
	if allocations != nil {
		if err := s.validateAllocations(ctx, q, userID, *allocations, overall); err != nil {
			return BudgetDTO{}, err
		}
	} else {
		existing, err := q.ListBudgetCategories(ctx, pgUUIDValue(budgetID))
		if err != nil {
			return BudgetDTO{}, mapDBError(err)
		}
		var allocated int64
		for _, a := range existing {
			allocated += a.LimitAmount
		}
		if allocated > overall {
			return BudgetDTO{}, Invalid("overallLimit", "Limit keseluruhan lebih kecil dari total alokasi kategori")
		}
	}

	updated, err := q.UpdateBudget(ctx, generated.UpdateBudgetParams{
		OverallLimit:    &overall,
		ID:              pgUUIDValue(budgetID),
		UserID:          pgUUIDValue(userID),
		ExpectedVersion: in.Version,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return BudgetDTO{}, BudgetStale("Budget sudah berubah, muat ulang data", map[string]any{
			"latestVersion": before.Version,
		})
	}
	if err != nil {
		return BudgetDTO{}, mapDBError(err)
	}
	if allocations != nil {
		if err := s.replaceAllocations(ctx, q, pgUUIDValue(budgetID), *allocations); err != nil {
			return BudgetDTO{}, err
		}
	}
	if err := s.audit(ctx, q, userID, wc, "budget", pgUUIDValue(budgetID), "update", before, updated); err != nil {
		return BudgetDTO{}, err
	}
	return s.budgetStatus(ctx, q, userID, updated)
}

// GetBudgetStatus returns computed actuals for one budget.
func (s *Service) GetBudgetStatus(ctx context.Context, q *generated.Queries, userID uuid.UUID, id string) (BudgetDTO, error) {
	return s.GetBudget(ctx, q, userID, id)
}

// budgetForPeriod returns the active budget whose period covers r, if any.
func (s *Service) budgetForPeriod(ctx context.Context, q *generated.Queries, userID uuid.UUID, r period.Range) (*BudgetDTO, error) {
	budget, err := q.GetBudgetForPeriod(ctx, generated.GetBudgetForPeriodParams{
		UserID:      pgUUIDValue(userID),
		PeriodStart: dateValue(r.StartDate),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, mapDBError(err)
	}
	dto, err := s.budgetStatus(ctx, q, userID, budget)
	if err != nil {
		return nil, err
	}
	return &dto, nil
}

func (s *Service) budgetStatus(ctx context.Context, q *generated.Queries, userID uuid.UUID, budget generated.Budget) (BudgetDTO, error) {
	spend, err := q.GetBudgetSpend(ctx, generated.GetBudgetSpendParams{
		BudgetID: budget.ID,
		UserID:   pgUUIDValue(userID),
	})
	if err != nil {
		return BudgetDTO{}, mapDBError(err)
	}
	rows, err := q.GetCategoryBudgetSpend(ctx, generated.GetCategoryBudgetSpendParams{
		BudgetID: budget.ID,
		UserID:   pgUUIDValue(userID),
	})
	if err != nil {
		return BudgetDTO{}, mapDBError(err)
	}

	categories := make([]BudgetCategoryStatus, 0, len(rows))
	var allocated int64
	for _, row := range rows {
		allocated += row.LimitAmount
		categories = append(categories, BudgetCategoryStatus{
			CategoryID: uuidString(row.CategoryID),
			Name:       row.CategoryName,
			Limit:      row.LimitAmount,
			Spent:      row.Spent,
			Remaining:  row.LimitAmount - row.Spent,
			Percent:    ratio(row.Spent, row.LimitAmount),
			Status:     budgetHealth(row.Spent, row.LimitAmount, s.Cfg.BudgetWarningThreshold),
		})
	}
	return budgetDTO(budget, spend.Spent, categories, budget.OverallLimit-allocated), nil
}

func (s *Service) validateAllocations(
	ctx context.Context,
	q *generated.Queries,
	userID uuid.UUID,
	allocations []AllocationInput,
	overall int64,
) error {
	seen := map[string]struct{}{}
	var total int64
	for i, allocation := range allocations {
		if allocation.LimitAmount < 0 {
			return Invalid("allocations", "Limit kategori tidak boleh negatif")
		}
		id, err := parseUUID(allocation.CategoryID)
		if err != nil {
			return err
		}
		key := id.String()
		if _, ok := seen[key]; ok {
			return Invalid("allocations", "Kategori tidak boleh muncul dua kali")
		}
		seen[key] = struct{}{}

		category, err := q.GetCategoryForUser(ctx, generated.GetCategoryForUserParams{
			ID:     pgUUIDValue(id),
			UserID: pgUUIDValue(userID),
		})
		if err != nil || category.ArchivedAt.Valid {
			return NotFound("Kategori")
		}
		if category.Kind != "expense" {
			return Invalid("allocations", "Hanya kategori pengeluaran yang bisa dianggarkan")
		}
		total += allocation.LimitAmount
		_ = i
	}
	if total > overall {
		return Invalid("allocations", "Total alokasi kategori melebihi limit keseluruhan")
	}
	return nil
}

func (s *Service) replaceAllocations(ctx context.Context, q *generated.Queries, budgetID pgtype.UUID, allocations []AllocationInput) error {
	if err := q.DeleteAllBudgetCategories(ctx, budgetID); err != nil {
		return mapDBError(err)
	}
	for _, allocation := range allocations {
		id, err := parseUUID(allocation.CategoryID)
		if err != nil {
			return err
		}
		if _, err := q.UpsertBudgetCategory(ctx, generated.UpsertBudgetCategoryParams{
			BudgetID:    budgetID,
			CategoryID:  pgUUIDValue(id),
			LimitAmount: allocation.LimitAmount,
		}); err != nil {
			return mapDBError(err)
		}
	}
	return nil
}

// CreateProposal stores a pending change; it never touches the active budget.
func (s *Service) CreateProposal(ctx context.Context, q *generated.Queries, userID uuid.UUID, in CreateProposalInput, wc WriteContext) (BudgetProposalDTO, error) {
	budgetID, err := parseUUID(in.BudgetID)
	if err != nil {
		return BudgetProposalDTO{}, err
	}
	budget, err := q.GetBudgetForUser(ctx, generated.GetBudgetForUserParams{
		ID:     pgUUIDValue(budgetID),
		UserID: pgUUIDValue(userID),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return BudgetProposalDTO{}, NotFound("Budget")
	}
	if err != nil {
		return BudgetProposalDTO{}, mapDBError(err)
	}
	if budget.Status != "active" {
		return BudgetProposalDTO{}, Conflict("Budget sudah diarsipkan", nil)
	}

	impact, err := s.proposalImpact(ctx, q, userID, budget, in.Changes)
	if err != nil {
		return BudgetProposalDTO{}, err
	}
	changesJSON, err := json.Marshal(in.Changes)
	if err != nil {
		return BudgetProposalDTO{}, err
	}
	impactJSON, err := json.Marshal(impact)
	if err != nil {
		return BudgetProposalDTO{}, err
	}

	proposal, err := q.CreateBudgetProposal(ctx, generated.CreateBudgetProposalParams{
		UserID:      pgUUIDValue(userID),
		BudgetID:    budget.ID,
		BaseVersion: budget.Version,
		Changes:     changesJSON,
		Rationale:   optionalString(in.Rationale),
		Impact:      impactJSON,
	})
	if err != nil {
		return BudgetProposalDTO{}, mapDBError(err)
	}
	if err := s.audit(ctx, q, userID, wc, "budget_proposal", proposal.ID, "create", nil, proposal); err != nil {
		return BudgetProposalDTO{}, err
	}
	return proposalDTO(proposal), nil
}

// GetProposal returns one proposal.
func (s *Service) GetProposal(ctx context.Context, q *generated.Queries, userID uuid.UUID, id string) (BudgetProposalDTO, error) {
	proposalID, err := parseUUID(id)
	if err != nil {
		return BudgetProposalDTO{}, err
	}
	proposal, err := q.GetBudgetProposalForUser(ctx, generated.GetBudgetProposalForUserParams{
		ID:     pgUUIDValue(proposalID),
		UserID: pgUUIDValue(userID),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return BudgetProposalDTO{}, NotFound("Proposal")
	}
	if err != nil {
		return BudgetProposalDTO{}, mapDBError(err)
	}
	return proposalDTO(proposal), nil
}

// ListProposals lists proposals, optionally filtered by status.
func (s *Service) ListProposals(ctx context.Context, q *generated.Queries, userID uuid.UUID, status string, limit int) ([]BudgetProposalDTO, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	var statusPtr *string
	if st := strings.TrimSpace(status); st != "" {
		switch st {
		case "pending", "approved", "rejected", "expired", "stale":
			statusPtr = &st
		default:
			return nil, Invalid("status", "Status proposal tidak valid")
		}
	}
	rows, err := q.ListBudgetProposals(ctx, generated.ListBudgetProposalsParams{
		UserID:   pgUUIDValue(userID),
		Status:   statusPtr,
		PageSize: int32(limit),
	})
	if err != nil {
		return nil, mapDBError(err)
	}
	out := make([]BudgetProposalDTO, 0, len(rows))
	for _, row := range rows {
		out = append(out, proposalDTO(row))
	}
	return out, nil
}

// UpdateProposal edits a pending proposal.
func (s *Service) UpdateProposal(ctx context.Context, q *generated.Queries, userID uuid.UUID, id string, in UpdateProposalInput) (BudgetProposalDTO, error) {
	proposalID, err := parseUUID(id)
	if err != nil {
		return BudgetProposalDTO{}, err
	}
	proposal, err := q.GetBudgetProposalForUser(ctx, generated.GetBudgetProposalForUserParams{
		ID:     pgUUIDValue(proposalID),
		UserID: pgUUIDValue(userID),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return BudgetProposalDTO{}, NotFound("Proposal")
	}
	if err != nil {
		return BudgetProposalDTO{}, mapDBError(err)
	}
	if proposal.Status != "pending" {
		return BudgetProposalDTO{}, Conflict("Proposal sudah diputuskan", nil)
	}

	changesJSON := proposal.Changes
	if in.Changes != nil {
		budget, err := q.GetBudgetForUser(ctx, generated.GetBudgetForUserParams{
			ID:     proposal.BudgetID,
			UserID: pgUUIDValue(userID),
		})
		if err != nil {
			return BudgetProposalDTO{}, mapDBError(err)
		}
		if _, err := s.proposalImpact(ctx, q, userID, budget, *in.Changes); err != nil {
			return BudgetProposalDTO{}, err
		}
		changesJSON, err = json.Marshal(*in.Changes)
		if err != nil {
			return BudgetProposalDTO{}, err
		}
	}

	updated, err := q.UpdateBudgetProposalChanges(ctx, generated.UpdateBudgetProposalChangesParams{
		Changes:   changesJSON,
		Rationale: optionalString(in.Rationale),
		ID:        pgUUIDValue(proposalID),
		UserID:    pgUUIDValue(userID),
	})
	if err != nil {
		return BudgetProposalDTO{}, mapDBError(err)
	}
	return proposalDTO(updated), nil
}

// ApproveProposal applies a pending change atomically. The budget row is locked
// FOR UPDATE and the version must match both the client and the proposal.
func (s *Service) ApproveProposal(ctx context.Context, q *generated.Queries, userID uuid.UUID, id string, baseVersion int32, wc WriteContext) (BudgetDTO, error) {
	proposalID, err := parseUUID(id)
	if err != nil {
		return BudgetDTO{}, err
	}
	proposal, err := q.LockBudgetProposal(ctx, generated.LockBudgetProposalParams{
		ID:     pgUUIDValue(proposalID),
		UserID: pgUUIDValue(userID),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return BudgetDTO{}, NotFound("Proposal")
	}
	if err != nil {
		return BudgetDTO{}, mapDBError(err)
	}
	if proposal.Status != "pending" {
		return BudgetDTO{}, Conflict("Proposal sudah diputuskan", map[string]any{"status": proposal.Status})
	}
	if timestamp(proposal.ExpiresAt).Before(s.NowUTC()) {
		return BudgetDTO{}, Conflict("Proposal sudah kedaluwarsa", nil)
	}

	budget, err := q.LockBudget(ctx, generated.LockBudgetParams{
		ID:     proposal.BudgetID,
		UserID: pgUUIDValue(userID),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return BudgetDTO{}, NotFound("Budget")
	}
	if err != nil {
		return BudgetDTO{}, mapDBError(err)
	}
	if baseVersion == 0 {
		baseVersion = proposal.BaseVersion
	}
	if budget.Version != proposal.BaseVersion || budget.Version != baseVersion {
		return BudgetDTO{}, BudgetStale("Budget sudah berubah sejak proposal dibuat", map[string]any{
			"baseVersion":    baseVersion,
			"proposalBase":   proposal.BaseVersion,
			"currentVersion": budget.Version,
		})
	}

	var changes BudgetChanges
	if err := json.Unmarshal(proposal.Changes, &changes); err != nil {
		return BudgetDTO{}, err
	}
	overall := budget.OverallLimit
	if changes.OverallLimit != nil {
		overall = *changes.OverallLimit
	}
	if overall <= 0 {
		return BudgetDTO{}, Invalid("overallLimit", "Limit budget harus lebih dari 0")
	}
	if changes.Allocations != nil {
		if err := s.validateAllocations(ctx, q, userID, *changes.Allocations, overall); err != nil {
			return BudgetDTO{}, err
		}
	} else {
		existing, err := q.ListBudgetCategories(ctx, budget.ID)
		if err != nil {
			return BudgetDTO{}, mapDBError(err)
		}
		var allocated int64
		for _, a := range existing {
			allocated += a.LimitAmount
		}
		if allocated > overall {
			return BudgetDTO{}, Invalid("overallLimit", "Limit keseluruhan lebih kecil dari total alokasi kategori")
		}
	}

	updated, err := q.UpdateBudget(ctx, generated.UpdateBudgetParams{
		OverallLimit:    &overall,
		ID:              budget.ID,
		UserID:          pgUUIDValue(userID),
		ExpectedVersion: budget.Version,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return BudgetDTO{}, BudgetStale("Budget sudah berubah sejak proposal dibuat", nil)
	}
	if err != nil {
		return BudgetDTO{}, mapDBError(err)
	}
	if changes.Allocations != nil {
		if err := s.replaceAllocations(ctx, q, budget.ID, *changes.Allocations); err != nil {
			return BudgetDTO{}, err
		}
	}
	if _, err := q.DecideBudgetProposal(ctx, generated.DecideBudgetProposalParams{
		Status: "approved",
		ID:     proposal.ID,
		UserID: pgUUIDValue(userID),
	}); err != nil {
		return BudgetDTO{}, mapDBError(err)
	}
	if err := s.audit(ctx, q, userID, wc, "budget", budget.ID, "approve_proposal", budget, updated); err != nil {
		return BudgetDTO{}, err
	}
	return s.budgetStatus(ctx, q, userID, updated)
}

// RejectProposal closes a pending proposal without touching the budget.
func (s *Service) RejectProposal(ctx context.Context, q *generated.Queries, userID uuid.UUID, id string, wc WriteContext) (BudgetProposalDTO, error) {
	proposalID, err := parseUUID(id)
	if err != nil {
		return BudgetProposalDTO{}, err
	}
	proposal, err := q.DecideBudgetProposal(ctx, generated.DecideBudgetProposalParams{
		Status: "rejected",
		ID:     pgUUIDValue(proposalID),
		UserID: pgUUIDValue(userID),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		// Not owned by this user (or absent): 404, never reveal another user's
		// row. Owned but already decided: 409.
		if _, lookupErr := q.GetBudgetProposalForUser(ctx, generated.GetBudgetProposalForUserParams{
			ID:     pgUUIDValue(proposalID),
			UserID: pgUUIDValue(userID),
		}); errors.Is(lookupErr, pgx.ErrNoRows) {
			return BudgetProposalDTO{}, NotFound("Proposal")
		} else if lookupErr != nil {
			return BudgetProposalDTO{}, mapDBError(lookupErr)
		}
		return BudgetProposalDTO{}, Conflict("Proposal sudah diputuskan", nil)
	}
	if err != nil {
		return BudgetProposalDTO{}, mapDBError(err)
	}
	if err := s.audit(ctx, q, userID, wc, "budget_proposal", proposal.ID, "reject", nil, proposal); err != nil {
		return BudgetProposalDTO{}, err
	}
	return proposalDTO(proposal), nil
}

type proposalImpactPayload struct {
	Before budgetImpactSnapshot `json:"before"`
	After  budgetImpactSnapshot `json:"after"`
}

type budgetImpactSnapshot struct {
	OverallLimit int64 `json:"overallLimit"`
	Allocated    int64 `json:"allocated"`
	Spent        int64 `json:"spent"`
	Remaining    int64 `json:"remaining"`
}

func (s *Service) proposalImpact(ctx context.Context, q *generated.Queries, userID uuid.UUID, budget generated.Budget, changes BudgetChanges) (proposalImpactPayload, error) {
	spend, err := q.GetBudgetSpend(ctx, generated.GetBudgetSpendParams{
		BudgetID: budget.ID,
		UserID:   pgUUIDValue(userID),
	})
	if err != nil {
		return proposalImpactPayload{}, mapDBError(err)
	}
	existing, err := q.ListBudgetCategories(ctx, budget.ID)
	if err != nil {
		return proposalImpactPayload{}, mapDBError(err)
	}
	var beforeAllocated int64
	for _, a := range existing {
		beforeAllocated += a.LimitAmount
	}

	afterOverall := budget.OverallLimit
	if changes.OverallLimit != nil {
		afterOverall = *changes.OverallLimit
	}
	if afterOverall <= 0 {
		return proposalImpactPayload{}, Invalid("overallLimit", "Limit budget harus lebih dari 0")
	}
	afterAllocated := beforeAllocated
	if changes.Allocations != nil {
		if err := s.validateAllocations(ctx, q, userID, *changes.Allocations, afterOverall); err != nil {
			return proposalImpactPayload{}, err
		}
		afterAllocated = 0
		for _, a := range *changes.Allocations {
			afterAllocated += a.LimitAmount
		}
	} else if beforeAllocated > afterOverall {
		return proposalImpactPayload{}, Invalid("overallLimit", "Limit keseluruhan lebih kecil dari total alokasi kategori")
	}

	return proposalImpactPayload{
		Before: budgetImpactSnapshot{
			OverallLimit: budget.OverallLimit,
			Allocated:    beforeAllocated,
			Spent:        spend.Spent,
			Remaining:    budget.OverallLimit - spend.Spent,
		},
		After: budgetImpactSnapshot{
			OverallLimit: afterOverall,
			Allocated:    afterAllocated,
			Spent:        spend.Spent,
			Remaining:    afterOverall - spend.Spent,
		},
	}, nil
}

func ratio(spent, limit int64) float64 {
	if limit <= 0 {
		return 0
	}
	return float64(spent) / float64(limit)
}

func budgetHealth(spent, limit int64, threshold float64) string {
	if limit <= 0 {
		return "ok"
	}
	if spent > limit {
		return "exceeded"
	}
	if float64(spent)/float64(limit) >= threshold {
		return "warning"
	}
	return "ok"
}

var _ = time.Time{}
