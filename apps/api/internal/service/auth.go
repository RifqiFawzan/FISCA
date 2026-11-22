package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/mail"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/fisca-app/fisca/apps/api/internal/auth"
	"github.com/fisca-app/fisca/apps/api/internal/infrastructure/database/generated"
)

// AuthResult adalah hasil login/registrasi yang sukses. Token hanya dikirim
// sekali di respons ini — server menyimpan hash-nya saja.
type AuthResult struct {
	Token     string
	ExpiresAt time.Time
	User      UserDTO
}

const maxEmailLength = 254

// invalidCredentials adalah pesan seragam untuk email yang tidak terdaftar,
// akun tanpa kata sandi, dan kata sandi yang salah. AC ISSUE-15 #2: respons
// tidak boleh membedakan kasus mana yang terjadi, karena perbedaan itulah yang
// membuat login bisa dipakai untuk memvalidasi keberadaan akun.
func invalidCredentials() *Error {
	return &Error{
		Status:  http.StatusUnauthorized,
		Code:    CodeUnauthorized,
		Message: "Email atau kata sandi salah",
	}
}

// Register membuat akun lokal beserta sesi pertamanya.
//
// Akun, default awal (akun & kategori), dan sesi dibuat dalam satu transaksi:
// bila salah satu gagal, tidak boleh ada akun yang tercipta tanpa sesi, dan
// tidak boleh ada sesi yang menunjuk akun setengah jadi.
func (s *Service) Register(ctx context.Context, email, password, displayName string) (AuthResult, error) {
	email, err := normalizeEmail(email)
	if err != nil {
		return AuthResult{}, err
	}
	if err := auth.ValidatePassword(password); err != nil {
		return AuthResult{}, Invalid("password", err.Error())
	}

	displayName = strings.TrimSpace(displayName)
	if displayName == "" {
		displayName = displayNameFromEmail(email)
	}
	if len([]rune(displayName)) > 60 {
		return AuthResult{}, Invalid("displayName", "Nama tampilan maksimal 60 karakter")
	}

	// Hash dan token dibuat di luar transaksi: Argon2id sengaja lambat dan
	// tidak boleh menahan kunci baris di database.
	passwordHash, err := auth.HashPassword(password)
	if err != nil {
		return AuthResult{}, fmt.Errorf("hashing password: %w", err)
	}
	rawToken, tokenHash, err := auth.NewSessionToken()
	if err != nil {
		return AuthResult{}, err
	}
	expiresAt := s.NowUTC().Add(auth.SessionTTL)
	userID := uuid.New()

	var created generated.User
	if err := s.WithTx(ctx, func(q *generated.Queries) error {		user, err := q.CreateUserWithPassword(ctx, generated.CreateUserWithPasswordParams{
			ID:           pgUUIDValue(userID),
			Email:        email,
			DisplayName:  displayName,
			PasswordHash: &passwordHash,
		})
		if err != nil {
			return err
		}
		created = user

		// Tanpa ini, pengguna baru tidak punya akun kas maupun kategori bawaan
		// — EnsureUser hanya menjalankannya saat Ia yang pertama membuat baris.
		if err := s.EnsureDefaults(ctx, q, userID); err != nil {
			return err
		}
		_, err = q.CreateSession(ctx, generated.CreateSessionParams{
			UserID:    pgUUIDValue(userID),
			TokenHash: tokenHash,
			ExpiresAt: pgtype.Timestamptz{Time: expiresAt, Valid: true},
		})
		return err
	}); err != nil {
		return AuthResult{}, mapDBError(err)
	}

	s.pruneSessions(ctx)
	return AuthResult{Token: rawToken, ExpiresAt: expiresAt, User: userDTO(created)}, nil
}

// Login memverifikasi kredensial dan membuka sesi baru.
func (s *Service) Login(ctx context.Context, email, password string) (AuthResult, error) {
	email = strings.TrimSpace(email)
	if email == "" || password == "" {
		return AuthResult{}, Invalid("email", "Email dan kata sandi wajib diisi")
	}

	user, err := s.Q.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Biaya Argon2id tetap dijalankan supaya waktu respons sama dengan
			// akun yang ada (lihat invalidCredentials).
			auth.VerifyPasswordOrDummy("", password)
			return AuthResult{}, invalidCredentials()
		}
		return AuthResult{}, mapDBError(err)
	}

	var stored string
	if user.PasswordHash != nil {
		stored = *user.PasswordHash
	}
	if !auth.VerifyPasswordOrDummy(stored, password) {
		return AuthResult{}, invalidCredentials()
	}

	rawToken, tokenHash, err := auth.NewSessionToken()
	if err != nil {
		return AuthResult{}, err
	}
	expiresAt := s.NowUTC().Add(auth.SessionTTL)
	if _, err := s.Q.CreateSession(ctx, generated.CreateSessionParams{
		UserID:    user.ID,
		TokenHash: tokenHash,
		ExpiresAt: pgtype.Timestamptz{Time: expiresAt, Valid: true},
	}); err != nil {
		return AuthResult{}, mapDBError(err)
	}

	s.pruneSessions(ctx)
	return AuthResult{Token: rawToken, ExpiresAt: expiresAt, User: userDTO(user)}, nil
}

// Logout mencabut sesi token yang sedang dipakai. Setelah ini token yang sama
// ditolak verifier, walau belum lewat masa berlakunya (AC ISSUE-15 #4).
func (s *Service) Logout(ctx context.Context, rawToken string) error {
	if strings.TrimSpace(rawToken) == "" {
		return Invalid("token", "Token sesi tidak ditemukan")
	}
	if _, err := s.Q.RevokeSessionByTokenHash(ctx, auth.HashToken(rawToken)); err != nil {
		return mapDBError(err)
	}
	return nil
}

// pruneSessions membuang sesi kedaluwarsa. Dijalankan pada login/registrasi,
// bukan tiap request — penghapusan berkala cukup untuk tabel yang tumbuh per
// kali pengguna masuk. Galat diabaikan: ini pembersihan, bukan alur utama.
func (s *Service) pruneSessions(ctx context.Context) {
	_, _ = s.Q.DeleteExpiredSessions(ctx)
}

// PasswordResetResult adalah hasil reset: akun yang berubah dan jumlah sesi
// yang dicabut, supaya pemanggil bisa melaporkan dampaknya.
type PasswordResetResult struct {
	User            UserDTO
	RevokedSessions int64
}

// ResetPassword mengganti kata sandi sebuah akun lalu mencabut SELURUH sesinya.
//
// Ini jalur pemulihan lokal (cmd/resetpassword). FISCA sengaja tidak punya
// reset lewat email karena dipakai untuk pemakaian pribadi tanpa SMTP
// (keputusan ISSUE-15), sehingga alat ini dijalankan dari mesin yang memang
// punya akses database.
//
// Semua sesi dicabut — bukan hanya yang sedang dipakai — karena reset biasanya
// terjadi ketika akun dianggap tidak aman lagi.
func (s *Service) ResetPassword(ctx context.Context, email, newPassword string) (PasswordResetResult, error) {
	email, err := normalizeEmail(email)
	if err != nil {
		return PasswordResetResult{}, err
	}
	if err := auth.ValidatePassword(newPassword); err != nil {
		return PasswordResetResult{}, Invalid("password", err.Error())
	}

	// Argon2id sengaja lambat, jadi dihitung di luar transaksi agar tidak
	// menahan kunci baris — sama seperti Register.
	passwordHash, err := auth.HashPassword(newPassword)
	if err != nil {
		return PasswordResetResult{}, fmt.Errorf("hashing password: %w", err)
	}

	var result PasswordResetResult
	if err := s.WithTx(ctx, func(q *generated.Queries) error {
		user, err := q.UpdateUserPassword(ctx, generated.UpdateUserPasswordParams{
			PasswordHash: &passwordHash,
			Email:        email,
		})
		if err != nil {
			return err
		}
		revoked, err := q.RevokeAllSessionsForUser(ctx, user.ID)
		if err != nil {
			return err
		}
		result = PasswordResetResult{User: userDTO(user), RevokedSessions: revoked}
		return nil
	}); err != nil {
		// Email tidak ada: WithTx meneruskan ErrNoRows apa adanya.
		if errors.Is(err, pgx.ErrNoRows) {
			return PasswordResetResult{}, NotFound("Email")
		}
		return PasswordResetResult{}, mapDBError(err)
	}

	s.pruneSessions(ctx)
	return result, nil
}

// normalizeEmail menstandarkan email dan menolak format yang tidak sah.
// users.email bertipe CITEXT, jadi unik sudah case-insensitive di database.
func normalizeEmail(raw string) (string, error) {
	email := strings.ToLower(strings.TrimSpace(raw))
	if email == "" {
		return "", Invalid("email", "Email wajib diisi")
	}
	if len(email) > maxEmailLength {
		return "", Invalid("email", "Email maksimal 254 karakter")
	}
	addr, err := mail.ParseAddress(email)
	// addr.Address harus persis input: menolak "Nama <email@x>" dan display name.
	if err != nil || !strings.EqualFold(addr.Address, email) {
		return "", Invalid("email", "Format email tidak valid")
	}
	return addr.Address, nil
}

// displayNameFromEmail mengambil bagian sebelum "@" sebagai nama tampilan
// awal. Pengguna bisa mengubahnya lewat profil.
func displayNameFromEmail(email string) string {
	local, _, _ := strings.Cut(email, "@")
	local = strings.TrimSpace(local)
	if local == "" {
		return "Pengguna FISCA"
	}
	runes := []rune(local)
	if len(runes) > 60 {
		runes = runes[:60]
	}
	return string(runes)
}
