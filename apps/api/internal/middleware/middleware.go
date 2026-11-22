// Package middleware contains the gin middleware chain shared by every route.
package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/fisca-app/fisca/apps/api/internal/auth"
	"github.com/fisca-app/fisca/apps/api/internal/httpx"
)

const requestIDHeader = "X-Request-Id"

// RequestID attaches a request id to the context and echoes it in the response.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := strings.TrimSpace(c.GetHeader(requestIDHeader))
		if id == "" || len(id) > 128 {
			id = newRequestID()
		}
		c.Set(httpx.RequestIDKey, id)
		c.Header(requestIDHeader, id)
		c.Next()
	}
}

func newRequestID() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "unknown"
	}
	return hex.EncodeToString(buf)
}

// CORS allows the configured browser origins. Requests from unknown origins do
// not receive CORS headers, which makes the browser block them.
func CORS(allowedOrigins []string) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(allowedOrigins))
	for _, o := range allowedOrigins {
		allowed[strings.TrimSpace(o)] = struct{}{}
	}

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin != "" {
			if _, ok := allowed[origin]; ok {
				c.Header("Access-Control-Allow-Origin", origin)
				c.Header("Vary", "Origin")
				c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type, Idempotency-Key, X-Request-Id")
				c.Header("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
				c.Header("Access-Control-Max-Age", "600")
			}
		}
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

// Auth requires a valid bearer token and stores the user id in the context.
func Auth(verifier auth.Verifier) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw, ok := bearerToken(c.GetHeader("Authorization"))
		if !ok {
			httpx.Unauthorized(c, "missing bearer token")
			return
		}

		userID, err := verifier.Verify(c.Request.Context(), raw)
		if err != nil {
			httpx.Unauthorized(c, "invalid or expired token")
			return
		}

		c.Set(httpx.UserIDKey, userID)
		// Disimpan juga token mentahnya agar logout bisa mencabut sesi ini.
		c.Set(httpx.TokenKey, raw)
		c.Next()
	}
}

func bearerToken(header string) (string, bool) {
	parts := strings.SplitN(strings.TrimSpace(header), " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
		return "", false
	}
	token := strings.TrimSpace(parts[1])
	if token == "" {
		return "", false
	}
	return token, true
}
