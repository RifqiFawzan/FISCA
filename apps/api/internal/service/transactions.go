package service

import (
	"context"
	"encoding/base64"
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

// WriteContext carries request metadata written to the audit log.
type WriteContext struct {
	Actor     string
	RequestID string
}

// CreateTransactionInput is the POST /transactions body.
type CreateTransactionInput struct {
	Type          string     `json:"type"`
	Amount        int64      `json:"amount"`
	Description   string     `json:"description"`
	CategoryID    *string    `json:"categoryId"`
	AccountID     *string    `json:"accountId"`
	FromAccountID *string    `json:"fromAccountId"`
	ToAccountID   *string    `json:"toAccountId"`
	PaymentMethod *string    `json:"paymentMethod"`
	MerchantName  *string    `json:"merchantName"`
	OccurredAt    *time.Time `json:"occurredAt"`
	Notes         *string    `json:"notes"`
	Source        string     `json:"source"`
}

// ListTransactionsInput is the query filter for GET /transactions.
type ListTransactionsInput struct {
	Type       string
	CategoryID string
	AccountID  string
	Q          string
	Period     string
	From       string
	To         string
	Cursor     string
	Limit      int
	MinAmount  *int64
	MaxAmount  *int64
}

// TransactionPage is a cursor-paginated page.
type TransactionPage struct {
	Items      []TransactionDTO `json:"items"`
	NextCursor string           `json:"nextCursor,omitempty"`
}

var transactionSources = map[string]struct{}{
	"manual": {}, "voice": {}, "chat": {},
}

// CreateTransaction validates, inserts, audits, and returns budget warnings.
// Every capture path (manual, voice, chat, AI tool) funnels through here.
func (s *Service) CreateTransaction(
	ctx context.Context,
	q *generated.Queries,
	userID uuid.UUID,
	in CreateTransactionInput,
	wc WriteContext,
) (TransactionDTO, []Warning, error) {
	in.Type = strings.TrimSpace(in.Type)
	if in.Type != "income" && in.Type != "expense" && in.Type != "transfer" {
		return TransactionDTO{}, nil, Invalid("type", "Tipe transaksi harus income, expense, atau transfer")
	}
	if in.Amount <= 0 {
		return TransactionDTO{}, nil, Invalid("amount", "Nominal harus lebih dari 0")
	}
	if in.Amount > 1_000_000_000_000 {
		return TransactionDTO{}, nil, Invalid("amount", "Nominal terlalu besar")
	}
	in.Description = strings.TrimSpace(in.Description)
	if in.Description == "" {
		return TransactionDTO{}, nil, Invalid("description", "Deskripsi wajib diisi")
	}
	if len([]rune(in.Description)) > 500 {
		return TransactionDTO{}, nil, Invalid("description", "Deskripsi maksimal 500 karakter")
	}
	if in.Source == "" {
		in.Source = "manual"
	}
	if _, ok := transactionSources[in.Source]; !ok {
		return TransactionDTO{}, nil, Invalid("source", "Source harus manual, voice, atau chat")
	}

	occurredAt := s.NowUTC()
	if in.OccurredAt != nil {
		occurredAt = in.OccurredAt.UTC()
	}

	params := generated.CreateTransactionParams{
		UserID:        pgUUIDValue(userID),
		Type:          in.Type,
		Amount:        in.Amount,
		Description:   in.Description,
		PaymentMethod: optionalString(in.PaymentMethod),
		MerchantName:  optionalString(in.MerchantName),
		OccurredAt:    timeStampValue(occurredAt),
		Notes:         optionalString(in.Notes),
		Source:        in.Source,
	}

	switch in.Type {
	case "income", "expense":
		account, err := s.resolveAccount(ctx, q, userID, in.AccountID)
		if err != nil {
			return TransactionDTO{}, nil, err
		}
		params.AccountID = account.ID

		if in.CategoryID != nil && strings.TrimSpace(*in.CategoryID) != "" {
			categoryID, err := parseUUID(*in.CategoryID)
			if err != nil {
				return TransactionDTO{}, nil, err
			}
			category, err := q.GetCategoryForUser(ctx, generated.GetCategoryForUserParams{
				ID:     pgUUIDValue(categoryID),
				UserID: pgUUIDValue(userID),
			})
			if err != nil || category.ArchivedAt.Valid {
				return TransactionDTO{}, nil, NotFound("Kategori")
			}
			if category.Kind != in.Type {
				return TransactionDTO{}, nil, Invalid("categoryId", "Kategori tidak sesuai dengan tipe transaksi")
			}
			params.CategoryID = category.ID
		}
	case "transfer":
		from, err := s.resolveAccount(ctx, q, userID, in.FromAccountID)
		if err != nil {
			return TransactionDTO{}, nil, Invalid("fromAccountId", "Akun asal wajib dipilih")
		}
		to, err := s.resolveAccount(ctx, q, userID, in.ToAccountID)
		if err != nil {
			return TransactionDTO{}, nil, Invalid("toAccountId", "Akun tujuan wajib dipilih")
		}
		if from.ID == to.ID {
			return TransactionDTO{}, nil, Invalid("toAccountId", "Akun asal dan tujuan harus berbeda")
		}
		params.FromAccountID = from.ID
		params.ToAccountID = to.ID
	}

	row, err := q.CreateTransaction(ctx, params)
	if err != nil {
		return TransactionDTO{}, nil, mapDBError(err)
	}

	if err := s.audit(ctx, q, userID, wc, "transaction", row.ID, "create", nil, row); err != nil {
		return TransactionDTO{}, nil, err
	}

	cache := newTxNameCache(ctx, q, pgUUIDValue(userID))
	dto := transactionDTO(row, cache)
	warnings, err := s.budgetWarnings(ctx, q, userID, row.CategoryID, occurredAt)
	if err != nil {
		return TransactionDTO{}, nil, err
	}
	return dto, warnings, nil
}

// ListTransactions returns a cursor page of active transactions.
func (s *Service) ListTransactions(ctx context.Context, q *generated.Queries, userID uuid.UUID, in ListTransactionsInput) (TransactionPage, error) {
	params, err := s.listParams(ctx, q, userID, in)
	if err != nil {
		return TransactionPage{}, err
	}
	rows, err := q.ListTransactions(ctx, params)
	if err != nil {
		return TransactionPage{}, mapDBError(err)
	}

	cache := newTxNameCache(ctx, q, pgUUIDValue(userID))
	items := make([]TransactionDTO, 0, len(rows))
	for _, row := range rows {
		items = append(items, transactionDTO(row, cache))
	}

	page := TransactionPage{Items: items}
	if len(rows) == int(params.PageSize) && len(rows) > 0 {
		last := rows[len(rows)-1]
		page.NextCursor = encodeCursor(timestamp(last.OccurredAt), uuidString(last.ID))
	}
	return page, nil
}

// GetTransaction returns the transaction plus its audit trail.
func (s *Service) GetTransaction(ctx context.Context, q *generated.Queries, userID uuid.UUID, id string) (TransactionDetail, error) {
	txID, err := parseUUID(id)
	if err != nil {
		return TransactionDetail{}, err
	}
	row, err := q.GetTransactionForUser(ctx, generated.GetTransactionForUserParams{
		ID:     pgUUIDValue(txID),
		UserID: pgUUIDValue(userID),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return TransactionDetail{}, NotFound("Transaksi")
	}
	if err != nil {
		return TransactionDetail{}, mapDBError(err)
	}

	audits, err := q.ListAuditLogsForEntity(ctx, generated.ListAuditLogsForEntityParams{
		UserID:     pgUUIDValue(userID),
		EntityType: "transaction",
		EntityID:   pgUUIDValue(txID),
		PageSize:   50,
	})
	if err != nil {
		return TransactionDetail{}, mapDBError(err)
	}
	out := TransactionDetail{
		Transaction: transactionDTO(row, newTxNameCache(ctx, q, pgUUIDValue(userID))),
		Audit:       make([]AuditDTO, 0, len(audits)),
	}
	for _, a := range audits {
		out.Audit = append(out.Audit, auditDTO(a))
	}
	return out, nil
}

// UpdateTransactionInput is the PATCH /transactions/{id} body. Money-bearing
// changes require the client's version (optimistic lock).
type UpdateTransactionInput struct {
	Version     int32      `json:"version"`
	Description *string    `json:"description"`
	Amount      *int64     `json:"amount"`
	CategoryID  *string    `json:"categoryId"`
	OccurredAt  *time.Time `json:"occurredAt"`
	Notes       *string    `json:"notes"`
}

// UpdateTransaction edits a transaction and writes the audit diff.
func (s *Service) UpdateTransaction(
	ctx context.Context,
	q *generated.Queries,
	userID uuid.UUID,
	id string,
	in UpdateTransactionInput,
	wc WriteContext,
) (TransactionDTO, error) {
	txID, err := parseUUID(id)
	if err != nil {
		return TransactionDTO{}, err
	}
	if in.Version <= 0 {
		return TransactionDTO{}, Invalid("version", "Version wajib dikirim")
	}
	before, err := q.GetTransactionForUser(ctx, generated.GetTransactionForUserParams{
		ID:     pgUUIDValue(txID),
		UserID: pgUUIDValue(userID),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return TransactionDTO{}, NotFound("Transaksi")
	}
	if err != nil {
		return TransactionDTO{}, mapDBError(err)
	}
	if before.Status != "active" {
		return TransactionDTO{}, Conflict("Transaksi sudah dibatalkan", nil)
	}

	description := optionalString(in.Description)
	if description != nil {
		*description = strings.TrimSpace(*description)
		if *description == "" {
			return TransactionDTO{}, Invalid("description", "Deskripsi wajib diisi")
		}
	}
	if in.Amount != nil && *in.Amount <= 0 {
		return TransactionDTO{}, Invalid("amount", "Nominal harus lebih dari 0")
	}

	params := generated.UpdateTransactionParams{
		Description:     description,
		Amount:          in.Amount,
		ID:              pgUUIDValue(txID),
		UserID:          pgUUIDValue(userID),
		ExpectedVersion: in.Version,
	}
	if in.OccurredAt != nil {
		params.OccurredAt = timeStampValue(in.OccurredAt.UTC())
	}
	if in.Notes != nil {
		params.Notes = optionalString(in.Notes)
	}
	if in.CategoryID != nil && before.Type != "transfer" {
		categoryID, err := parseUUID(*in.CategoryID)
		if err != nil {
			return TransactionDTO{}, err
		}
		category, err := q.GetCategoryForUser(ctx, generated.GetCategoryForUserParams{
			ID:     pgUUIDValue(categoryID),
			UserID: pgUUIDValue(userID),
		})
		if err != nil || category.ArchivedAt.Valid {
			return TransactionDTO{}, NotFound("Kategori")
		}
		if category.Kind != before.Type {
			return TransactionDTO{}, Invalid("categoryId", "Kategori tidak sesuai dengan tipe transaksi")
		}
		params.CategoryID = category.ID
	}

	after, err := q.UpdateTransaction(ctx, params)
	if errors.Is(err, pgx.ErrNoRows) {
		latest, _ := q.GetTransactionForUser(ctx, generated.GetTransactionForUserParams{
			ID:     pgUUIDValue(txID),
			UserID: pgUUIDValue(userID),
		})
		return TransactionDTO{}, Conflict("Transaksi sudah berubah, muat ulang data", map[string]any{
			"latestVersion": latest.Version,
		})
	}
	if err != nil {
		return TransactionDTO{}, mapDBError(err)
	}

	if err := s.audit(ctx, q, userID, wc, "transaction", after.ID, "update", before, after); err != nil {
		return TransactionDTO{}, err
	}
	return transactionDTO(after, newTxNameCache(ctx, q, pgUUIDValue(userID))), nil
}

// VoidTransaction marks a transaction voided. It never deletes rows.
func (s *Service) VoidTransaction(
	ctx context.Context,
	q *generated.Queries,
	userID uuid.UUID,
	id string,
	reason string,
	wc WriteContext,
) (TransactionDTO, error) {
	txID, err := parseUUID(id)
	if err != nil {
		return TransactionDTO{}, err
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return TransactionDTO{}, Invalid("reason", "Alasan pembatalan wajib diisi")
	}
	before, err := q.GetTransactionForUser(ctx, generated.GetTransactionForUserParams{
		ID:     pgUUIDValue(txID),
		UserID: pgUUIDValue(userID),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return TransactionDTO{}, NotFound("Transaksi")
	}
	if err != nil {
		return TransactionDTO{}, mapDBError(err)
	}

	after, err := q.VoidTransaction(ctx, generated.VoidTransactionParams{
		Reason: &reason,
		ID:     pgUUIDValue(txID),
		UserID: pgUUIDValue(userID),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return TransactionDTO{}, Conflict("Transaksi sudah dibatalkan", nil)
	}
	if err != nil {
		return TransactionDTO{}, mapDBError(err)
	}

	if err := s.audit(ctx, q, userID, wc, "transaction", after.ID, "void", before, after); err != nil {
		return TransactionDTO{}, err
	}
	return transactionDTO(after, newTxNameCache(ctx, q, pgUUIDValue(userID))), nil
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func (s *Service) resolveAccount(ctx context.Context, q *generated.Queries, userID uuid.UUID, raw *string) (generated.Account, error) {
	if raw != nil && strings.TrimSpace(*raw) != "" {
		id, err := parseUUID(*raw)
		if err != nil {
			return generated.Account{}, err
		}
		account, err := q.GetAccountForUser(ctx, generated.GetAccountForUserParams{
			ID:     pgUUIDValue(id),
			UserID: pgUUIDValue(userID),
		})
		if err != nil || account.ArchivedAt.Valid {
			return generated.Account{}, NotFound("Akun")
		}
		return account, nil
	}

	account, err := q.GetDefaultAccount(ctx, pgUUIDValue(userID))
	if err == nil {
		return account, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return generated.Account{}, mapDBError(err)
	}

	accounts, err := q.ListAccountsByUser(ctx, pgUUIDValue(userID))
	if err != nil {
		return generated.Account{}, mapDBError(err)
	}
	if len(accounts) == 1 {
		return accounts[0], nil
	}
	return generated.Account{}, Validation("Akun belum dipilih", map[string]string{
		"field": "accountId",
		"hint":  "Pilih akun karena kamu punya lebih dari satu akun aktif",
	})
}

func (s *Service) listParams(ctx context.Context, q *generated.Queries, userID uuid.UUID, in ListTransactionsInput) (generated.ListTransactionsParams, error) {
	limit := in.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	params := generated.ListTransactionsParams{
		UserID:    pgUUIDValue(userID),
		PageSize:  int32(limit),
		MinAmount: in.MinAmount,
		MaxAmount: in.MaxAmount,
	}

	if t := strings.TrimSpace(in.Type); t != "" {
		if t != "income" && t != "expense" && t != "transfer" {
			return params, Invalid("type", "Tipe transaksi harus income, expense, atau transfer")
		}
		params.Type = &t
	}
	if in.CategoryID != "" {
		id, err := parseUUID(in.CategoryID)
		if err != nil {
			return params, err
		}
		params.CategoryID = pgUUIDValue(id)
	}
	if in.AccountID != "" {
		id, err := parseUUID(in.AccountID)
		if err != nil {
			return params, err
		}
		params.AccountID = pgUUIDValue(id)
	}
	if qstr := strings.TrimSpace(in.Q); qstr != "" {
		if len([]rune(qstr)) > 100 {
			return params, Invalid("q", "Pencarian maksimal 100 karakter")
		}
		params.Q = &qstr
	}

	tz, err := s.UserTimezone(ctx, q, userID)
	if err != nil {
		return params, err
	}
	loc := period.Location(tz)

	if in.Period != "" {
		r, err := period.Parse(in.Period, loc, s.Now())
		if err != nil {
			return params, Invalid("period", err.Error())
		}
		params.FromTs = timeStampValue(r.Start)
		params.ToTs = timeStampValue(r.End)
	}
	if in.From != "" {
		from, err := parseBoundary(in.From, loc)
		if err != nil {
			return params, Invalid("from", "Format tanggal tidak valid")
		}
		params.FromTs = timeStampValue(from)
	}
	if in.To != "" {
		to, err := parseBoundary(in.To, loc)
		if err != nil {
			return params, Invalid("to", "Format tanggal tidak valid")
		}
		params.ToTs = timeStampValue(to)
	}

	if in.Cursor != "" {
		ts, id, err := decodeCursor(in.Cursor)
		if err != nil {
			return params, Invalid("cursor", "Cursor tidak valid")
		}
		params.CursorTs = timeStampValue(ts)
		params.CursorID = pgUUIDValue(id)
	}
	return params, nil
}

func parseBoundary(raw string, loc *time.Location) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, raw); err == nil {
		return t.UTC(), nil
	}
	t, err := time.ParseInLocation("2006-01-02", raw, loc)
	if err != nil {
		return time.Time{}, err
	}
	return t.UTC(), nil
}

func encodeCursor(ts time.Time, id string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(ts.UTC().Format(time.RFC3339Nano) + "|" + id))
}

func decodeCursor(raw string) (time.Time, uuid.UUID, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return time.Time{}, uuid.Nil, err
	}
	parts := strings.SplitN(string(decoded), "|", 2)
	if len(parts) != 2 {
		return time.Time{}, uuid.Nil, errors.New("malformed cursor")
	}
	ts, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		return time.Time{}, uuid.Nil, err
	}
	id, err := uuid.Parse(parts[1])
	if err != nil {
		return time.Time{}, uuid.Nil, err
	}
	return ts, id, nil
}

func (s *Service) audit(ctx context.Context, q *generated.Queries, userID uuid.UUID, wc WriteContext, entityType string, entityID pgtype.UUID, action string, before, after any) error {
	actor := wc.Actor
	if actor == "" {
		actor = "user"
	}
	var beforeJSON, afterJSON []byte
	var err error
	if before != nil {
		beforeJSON, err = json.Marshal(before)
		if err != nil {
			return err
		}
	}
	if after != nil {
		afterJSON, err = json.Marshal(after)
		if err != nil {
			return err
		}
	}
	var requestID *string
	if wc.RequestID != "" {
		requestID = &wc.RequestID
	}
	return q.CreateAuditLog(ctx, generated.CreateAuditLogParams{
		UserID:     pgUUIDValue(userID),
		Actor:      actor,
		EntityType: entityType,
		EntityID:   entityID,
		Action:     action,
		Before:     beforeJSON,
		After:      afterJSON,
		RequestID:  requestID,
	})
}
