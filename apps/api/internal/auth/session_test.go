package auth_test

import (
	"strings"
	"testing"
	"time"

	"github.com/fisca-app/fisca/apps/api/internal/auth"
)

func TestNewSessionTokenIsUnique(t *testing.T) {
	first, firstHash, err := auth.NewSessionToken()
	if err != nil {
		t.Fatalf("token 1: %v", err)
	}
	second, _, err := auth.NewSessionToken()
	if err != nil {
		t.Fatalf("token 2: %v", err)
	}

	if first == second {
		t.Fatal("two sessions produced the same token")
	}
	if firstHash == auth.HashToken(second) {
		t.Fatal("hashes of two different tokens matched")
	}
	if first == firstHash {
		t.Fatal("the raw token must not equal its stored hash")
	}
	// URL-safe agar aman di header Authorization tanpa escaping.
	if strings.ContainsAny(first, "+/=") {
		t.Fatalf("token %q is not URL-safe", first)
	}
}

// Token disimpan sebagai SHA-256, jadi hash-nya harus deterministik dan tidak
// bisa dibalik ke token aslinya.
func TestHashTokenIsDeterministicAndOpaque(t *testing.T) {
	const token = "satu-token-untuk-diuji"

	if auth.HashToken(token) != auth.HashToken(token) {
		t.Fatal("hashing the same token twice produced different digests")
	}
	digest := auth.HashToken(token)
	if len(digest) != 64 {
		t.Fatalf("digest length = %d, want 64 hex chars", len(digest))
	}
	if strings.Contains(digest, token) {
		t.Fatal("digest leaked the raw token")
	}
	if auth.HashToken("") == digest {
		t.Fatal("empty input must not hash to a non-empty digest")
	}
}

// 32 byte acak = 256 bit entropi, sehingga token tidak bisa ditebak meski
// hash-nya bocor (lihat catatan di 0003_auth.sql).
func TestNewSessionTokenHasFullEntropy(t *testing.T) {
	raw, _, err := auth.NewSessionToken()
	if err != nil {
		t.Fatalf("token: %v", err)
	}
	// base64.RawURLEncoding: 32 byte → 43 karakter tanpa padding.
	if len(raw) != 43 {
		t.Fatalf("token length = %d, want 43 (32 random bytes)", len(raw))
	}
}

// Menjaga TTL tidak pernah tak sengaja jatuh ke satuan yang salah (detik
// alih-alih hari), yang akan membuat sesi kedaluwarsa seketika.
func TestSessionTTLCoversPersonalUse(t *testing.T) {
	if auth.SessionTTL < 7*24*time.Hour {
		t.Fatalf("SessionTTL = %v, want at least 7 days", auth.SessionTTL)
	}
	if auth.SessionTTL > 365*24*time.Hour {
		t.Fatalf("SessionTTL = %v, want at most 1 year", auth.SessionTTL)
	}
}
