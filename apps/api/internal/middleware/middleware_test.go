package middleware_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/fisca-app/fisca/apps/api/internal/auth"
	"github.com/fisca-app/fisca/apps/api/internal/httpx"
	"github.com/fisca-app/fisca/apps/api/internal/middleware"
)

const stubUserID = "11111111-1111-4111-8111-111111111111"

// stubVerifier menerima token "stub:<uuid>". Ia didefinisikan di file _test
// ini sehingga mustahil terkompilasi ke binary API — produksi tidak punya lagi
// token yang dipercaya apa adanya sejak ISSUE-15 menghapus AUTH_MODE=dev.
// Middleware diuji di sini dengan verifier sembarang, karena itu memang yang
// diuji middleware: meneruskan hasil verifikasi, bukan memverifikasinya.
type stubVerifier struct{}

func (stubVerifier) Mode() string { return "stub" }

func (stubVerifier) Verify(_ context.Context, raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if !strings.HasPrefix(raw, "stub:") {
		return "", auth.ErrInvalidToken
	}
	id := strings.TrimSpace(strings.TrimPrefix(raw, "stub:"))
	if _, err := uuid.Parse(id); err != nil {
		return "", auth.ErrInvalidToken
	}
	return id, nil
}

func newTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(middleware.RequestID())
	r.GET("/private", middleware.Auth(stubVerifier{}), func(c *gin.Context) {
		httpx.OK(c, gin.H{"userId": httpx.UserID(c)})
	})
	return r
}

func TestAuthRejectsMissingToken(t *testing.T) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/private", nil)

	newTestRouter().ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
	if got := w.Header().Get("X-Request-Id"); got == "" {
		t.Fatal("expected X-Request-Id header")
	}

	var body httpx.ErrorEnvelope
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.Error.Code != httpx.CodeUnauthorized {
		t.Fatalf("error code = %q, want %q", body.Error.Code, httpx.CodeUnauthorized)
	}
}

func TestAuthRejectsInvalidToken(t *testing.T) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/private", nil)
	req.Header.Set("Authorization", "Bearer stub:not-a-uuid")

	newTestRouter().ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
}

func TestAuthAttachesUserIDToContext(t *testing.T) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/private", nil)
	req.Header.Set("Authorization", "Bearer stub:"+stubUserID)

	newTestRouter().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body: %s)", w.Code, w.Body.String())
	}

	var body struct {
		Data struct {
			UserID string `json:"userId"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.Data.UserID != stubUserID {
		t.Fatalf("userId = %q, want %q", body.Data.UserID, stubUserID)
	}
}
