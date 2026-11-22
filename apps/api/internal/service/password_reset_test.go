package service

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/fisca-app/fisca/apps/api/internal/auth"
	"github.com/fisca-app/fisca/apps/api/internal/config"
)

// Reset password lokal (cmd/resetpassword) diuji pada service, bukan lewat HTTP,
// karena alat ini memang tidak punya endpoint — pemulihan lewat akses mesin,
// bukan lewat email (keputusan ISSUE-15).
//
//	TEST_DATABASE_URL=postgres://... go test ./internal/service/ -run ResetPassword

func newResetEnv(t *testing.T) (*Service, *pgxpool.Pool) {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set TEST_DATABASE_URL to run the password reset test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(pool.Close)

	// Config kosong disengaja: ResetPassword hanya memakai pool dan query.
	return New(pool, &config.Config{}), pool
}

// deleteUsers menjadwalkan penghapusan akun uji beserta sesinya (cascade).
func deleteUsers(t *testing.T, pool *pgxpool.Pool, emails ...string) {
	t.Helper()
	t.Cleanup(func() {
		for _, email := range emails {
			_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE email = $1`, email)
		}
	})
}

// Reset harus mengganti kata sandi SEKALIGUS mematikan semua sesi lama:
// pemulihan biasanya terjadi karena akun dianggap tidak aman lagi.
func TestResetPasswordChangesPasswordAndRevokesEverySession(t *testing.T) {
	svc, pool := newResetEnv(t)
	ctx := context.Background()

	email := fmt.Sprintf("reset-%s@example.test", uuid.New())
	const oldPassword = "kata-sandi-asli-123"
	const newPassword = "kata-sandi-baru-456"
	deleteUsers(t, pool, email)

	// Registrasi membuka sesi pertama; login membuka sesi kedua.
	if _, err := svc.Register(ctx, email, oldPassword, "Uji Reset"); err != nil {
		t.Fatalf("register: %v", err)
	}
	second, err := svc.Login(ctx, email, oldPassword)
	if err != nil {
		t.Fatalf("login kedua: %v", err)
	}

	var active int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM sessions s JOIN users u ON u.id = s.user_id
		 WHERE u.email = $1 AND s.revoked_at IS NULL AND s.expires_at > now()`,
		email).Scan(&active); err != nil {
		t.Fatalf("hitung sesi: %v", err)
	}
	if active < 2 {
		t.Fatalf("persiapan gagal: sesi aktif = %d, ingin >= 2", active)
	}

	result, err := svc.ResetPassword(ctx, email, newPassword)
	if err != nil {
		t.Fatalf("reset: %v", err)
	}
	if result.User.Email != email {
		t.Fatalf("email hasil reset = %q, want %q", result.User.Email, email)
	}
	if result.RevokedSessions != int64(active) {
		t.Fatalf("sesi dicabut = %d, want %d (semua sesi aktif)", result.RevokedSessions, active)
	}

	// Token yang baru saja terbit harus ditolak VERIFIER PRODUksi — inilah
	// bukti pencabutan, bukan sekadar kolom revoked_at yang terisi.
	verifier := auth.NewSessionVerifier(pool)
	if _, err := verifier.Verify(ctx, second.Token); err == nil {
		t.Fatal("token lama masih diterima verifier setelah reset")
	}

	// Kata sandi lama ditolak dengan pesan seragam; yang baru diterima.
	if _, err := svc.Login(ctx, email, oldPassword); err == nil {
		t.Fatal("kata sandi lama masih diterima setelah reset")
	} else if err.Error() != invalidCredentials().Error() {
		t.Fatalf("pesan galat lama = %q, want %q", err.Error(), invalidCredentials().Error())
	}
	if _, err := svc.Login(ctx, email, newPassword); err != nil {
		t.Fatalf("kata sandi baru ditolak: %v", err)
	}
}

func TestResetPasswordRejectsUnknownEmailAndWeakPassword(t *testing.T) {
	svc, pool := newResetEnv(t)
	ctx := context.Background()

	// Email tidak ada — pesan harus jelas, bukan 500.
	if _, err := svc.ResetPassword(ctx, "tidak-ada@example.test", "kata-sandi-cukup-panjang"); err == nil {
		t.Fatal("expected an error for an unknown email")
	} else if err.Error() != "Email tidak ditemukan" {
		t.Fatalf("pesan = %q", err.Error())
	}

	// Sandi terlalu pendek ditolak SEBELUM menyentuh database, sehingga tidak
	// ada baris user yang tercipta.
	email := fmt.Sprintf("reset-lemah-%s@example.test", uuid.New())
	deleteUsers(t, pool, email)
	if _, err := svc.ResetPassword(ctx, email, "abc1234"); err == nil {
		t.Fatal("expected a weak password to be rejected")
	}
	var rows int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM users WHERE email = $1`, email).Scan(&rows); err != nil {
		t.Fatalf("query: %v", err)
	}
	if rows != 0 {
		t.Fatalf("user %s seharusnya tidak tercipta, tetapi ada %d baris", email, rows)
	}
}

// Huruf besar/kecil dan spasi di sekitar email tidak boleh membuat reset gagal,
// mengikuti aturan endpoint daftar.
func TestResetPasswordNormalizesEmail(t *testing.T) {
	svc, pool := newResetEnv(t)
	ctx := context.Background()

	email := fmt.Sprintf("normal-%s@example.test", uuid.New())
	deleteUsers(t, pool, email)
	if _, err := svc.Register(ctx, email, "kata-sandi-asli-123", "Uji"); err != nil {
		t.Fatalf("register: %v", err)
	}

	messy := "  " + strings.ToUpper(email) + "  "
	if _, err := svc.ResetPassword(ctx, messy, "kata-sandi-baru-456"); err != nil {
		t.Fatalf("reset dengan huruf campuran gagal: %v", err)
	}
	if _, err := svc.Login(ctx, email, "kata-sandi-baru-456"); err != nil {
		t.Fatalf("login setelah reset gagal: %v", err)
	}
}

// Hash Argon2id harus berbeda tiap reset walau kata sandinya sama — kalau tidak,
// salt-nya tidak acak dan hash bisa dibalik lewat rainbow table.
func TestResetPasswordProducesFreshHash(t *testing.T) {
	svc, pool := newResetEnv(t)
	ctx := context.Background()

	email := fmt.Sprintf("hash-%s@example.test", uuid.New())
	const password = "kata-sandi-sama-123"
	deleteUsers(t, pool, email)
	if _, err := svc.Register(ctx, email, password, "Uji"); err != nil {
		t.Fatalf("register: %v", err)
	}

	if _, err := svc.ResetPassword(ctx, email, password); err != nil {
		t.Fatalf("reset pertama: %v", err)
	}
	var first string
	if err := pool.QueryRow(ctx, `SELECT password_hash FROM users WHERE email = $1`, email).Scan(&first); err != nil {
		t.Fatalf("baca hash: %v", err)
	}

	if _, err := svc.ResetPassword(ctx, email, password); err != nil {
		t.Fatalf("reset kedua: %v", err)
	}
	var second string
	if err := pool.QueryRow(ctx, `SELECT password_hash FROM users WHERE email = $1`, email).Scan(&second); err != nil {
		t.Fatalf("baca hash: %v", err)
	}

	if first == second {
		t.Fatal("dua reset dengan kata sandi sama menghasilkan hash identik")
	}
	if ok, err := auth.VerifyPassword(second, password); err != nil || !ok {
		t.Fatalf("hash hasil reset tidak valid: ok=%v err=%v", ok, err)
	}
}

// Reset satu akun tidak boleh menyentuh sesi milik akun lain.
func TestResetPasswordDoesNotTouchOtherUsers(t *testing.T) {
	svc, pool := newResetEnv(t)
	ctx := context.Background()

	me := fmt.Sprintf("aku-%s@example.test", uuid.New())
	other := fmt.Sprintf("dia-%s@example.test", uuid.New())
	deleteUsers(t, pool, me, other)

	if _, err := svc.Register(ctx, me, "kata-sandi-asli-123", "Aku"); err != nil {
		t.Fatalf("register aku: %v", err)
	}
	if _, err := svc.Register(ctx, other, "kata-sandi-asli-123", "Dia"); err != nil {
		t.Fatalf("register dia: %v", err)
	}

	countActive := func(email string) int {
		t.Helper()
		var n int
		if err := pool.QueryRow(ctx,
			`SELECT count(*) FROM sessions s JOIN users u ON u.id = s.user_id
			 WHERE u.email = $1 AND s.revoked_at IS NULL`, email).Scan(&n); err != nil {
			t.Fatalf("hitung sesi %s: %v", email, err)
		}
		return n
	}

	before := countActive(other)
	if before == 0 {
		t.Fatal("persiapan: user lain tidak punya sesi aktif")
	}
	if _, err := svc.ResetPassword(ctx, me, "kata-sandi-baru-456"); err != nil {
		t.Fatalf("reset: %v", err)
	}
	if after := countActive(other); after != before {
		t.Fatalf("sesi user lain berubah: %d -> %d", before, after)
	}
}
