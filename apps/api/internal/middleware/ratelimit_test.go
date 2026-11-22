package middleware_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/fisca-app/fisca/apps/api/internal/httpx"
	"github.com/fisca-app/fisca/apps/api/internal/middleware"
)

func TestLimiterBurst(t *testing.T) {
	limiter := middleware.NewLimiter(0.001, 2)
	for i := 0; i < 2; i++ {
		if !limiter.Allow("user-a") {
			t.Fatalf("request %d should be allowed within the burst", i+1)
		}
	}
	if limiter.Allow("user-a") {
		t.Fatal("third request must be rejected once the burst is exhausted")
	}
	if !limiter.Allow("user-b") {
		t.Fatal("a different identity must have its own bucket")
	}
}

func TestRateLimitReturnsEnvelope(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.RequestID())
	router.Use(middleware.RateLimit(0.001, 1, "test"))
	router.GET("/limited", func(c *gin.Context) { httpx.OK(c, gin.H{"ok": true}) })

	first := httptest.NewRecorder()
	router.ServeHTTP(first, httptest.NewRequest(http.MethodGet, "/limited", nil))
	if first.Code != http.StatusOK {
		t.Fatalf("first status = %d, want 200", first.Code)
	}

	second := httptest.NewRecorder()
	router.ServeHTTP(second, httptest.NewRequest(http.MethodGet, "/limited", nil))
	if second.Code != http.StatusTooManyRequests {
		t.Fatalf("second status = %d, want 429", second.Code)
	}
	if second.Header().Get("Retry-After") == "" {
		t.Fatal("expected Retry-After header")
	}
	var body httpx.ErrorEnvelope
	if err := json.Unmarshal(second.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.Error.Code != httpx.CodeRateLimited {
		t.Fatalf("code = %q, want %q", body.Error.Code, httpx.CodeRateLimited)
	}
}

func TestExpensiveOnlyMatchesAIAndSpeech(t *testing.T) {
	gin.SetMode(gin.TestMode)
	// FullPath is assigned by the router, so verify the matcher through real
	// routes rather than a synthetic context.
	router := gin.New()
	matched := map[string]bool{}
	for _, path := range []string{"/api/v1/assistant/messages", "/api/v1/speech/transcriptions", "/api/v1/transactions"} {
		p := path
		router.POST(p, func(c *gin.Context) {
			matched[p] = middleware.ExpensiveOnly(c)
			c.Status(http.StatusOK)
		})
	}
	for _, path := range []string{"/api/v1/assistant/messages", "/api/v1/speech/transcriptions", "/api/v1/transactions"} {
		router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, path, nil))
	}
	if !matched["/api/v1/assistant/messages"] || !matched["/api/v1/speech/transcriptions"] {
		t.Fatal("AI and speech routes must be matched")
	}
	if matched["/api/v1/transactions"] {
		t.Fatal("regular routes must not be matched")
	}
}
