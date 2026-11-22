package server_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/google/uuid"

	"github.com/fisca-app/fisca/apps/api/internal/auth"
)

// Alur auth lokal (ISSUE-15). Berbeda dari test bisnis di integration_test.go
// yang memakai stubVerifier, file ini memakai verifier sesi produksi
// (newEnvWithSessionAuth) sehingga register → login → request terautentikasi
// → logout benar-benar melewati tabel sessions.

// uniqueEmail memakai domain .test (reserved oleh RFC 6761 untuk pengujian)
// dan UUID agar tidak pernah bertabrakan dengan baris dari run sebelumnya bila
// sebuah test gagal sebelum sempat membersihkan diri.
func uniqueEmail() string {
	return fmt.Sprintf("auth-%s@example.test", uuid.New().String())
}

// registerAccount mendaftar dan menjadwalkan penghapusan akunnya. Sesi ikut
// terhapus otomatis karena foreign key-nya ON DELETE CASCADE.
func (e *testEnv) registerAccount(email, password string) response {
	e.t.Helper()
	r := e.call("", http.MethodPost, "/api/v1/auth/register",
		map[string]string{"email": email, "password": password}, "")
	if r.Status == http.StatusCreated {
		e.t.Cleanup(func() {
			_, _ = e.pool.Exec(context.Background(), `DELETE FROM users WHERE email = $1`, email)
		})
	}
	return r
}

func tokenOf(r response) string {
	token, _ := dataOf(r)["token"].(string)
	return token
}

func errorMessage(r response) string {
	body, _ := r.Body["error"].(map[string]any)
	message, _ := body["message"].(string)
	return message
}

// AC ISSUE-15 #1: daftar menerbitkan token yang benar-benar diterima API.
func TestRegisterIssuesWorkingSession(t *testing.T) {
	e := newEnvWithSessionAuth(t)
	email := uniqueEmail()

	r := e.registerAccount(email, "kata-sandi-rahasia-123")
	if r.Status != http.StatusCreated {
		t.Fatalf("register = %d: %s", r.Status, r.Raw)
	}

	token := tokenOf(r)
	if token == "" {
		t.Fatal("register returned no token")
	}
	if dataOf(r)["expiresAt"] == "" || dataOf(r)["expiresAt"] == nil {
		t.Fatal("register returned no expiresAt")
	}
	// AC: email asli tersimpan, bukan <uuid>@fisca.local.
	if got := dataOf(r)["user"].(map[string]any)["email"]; got != email {
		t.Fatalf("email = %v, want %s", got, email)
	}

	me := e.call(token, http.MethodGet, "/api/v1/me", nil, "")
	if me.Status != http.StatusOK {
		t.Fatalf("GET /me with the new token = %d: %s", me.Status, me.Raw)
	}
	if got := dataOf(me)["email"]; got != email {
		t.Fatalf("/me email = %v, want %s", got, email)
	}
}

// AC ISSUE-15 #3: pengguna baru mendapat akun & kategori bawaan sehingga
// halaman tidak kosong begitu masuk.
func TestRegisterProvisionsDefaults(t *testing.T) {
	e := newEnvWithSessionAuth(t)
	token := tokenOf(e.registerAccount(uniqueEmail(), "kata-sandi-rahasia-123"))

	accounts := e.call(token, http.MethodGet, "/api/v1/accounts", nil, "")
	if accounts.Status != http.StatusOK {
		t.Fatalf("GET /accounts = %d: %s", accounts.Status, accounts.Raw)
	}
	if len(listOf(accounts)) == 0 {
		t.Fatal("a new account should come with a default account")
	}

	categories := e.call(token, http.MethodGet, "/api/v1/categories", nil, "")
	if categories.Status != http.StatusOK {
		t.Fatalf("GET /categories = %d: %s", categories.Status, categories.Raw)
	}
	if len(listOf(categories)) == 0 {
		t.Fatal("a new account should come with default categories")
	}
}

// AC ISSUE-15 #2: email tak dikenal, akun tanpa kata sandi, dan kata sandi
// salah harus menghasilkan pesan IDENTIK — kalau tidak, login bisa dipakai
// untuk menebak apakah sebuah email terdaftar.
func TestLoginDoesNotRevealWhetherEmailExists(t *testing.T) {
	e := newEnvWithSessionAuth(t)
	email := uniqueEmail()
	if r := e.registerAccount(email, "kata-sandi-benar-123"); r.Status != http.StatusCreated {
		t.Fatalf("register = %d: %s", r.Status, r.Raw)
	}

	wrongPassword := e.call("", http.MethodPost, "/api/v1/auth/login",
		map[string]string{"email": email, "password": "kata-sandi-salah-123"}, "")
	unknownEmail := e.call("", http.MethodPost, "/api/v1/auth/login",
		map[string]string{"email": uniqueEmail(), "password": "kata-sandi-salah-123"}, "")
	noPasswordAccount := e.call("", http.MethodPost, "/api/v1/auth/login",
		map[string]string{"email": "pengguna-tanpa-sandi@example.test", "password": "kata-sandi-salah-123"}, "")

	for name, r := range map[string]response{
		"salah password":      wrongPassword,
		"email tak dikenal":   unknownEmail,
		"akun tanpa password": noPasswordAccount,
	} {
		if r.Status != http.StatusUnauthorized {
			t.Errorf("%s: status = %d, want 401", name, r.Status)
		}
	}

	base := errorMessage(wrongPassword)
	if base == "" {
		t.Fatal("expected an error message for a wrong password")
	}
	for name, r := range map[string]response{
		"email tak dikenal":   unknownEmail,
		"akun tanpa password": noPasswordAccount,
	} {
		if got := errorMessage(r); got != base {
			t.Errorf("%s: message = %q, want identical %q", name, got, base)
		}
	}

	// Kredensial yang benar tetap harus masuk.
	ok := e.call("", http.MethodPost, "/api/v1/auth/login",
		map[string]string{"email": email, "password": "kata-sandi-benar-123"}, "")
	if ok.Status != http.StatusOK {
		t.Fatalf("login with correct password = %d: %s", ok.Status, ok.Raw)
	}
	if tokenOf(ok) == "" {
		t.Fatal("login returned no token")
	}
}

func TestRegisterRejectsDuplicateEmail(t *testing.T) {
	e := newEnvWithSessionAuth(t)
	email := uniqueEmail()

	if r := e.registerAccount(email, "kata-sandi-rahasia-123"); r.Status != http.StatusCreated {
		t.Fatalf("first register = %d: %s", r.Status, r.Raw)
	}
	second := e.registerAccount(email, "kata-sandi-lain-456")
	if second.Status != http.StatusConflict {
		t.Fatalf("duplicate register = %d, want 409 (%s)", second.Status, second.Raw)
	}
}

func TestRegisterValidatesInput(t *testing.T) {
	e := newEnvWithSessionAuth(t)

	cases := []struct {
		name  string
		body  map[string]string
		field string
	}{
		{"email kosong", map[string]string{"email": "", "password": "kata-sandi-cukup-panjang"}, "email"},
		{"email tak valid", map[string]string{"email": "bukan-email", "password": "kata-sandi-cukup-panjang"}, "email"},
		{"password pendek", map[string]string{"email": uniqueEmail(), "password": "abc1234"}, "password"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := e.call("", http.MethodPost, "/api/v1/auth/register", tc.body, "")
			if r.Status != http.StatusUnprocessableEntity {
				t.Fatalf("status = %d, want 422 (%s)", r.Status, r.Raw)
			}
			details, _ := r.Body["error"].(map[string]any)["details"].(map[string]any)
			if got := details["field"]; got != tc.field {
				t.Fatalf("field = %v, want %s", got, tc.field)
			}
		})
	}
}

// AC ISSUE-15 #4: logout mencabut sesi di server, bukan sekadar menghapus
// token dari klien — token lama harus ditolak bila dipakai ulang.
func TestLogoutRevokesTokenServerSide(t *testing.T) {
	e := newEnvWithSessionAuth(t)
	token := tokenOf(e.registerAccount(uniqueEmail(), "kata-sandi-rahasia-123"))

	if r := e.call(token, http.MethodGet, "/api/v1/me", nil, ""); r.Status != http.StatusOK {
		t.Fatalf("GET /me before logout = %d: %s", r.Status, r.Raw)
	}

	out := e.call(token, http.MethodPost, "/api/v1/auth/logout", nil, "")
	if out.Status != http.StatusOK {
		t.Fatalf("logout = %d: %s", out.Status, out.Raw)
	}

	after := e.call(token, http.MethodGet, "/api/v1/me", nil, "")
	if after.Status != http.StatusUnauthorized {
		t.Fatalf("GET /me after logout = %d, want 401 (%s)", after.Status, after.Raw)
	}
}

// AC ISSUE-15 #5: sesi yang lewat masa berlaku ditolak.
func TestExpiredSessionIsRejected(t *testing.T) {
	e := newEnvWithSessionAuth(t)

	email := uniqueEmail()
	if r := e.registerAccount(email, "kata-sandi-rahasia-123"); r.Status != http.StatusCreated {
		t.Fatalf("register = %d: %s", r.Status, r.Raw)
	}

	// Sesi kadaluarsa dibuat langsung di database: alur API tidak mungkin
	// menerbitkan token yang sudah kedaluwarsa.
	raw, hash, err := auth.NewSessionToken()
	if err != nil {
		t.Fatalf("token: %v", err)
	}
	var userID string
	if err := e.pool.QueryRow(context.Background(),
		`SELECT id::text FROM users WHERE email = $1`, email).Scan(&userID); err != nil {
		t.Fatalf("user id: %v", err)
	}
	if _, err := e.pool.Exec(context.Background(),
		`INSERT INTO sessions (user_id, token_hash, expires_at)
		 VALUES ($1, $2, now() - interval '1 hour')`, userID, hash); err != nil {
		t.Fatalf("insert expired session: %v", err)
	}

	if r := e.call(raw, http.MethodGet, "/api/v1/me", nil, ""); r.Status != http.StatusUnauthorized {
		t.Fatalf("expired session = %d, want 401 (%s)", r.Status, r.Raw)
	}
}

// AC ISSUE-15 #6: jalur dev sudah hilang. Token `dev:<uuid>` yang dulu
// dipercaya apa adanya kini sama seperti token asing mana pun.
func TestLegacyDevTokenIsRejected(t *testing.T) {
	e := newEnvWithSessionAuth(t)
	legacy := "dev:" + uuid.New().String()

	if r := e.call(legacy, http.MethodGet, "/api/v1/me", nil, ""); r.Status != http.StatusUnauthorized {
		t.Fatalf("dev token = %d, want 401 (%s)", r.Status, r.Raw)
	}
}

// Logout dua kali tidak boleh mengubah respons pertama menjadi error — token
// kedua sudah tercabut sejak panggilan pertama, dan 401 memang benar.
func TestLogoutIsIdempotentForTheCaller(t *testing.T) {
	e := newEnvWithSessionAuth(t)
	token := tokenOf(e.registerAccount(uniqueEmail(), "kata-sandi-rahasia-123"))

	if r := e.call(token, http.MethodPost, "/api/v1/auth/logout", nil, ""); r.Status != http.StatusOK {
		t.Fatalf("first logout = %d: %s", r.Status, r.Raw)
	}
	if r := e.call(token, http.MethodPost, "/api/v1/auth/logout", nil, ""); r.Status != http.StatusUnauthorized {
		t.Fatalf("second logout = %d, want 401 (session already revoked)", r.Status)
	}
}
