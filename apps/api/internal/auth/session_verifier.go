package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/fisca-app/fisca/apps/api/internal/infrastructure/database/generated"
)

// sessionVerifier memverifikasi token sesi lokal dengan membaca tabel sessions.
//
// Verifikasi menyentuh database pada setiap request. Untuk pemakaian pribadi
// biayanya jauh di bawah cache tambahan; keuntungannya logout langsung berlaku
// karena tidak ada token yang masih sah secara kriptografis di sisi klien.
type sessionVerifier struct {
	q *generated.Queries
}

// NewSessionVerifier membangun verifier untuk AUTH_MODE=session.
func NewSessionVerifier(pool *pgxpool.Pool) Verifier {
	return &sessionVerifier{q: generated.New(pool)}
}

func (*sessionVerifier) Mode() string { return "session" }

// Verify mengembalikan user id pemilik token. Token yang tidak ada, sudah
// dicabut, atau lewat masa berlaku semuanya menghasilkan ErrInvalidToken —
// pesan seragam agar tidak bocor informasi.
func (v *sessionVerifier) Verify(ctx context.Context, rawToken string) (string, error) {
	rawToken = strings.TrimSpace(rawToken)
	if rawToken == "" {
		return "", ErrInvalidToken
	}

	row, err := v.q.GetActiveSessionByTokenHash(ctx, HashToken(rawToken))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrInvalidToken
		}
		return "", fmt.Errorf("membaca sesi: %w", err)
	}
	if !row.UserID.Valid {
		return "", ErrInvalidToken
	}
	return uuid.UUID(row.UserID.Bytes).String(), nil
}
