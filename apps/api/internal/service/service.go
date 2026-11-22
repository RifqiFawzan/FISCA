package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/fisca-app/fisca/apps/api/internal/ai"
	"github.com/fisca-app/fisca/apps/api/internal/config"
	"github.com/fisca-app/fisca/apps/api/internal/infrastructure/database/generated"
)

// Service implements the financial rules. It is stateless apart from the pool;
// methods that mutate money run inside a caller-provided transaction so audit
// rows and idempotency records commit atomically (plan.md §8.5, §8.10).
type Service struct {
	Pool *pgxpool.Pool
	Q    *generated.Queries
	Cfg  *config.Config
	AI   *ai.Client
	Now  func() time.Time
}

// New builds a Service.
func New(pool *pgxpool.Pool, cfg *config.Config) *Service {
	return &Service{Pool: pool, Q: generated.New(pool), Cfg: cfg, Now: time.Now}
}

// NowUTC returns the current time in UTC.
func (s *Service) NowUTC() time.Time { return s.Now().UTC() }

// WithTx runs fn inside a transaction and commits only when fn succeeds.
func (s *Service) WithTx(ctx context.Context, fn func(q *generated.Queries) error) error {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := fn(generated.New(tx)); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}

// EnsureUser returns the user row, provisioning it on first login.
func (s *Service) EnsureUser(ctx context.Context, q *generated.Queries, id uuid.UUID, email, name string) (generated.User, error) {
	user, err := q.GetUserByID(ctx, pgUUIDValue(id))
	if err == nil {
		return user, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return generated.User{}, err
	}

	email = strings.TrimSpace(email)
	if email == "" {
		email = id.String() + "@fisca.local"
	}
	name = strings.TrimSpace(name)
	if name == "" {
		name = "Pengguna FISCA"
	}
	user, err = q.UpsertUser(ctx, generated.UpsertUserParams{
		ID:          pgUUIDValue(id),
		Email:       email,
		DisplayName: name,
	})
	if err != nil {
		return generated.User{}, mapDBError(err)
	}
	if err := s.EnsureDefaults(ctx, q, id); err != nil {
		return generated.User{}, err
	}
	return user, nil
}

type defaultCategory struct {
	Name  string `json:"name"`
	Kind  string `json:"kind"`
	Icon  string `json:"icon"`
	Color string `json:"color"`
}

var defaultCategories = []defaultCategory{
	{Name: "Makanan & Minuman", Kind: "expense", Icon: "utensils", Color: "#FB7185"},
	{Name: "Transportasi", Kind: "expense", Icon: "car", Color: "#38BDF8"},
	{Name: "Belanja", Kind: "expense", Icon: "shopping-bag", Color: "#FBBF24"},
	{Name: "Tagihan & Utilitas", Kind: "expense", Icon: "receipt", Color: "#6366F1"},
	{Name: "Hiburan", Kind: "expense", Icon: "popcorn", Color: "#A78BFA"},
	{Name: "Kesehatan", Kind: "expense", Icon: "heart-pulse", Color: "#F43F5E"},
	{Name: "Pendidikan", Kind: "expense", Icon: "graduation-cap", Color: "#60A5FA"},
	{Name: "Gaji", Kind: "income", Icon: "banknote", Color: "#34D399"},
	{Name: "Freelance", Kind: "income", Icon: "laptop", Color: "#10B981"},
	{Name: "Pemasukan Lain", Kind: "income", Icon: "plus-circle", Color: "#94A3B8"},
}

// EnsureDefaults gives a fresh user one cash account and the default categories.
func (s *Service) EnsureDefaults(ctx context.Context, q *generated.Queries, userID uuid.UUID) error {
	count, err := q.CountActiveAccounts(ctx, pgUUIDValue(userID))
	if err != nil {
		return err
	}
	if count == 0 {
		if _, err := q.CreateAccount(ctx, generated.CreateAccountParams{
			UserID:    pgUUIDValue(userID),
			Name:      "Cash",
			Type:      "cash",
			IsDefault: true,
		}); err != nil {
			return err
		}
	}

	categories, err := q.ListCategoriesByUser(ctx, generated.ListCategoriesByUserParams{UserID: pgUUIDValue(userID)})
	if err != nil {
		return err
	}
	if len(categories) > 0 {
		return nil
	}

	raw, err := json.Marshal(defaultCategories)
	if err != nil {
		return err
	}
	return q.SeedDefaultCategories(ctx, generated.SeedDefaultCategoriesParams{
		UserID: pgUUIDValue(userID),
		Items:  raw,
	})
}

// ---------------------------------------------------------------------------
// conversion helpers
// ---------------------------------------------------------------------------

func pgUUIDValue(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: id, Valid: true}
}

func parseUUID(raw string) (uuid.UUID, error) {
	id, err := uuid.Parse(strings.TrimSpace(raw))
	if err != nil {
		return uuid.Nil, Validation("ID tidak valid", map[string]string{"id": raw})
	}
	return id, nil
}

func pgUUIDOpt(raw *string) (pgtype.UUID, error) {
	if raw == nil || strings.TrimSpace(*raw) == "" {
		return pgtype.UUID{}, nil
	}
	id, err := parseUUID(*raw)
	if err != nil {
		return pgtype.UUID{}, err
	}
	return pgUUIDValue(id), nil
}

func uuidString(id pgtype.UUID) string {
	if !id.Valid {
		return ""
	}
	return uuid.UUID(id.Bytes).String()
}

func uuidPtr(id pgtype.UUID) *string {
	if !id.Valid {
		return nil
	}
	v := uuidString(id)
	return &v
}

func timestamp(t pgtype.Timestamptz) time.Time {
	if !t.Valid {
		return time.Time{}
	}
	return t.Time.UTC()
}

func timeStampValue(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t.UTC(), Valid: true}
}

func dateValue(t time.Time) pgtype.Date {
	return pgtype.Date{Time: periodDate(t), Valid: true}
}

func dateString(d pgtype.Date) string {
	if !d.Valid {
		return ""
	}
	return d.Time.Format("2006-01-02")
}

func periodDate(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

func optionalString(raw *string) *string {
	if raw == nil {
		return nil
	}
	v := strings.TrimSpace(*raw)
	if v == "" {
		return nil
	}
	return &v
}

// mapDBError translates common PostgreSQL constraint violations into domain
// errors. Unknown errors are returned untouched (500 at the HTTP layer).
func mapDBError(err error) error {
	if err == nil {
		return nil
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			switch pgErr.ConstraintName {
			case "accounts_user_name_unique":
				return Conflict("Nama akun sudah dipakai", nil)
			case "categories_user_name_kind_unique":
				return Conflict("Nama kategori sudah dipakai", nil)
			case "budgets_user_period_unique":
				return Conflict("Budget untuk periode ini sudah ada", nil)
			default:
				return Conflict("Data sudah ada", nil)
			}
		case "23503":
			return Validation("Data terkait tidak ditemukan", nil)
		case "23514":
			return Validation("Data melanggar aturan validasi", nil)
		}
	}
	return err
}
