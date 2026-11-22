package auth_test

import (
	"strings"
	"testing"

	"github.com/fisca-app/fisca/apps/api/internal/auth"
)

func TestHashPasswordRoundTrip(t *testing.T) {
	const plain = "kata-sandi-rahasia-123"

	encoded, err := auth.HashPassword(plain)
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	if !strings.HasPrefix(encoded, "$argon2id$") {
		t.Fatalf("hash = %q, want argon2id PHC format", encoded)
	}

	ok, err := auth.VerifyPassword(encoded, plain)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if !ok {
		t.Fatal("expected the original password to verify")
	}
}

func TestVerifyPasswordRejectsWrongPassword(t *testing.T) {
	encoded, err := auth.HashPassword("kata-sandi-asli-123")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}

	ok, err := auth.VerifyPassword(encoded, "kata-sandi-salah-123")
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if ok {
		t.Fatal("expected a wrong password to be rejected")
	}
}

// Salt harus acak: dua hash dari kata sandi identik tidak boleh sama, kalau
// tidak hash bisa dibalik lewat rainbow table.
func TestHashPasswordIsSalted(t *testing.T) {
	const plain = "kata-sandi-sama-123"

	first, err := auth.HashPassword(plain)
	if err != nil {
		t.Fatalf("hash 1: %v", err)
	}
	second, err := auth.HashPassword(plain)
	if err != nil {
		t.Fatalf("hash 2: %v", err)
	}
	if first == second {
		t.Fatalf("identical passwords produced identical hashes: %s", first)
	}
}

func TestVerifyPasswordRejectsMalformedHash(t *testing.T) {
	malformed := []string{
		"",
		"plaintext",
		"$argon2id$",
		"$argon2i$v=19$m=65536,t=3,p=4$c2FsdA$aGFzaA",
		"$argon2id$v=99$m=65536,t=3,p=4$c2FsdA$aGFzaA",
		// Memori tak terbatas harus ditolak, bukan dialokasikan.
		"$argon2id$v=19$m=999999999,t=3,p=4$c2FsdA$aGFzaA",
	}

	for _, hash := range malformed {
		if _, err := auth.VerifyPassword(hash, "apapun"); err == nil {
			t.Errorf("hash %q should have been rejected", hash)
		}
	}
}

// AC ISSUE-15 #2: gagal masuk tidak boleh membocorkan apakah email terdaftar.
// VerifyPasswordOrDummy menjalankan derivasi yang sama walau hash kosong.
func TestVerifyPasswordOrDummyNeverAcceptsEmptyHash(t *testing.T) {
	if auth.VerifyPasswordOrDummy("", "kata-sandi-apapun-juga") {
		t.Fatal("an empty hash must never verify, regardless of the password")
	}

	encoded, err := auth.HashPassword("kata-sandi-benar-123")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	if !auth.VerifyPasswordOrDummy(encoded, "kata-sandi-benar-123") {
		t.Fatal("expected the correct password to verify")
	}
	if auth.VerifyPasswordOrDummy(encoded, "kata-sandi-salah-123") {
		t.Fatal("expected a wrong password to fail")
	}
}

func TestValidatePassword(t *testing.T) {
	cases := []struct {
		name  string
		plain string
		ok    bool
	}{
		{"terlalu pendek", "abc1234", false},
		{"tepat 8", "abc12345", true},
		{"unicode dihitung per rune", "áááááááá", true},
		{"terlalu panjang", strings.Repeat("a", 257), false},
		{"tepat 256", strings.Repeat("a", 256), true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := auth.ValidatePassword(tc.plain)
			if tc.ok && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !tc.ok && err == nil {
				t.Fatal("expected the password to be rejected")
			}
		})
	}
}
