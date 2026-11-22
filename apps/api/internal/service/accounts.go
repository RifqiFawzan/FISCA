package service

import (
	"context"
	"strings"
	"time"

	"github.com/fisca-app/fisca/apps/api/internal/infrastructure/database/generated"
	"github.com/google/uuid"
)

// CreateAccountInput is the POST /accounts body.
type CreateAccountInput struct {
	Name           string `json:"name"`
	Type           string `json:"type"`
	OpeningBalance int64  `json:"openingBalance"`
	IsDefault      bool   `json:"isDefault"`
}

var accountTypes = map[string]struct{}{
	"cash": {}, "bank": {}, "ewallet": {}, "other": {},
}

// ListAccounts returns active accounts with ledger-derived balances.
func (s *Service) ListAccounts(ctx context.Context, q *generated.Queries, userID uuid.UUID) ([]AccountDTO, error) {
	rows, err := q.ListAccountBalances(ctx, pgUUIDValue(userID))
	if err != nil {
		return nil, mapDBError(err)
	}
	out := make([]AccountDTO, 0, len(rows))
	for _, r := range rows {
		out = append(out, AccountDTO{
			ID:             uuidString(r.ID),
			Name:           r.Name,
			Type:           r.Type,
			OpeningBalance: r.OpeningBalance,
			Balance:        r.Balance,
			IsDefault:      r.IsDefault,
			CreatedAt:      timestamp(r.CreatedAt).Format(time.RFC3339),
		})
	}
	return out, nil
}

// CreateAccount creates an account. The first account becomes the default.
func (s *Service) CreateAccount(ctx context.Context, q *generated.Queries, userID uuid.UUID, in CreateAccountInput) (AccountDTO, error) {
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" {
		return AccountDTO{}, Invalid("name", "Nama akun wajib diisi")
	}
	if _, ok := accountTypes[in.Type]; !ok {
		return AccountDTO{}, Invalid("type", "Tipe akun harus cash, bank, ewallet, atau other")
	}

	count, err := q.CountActiveAccounts(ctx, pgUUIDValue(userID))
	if err != nil {
		return AccountDTO{}, mapDBError(err)
	}
	makeDefault := in.IsDefault || count == 0
	if makeDefault && count > 0 {
		if err := q.ClearDefaultAccount(ctx, pgUUIDValue(userID)); err != nil {
			return AccountDTO{}, mapDBError(err)
		}
	}

	account, err := q.CreateAccount(ctx, generated.CreateAccountParams{
		UserID:         pgUUIDValue(userID),
		Name:           in.Name,
		Type:           in.Type,
		OpeningBalance: in.OpeningBalance,
		IsDefault:      makeDefault,
	})
	if err != nil {
		return AccountDTO{}, mapDBError(err)
	}
	return accountDTO(account, account.OpeningBalance), nil
}

// UpdateAccountInput is the PATCH /accounts/{id} body.
type UpdateAccountInput struct {
	Name *string `json:"name"`
	Type *string `json:"type"`
}

// UpdateAccount renames or retypes an account. Balances are never editable.
func (s *Service) UpdateAccount(ctx context.Context, q *generated.Queries, userID uuid.UUID, accountID string, in UpdateAccountInput) (AccountDTO, error) {
	id, err := parseUUID(accountID)
	if err != nil {
		return AccountDTO{}, err
	}
	name := optionalString(in.Name)
	if name != nil && strings.TrimSpace(*name) == "" {
		return AccountDTO{}, Invalid("name", "Nama akun wajib diisi")
	}
	if in.Type != nil {
		if _, ok := accountTypes[*in.Type]; !ok {
			return AccountDTO{}, Invalid("type", "Tipe akun harus cash, bank, ewallet, atau other")
		}
	}

	account, err := q.UpdateAccount(ctx, generated.UpdateAccountParams{
		Name:   name,
		Type:   in.Type,
		ID:     pgUUIDValue(id),
		UserID: pgUUIDValue(userID),
	})
	if err != nil {
		return AccountDTO{}, mapDBError(err)
	}

	balance := account.OpeningBalance
	rows, err := q.ListAccountBalances(ctx, pgUUIDValue(userID))
	if err == nil {
		for _, r := range rows {
			if uuidString(r.ID) == uuidString(account.ID) {
				balance = r.Balance
				break
			}
		}
	}
	return accountDTO(account, balance), nil
}

// ArchiveAccount soft-deletes an account. Archiving the default promotes the
// next active account so the user always has somewhere to record money.
func (s *Service) ArchiveAccount(ctx context.Context, q *generated.Queries, userID uuid.UUID, accountID string) error {
	id, err := parseUUID(accountID)
	if err != nil {
		return err
	}
	account, err := q.GetAccountForUser(ctx, generated.GetAccountForUserParams{ID: pgUUIDValue(id), UserID: pgUUIDValue(userID)})
	if err != nil {
		return NotFound("Akun")
	}
	if account.ArchivedAt.Valid {
		return NotFound("Akun")
	}

	if err := q.ArchiveAccount(ctx, generated.ArchiveAccountParams{ID: pgUUIDValue(id), UserID: pgUUIDValue(userID)}); err != nil {
		return mapDBError(err)
	}

	if account.IsDefault {
		remaining, err := q.ListAccountsByUser(ctx, pgUUIDValue(userID))
		if err != nil {
			return mapDBError(err)
		}
		if len(remaining) > 0 {
			return q.SetDefaultAccount(ctx, generated.SetDefaultAccountParams{
				ID:     remaining[0].ID,
				UserID: pgUUIDValue(userID),
			})
		}
	}
	return nil
}

// SetDefaultAccount marks one active account as the default.
func (s *Service) SetDefaultAccount(ctx context.Context, q *generated.Queries, userID uuid.UUID, accountID string) (AccountDTO, error) {
	id, err := parseUUID(accountID)
	if err != nil {
		return AccountDTO{}, err
	}
	account, err := q.GetAccountForUser(ctx, generated.GetAccountForUserParams{ID: pgUUIDValue(id), UserID: pgUUIDValue(userID)})
	if err != nil || account.ArchivedAt.Valid {
		return AccountDTO{}, NotFound("Akun")
	}
	if err := q.ClearDefaultAccount(ctx, pgUUIDValue(userID)); err != nil {
		return AccountDTO{}, mapDBError(err)
	}
	if err := q.SetDefaultAccount(ctx, generated.SetDefaultAccountParams{ID: pgUUIDValue(id), UserID: pgUUIDValue(userID)}); err != nil {
		return AccountDTO{}, mapDBError(err)
	}
	account.IsDefault = true
	return accountDTO(account, account.OpeningBalance), nil
}
