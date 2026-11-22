package database_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/fisca-app/fisca/apps/api/internal/infrastructure/database/generated"
)

// TestTransactionQueriesAreUserScoped proves that generated queries cannot read
// another user's transaction. Requires a migrated database:
//
//	TEST_DATABASE_URL=postgres://fisca:fisca@localhost:5434/fisca?sslmode=disable go test ./internal/infrastructure/database/...
func TestTransactionQueriesAreUserScoped(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set TEST_DATABASE_URL to run database integration tests")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer pool.Close()

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	owner := uuid.New()
	other := uuid.New()

	_, err = tx.Exec(ctx,
		`INSERT INTO users (id, email, display_name) VALUES ($1, $2, $3), ($4, $5, $6)`,
		owner, owner.String()+"@test.local", "Owner",
		other, other.String()+"@test.local", "Other",
	)
	if err != nil {
		t.Fatalf("insert users: %v", err)
	}

	var accountID uuid.UUID
	err = tx.QueryRow(ctx,
		`INSERT INTO accounts (user_id, name, type, is_default)
		 VALUES ($1, 'Cash', 'cash', true)
		 RETURNING id`,
		owner,
	).Scan(&accountID)
	if err != nil {
		t.Fatalf("insert account: %v", err)
	}

	queries := generated.New(tx)

	created, err := queries.CreateTransaction(ctx, generated.CreateTransactionParams{
		UserID:      pgtype.UUID{Bytes: owner, Valid: true},
		Type:        "expense",
		Amount:      25000,
		Description: "Kopi",
		AccountID:   pgtype.UUID{Bytes: accountID, Valid: true},
		OccurredAt:  pgtype.Timestamptz{Time: time.Now(), Valid: true},
		Source:      "manual",
	})
	if err != nil {
		t.Fatalf("create transaction: %v", err)
	}

	if _, err := queries.GetTransactionForUser(ctx, generated.GetTransactionForUserParams{
		ID:     created.ID,
		UserID: pgtype.UUID{Bytes: other, Valid: true},
	}); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("cross-user read: err = %v, want pgx.ErrNoRows", err)
	}

	got, err := queries.GetTransactionForUser(ctx, generated.GetTransactionForUserParams{
		ID:     created.ID,
		UserID: pgtype.UUID{Bytes: owner, Valid: true},
	})
	if err != nil {
		t.Fatalf("owner read: %v", err)
	}
	if got.ID != created.ID {
		t.Fatalf("owner read returned %v, want %v", got.ID, created.ID)
	}

	list, err := queries.ListTransactions(ctx, generated.ListTransactionsParams{
		UserID:   pgtype.UUID{Bytes: other, Valid: true},
		PageSize: 10,
	})
	if err != nil {
		t.Fatalf("list other user: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("other user sees %d transactions, want 0", len(list))
	}
}
