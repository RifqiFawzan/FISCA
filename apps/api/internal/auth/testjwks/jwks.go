// Package testjwks builds local JWKS fixtures so tests can prove the
// AUTH_MODE=jwt path end-to-end without a live Supabase project. It is only
// imported from _test files, so it never links into the API binary.
package testjwks

import (
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// KID is the key identifier shared by the served JWKS and the signed tokens.
const KID = "fisca-test-key"

type handler struct{ body []byte }

func (h *handler) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(h.body)
}

// Serve publishes pub on a local JWKS endpoint and closes it with the test.
func Serve(t *testing.T, pub *rsa.PublicKey) *httptest.Server {
	t.Helper()
	exp := []byte{0x01, 0x00, 0x01} // 65537
	if pub.E != 65537 {
		exp = []byte{byte(pub.E >> 16), byte(pub.E >> 8), byte(pub.E)}
	}
	raw, err := json.Marshal(map[string]any{"keys": []map[string]string{{
		"kty": "RSA",
		"kid": KID,
		"use": "sig",
		"alg": "RS256",
		"n":   base64.RawURLEncoding.EncodeToString(pub.N.Bytes()),
		"e":   base64.RawURLEncoding.EncodeToString(exp),
	}}})
	if err != nil {
		t.Fatalf("marshal jwks: %v", err)
	}
	server := httptest.NewServer(&handler{body: raw})
	t.Cleanup(server.Close)
	return server
}

// Options describes one token to sign.
type Options struct {
	Signer    *rsa.PrivateKey
	Method    jwt.SigningMethod
	Issuer    string
	Audience  string
	Subject   string
	ExpiresAt time.Time
	OmitExp   bool
}

// Sign produces a token using the shared kid so it matches the served JWKS.
func Sign(t *testing.T, o Options) string {
	t.Helper()
	claims := jwt.MapClaims{
		"iss": o.Issuer,
		"aud": o.Audience,
		"sub": o.Subject,
		"iat": time.Now().Add(-time.Minute).Unix(),
	}
	if !o.OmitExp {
		claims["exp"] = o.ExpiresAt.Unix()
	}
	token := jwt.NewWithClaims(o.Method, claims)
	token.Header["kid"] = KID

	var key any = o.Signer
	if _, ok := o.Method.(*jwt.SigningMethodHMAC); ok {
		// Algorithm-confusion attempt: a shared secret instead of the RSA key.
		key = []byte("not-a-real-secret")
	}
	raw, err := token.SignedString(key)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return raw
}
