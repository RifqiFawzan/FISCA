package auth_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/fisca-app/fisca/apps/api/internal/auth"
	"github.com/fisca-app/fisca/apps/api/internal/auth/testjwks"
)

const (
	testIssuer = "https://example.supabase.co/auth/v1"
	testAud    = "authenticated"
)

func newJWTVerifier(t *testing.T, jwksURL string) auth.Verifier {
	t.Helper()
	verifier, err := auth.NewJWTVerifier(context.Background(), jwksURL, testIssuer, testAud)
	if err != nil {
		t.Fatalf("new jwt verifier: %v", err)
	}
	return verifier
}

// Happy path: RS256 token signed by the key published in JWKS, matching
// issuer/audience, UUID subject, unexpired.
func TestJWTVerifierAcceptsSupabaseShapedToken(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	server := testjwks.Serve(t, &priv.PublicKey)
	verifier := newJWTVerifier(t, server.URL)

	raw := testjwks.Sign(t, testjwks.Options{
		Signer:    priv,
		Method:    jwt.SigningMethodRS256,
		Issuer:    testIssuer,
		Audience:  testAud,
		Subject:   "11111111-1111-4111-8111-111111111111",
		ExpiresAt: time.Now().Add(time.Hour),
	})

	userID, err := verifier.Verify(context.Background(), raw)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if userID != "11111111-1111-4111-8111-111111111111" {
		t.Fatalf("subject = %q", userID)
	}
}

// Every path that would otherwise turn a forged token into a user id.
func TestJWTVerifierRejectsUntrustedTokens(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	other, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate other key: %v", err)
	}
	server := testjwks.Serve(t, &priv.PublicKey)
	verifier := newJWTVerifier(t, server.URL)
	validSub := "11111111-1111-4111-8111-111111111111"
	hourly := time.Now().Add(time.Hour)

	cases := []struct {
		name  string
		token string
	}{
		{"expired", testjwks.Sign(t, testjwks.Options{
			Signer: priv, Method: jwt.SigningMethodRS256, Issuer: testIssuer,
			Audience: testAud, Subject: validSub, ExpiresAt: time.Now().Add(-time.Hour),
		})},
		{"missing exp", testjwks.Sign(t, testjwks.Options{
			Signer: priv, Method: jwt.SigningMethodRS256, Issuer: testIssuer,
			Audience: testAud, Subject: validSub, OmitExp: true,
		})},
		{"wrong issuer", testjwks.Sign(t, testjwks.Options{
			Signer: priv, Method: jwt.SigningMethodRS256, Issuer: "https://evil.example",
			Audience: testAud, Subject: validSub, ExpiresAt: hourly,
		})},
		{"wrong audience", testjwks.Sign(t, testjwks.Options{
			Signer: priv, Method: jwt.SigningMethodRS256, Issuer: testIssuer,
			Audience: "admin", Subject: validSub, ExpiresAt: hourly,
		})},
		{"signed by unknown key", testjwks.Sign(t, testjwks.Options{
			Signer: other, Method: jwt.SigningMethodRS256, Issuer: testIssuer,
			Audience: testAud, Subject: validSub, ExpiresAt: hourly,
		})},
		{"alg confusion HS256", testjwks.Sign(t, testjwks.Options{
			Method: jwt.SigningMethodHS256, Issuer: testIssuer, Audience: testAud,
			Subject: validSub, ExpiresAt: hourly,
		})},
		{"non-uuid subject", testjwks.Sign(t, testjwks.Options{
			Signer: priv, Method: jwt.SigningMethodRS256, Issuer: testIssuer,
			Audience: testAud, Subject: "1; DROP TABLE users", ExpiresAt: hourly,
		})},
		{"garbage", "not.a.jwt"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := verifier.Verify(context.Background(), tc.token); err == nil {
				t.Fatal("token was accepted, want rejection")
			}
		})
	}
}
