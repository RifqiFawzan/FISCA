// Package auth turns an incoming bearer token into a trusted user id.
//
// Two modes exist:
//   - "session" : local accounts (ISSUE-15). Token acak diterbitkan FISCA,
//     disimpan ter-hash di tabel sessions, dan dicabut lewat logout.
//   - "jwt"     : RS256 JWTs signed by an auth provider (Supabase), verified
//     against a JWKS endpoint with issuer/audience checks. Jalur ini dipertahankan
//     sebagai cadangan bila FISCA kelak menjadi proyek SaaS; tidak dipakai saat ini.
//
// Mode "dev" (`dev:<uuid>` yang dipercaya apa adanya) dihapus pada ISSUE-15:
// satu jalur auth saja, agar seluruh suite melewatinya dan tidak ada token
// backdoor yang terpampang di UI.
package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// ErrInvalidToken is returned for any token that cannot be trusted.
var ErrInvalidToken = errors.New("invalid token")

// Verifier validates a raw bearer token and returns the owning user id.
type Verifier interface {
	Verify(ctx context.Context, rawToken string) (userID string, err error)
	Mode() string
}

// NewJWTVerifier builds an RS256/JWKS verifier. The JWKS document is fetched
// lazily and cached in memory by keyfunc. Jalur ini dormant sampai
// AUTH_MODE=jwt dipakai kembali (Supabase).
func NewJWTVerifier(ctx context.Context, jwksURL, issuer, audience string) (Verifier, error) {
	if jwksURL == "" {
		return nil, fmt.Errorf("jwks url is required")
	}
	keyfuncs, err := keyfunc.NewDefaultCtx(ctx, []string{jwksURL})
	if err != nil {
		return nil, fmt.Errorf("load jwks: %w", err)
	}
	return &jwtVerifier{keyfuncs: keyfuncs, issuer: issuer, audience: audience}, nil
}

type jwtVerifier struct {
	keyfuncs keyfunc.Keyfunc
	issuer   string
	audience string
}

func (*jwtVerifier) Mode() string { return "jwt" }

func (v *jwtVerifier) Verify(_ context.Context, rawToken string) (string, error) {
	claims := jwt.MapClaims{}
	options := []jwt.ParserOption{
		jwt.WithValidMethods([]string{jwt.SigningMethodRS256.Alg()}),
		jwt.WithExpirationRequired(),
	}
	if v.issuer != "" {
		options = append(options, jwt.WithIssuer(v.issuer))
	}
	if v.audience != "" {
		options = append(options, jwt.WithAudience(v.audience))
	}

	if _, err := jwt.ParseWithClaims(rawToken, claims, v.keyfuncs.Keyfunc, options...); err != nil {
		return "", ErrInvalidToken
	}

	subject, err := claims.GetSubject()
	if err != nil || subject == "" {
		return "", ErrInvalidToken
	}
	if _, err := uuid.Parse(subject); err != nil {
		return "", ErrInvalidToken
	}
	return subject, nil
}
