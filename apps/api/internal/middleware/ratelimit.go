package middleware

import (
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/fisca-app/fisca/apps/api/internal/httpx"
)

// bucket is a token bucket. Tokens refill continuously at rate tokens/second
// and never exceed burst.
type bucket struct {
	tokens float64
	last   time.Time
}

// Limiter is an in-process, per-key token-bucket limiter. It is deliberately
// simple and has no external dependency: one API process serves one limiter.
// For a multi-replica deployment this must be replaced by a shared store.
type Limiter struct {
	mu      sync.Mutex
	rate    float64
	burst   float64
	buckets map[string]*bucket
	lastGC  time.Time
}

func NewLimiter(rate float64, burst int) *Limiter {
	if rate <= 0 {
		rate = 10
	}
	if burst <= 0 {
		burst = int(rate)
		if burst < 1 {
			burst = 1
		}
	}
	return &Limiter{
		rate:    rate,
		burst:   float64(burst),
		buckets: make(map[string]*bucket),
		lastGC:  time.Now(),
	}
}

func (l *Limiter) Allow(key string) bool {
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()

	b, ok := l.buckets[key]
	if !ok {
		b = &bucket{tokens: l.burst, last: now}
		l.buckets[key] = b
	}

	elapsed := now.Sub(b.last).Seconds()
	if elapsed > 0 {
		b.tokens += elapsed * l.rate
		if b.tokens > l.burst {
			b.tokens = l.burst
		}
	}
	b.last = now

	if now.Sub(l.lastGC) > 5*time.Minute {
		for k, v := range l.buckets {
			if now.Sub(v.last) > 10*time.Minute {
				delete(l.buckets, k)
			}
		}
		l.lastGC = now
	}

	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

// RateLimit applies a per-identity token bucket. Identity is the authenticated
// user id when available, otherwise the client IP, so one noisy user cannot
// starve everybody else. The optional matcher limits only selected requests,
// which is how AI/speech routes get a stricter budget (PRD §21.3).
func RateLimit(rate float64, burst int, prefix string, match ...func(*gin.Context) bool) gin.HandlerFunc {
	limiter := NewLimiter(rate, burst)
	return func(c *gin.Context) {
		if len(match) > 0 && match[0] != nil && !match[0](c) {
			c.Next()
			return
		}

		identity := httpx.UserID(c)
		if identity == "" {
			identity = c.ClientIP()
		}
		if !limiter.Allow(prefix + "|" + identity) {
			c.Header("Retry-After", "1")
			httpx.Fail(c, http.StatusTooManyRequests, httpx.CodeRateLimited,
				"Terlalu banyak permintaan, coba lagi sebentar", nil)
			return
		}
		c.Next()
	}
}

// ExpensiveOnly matches AI and speech endpoints, which cost money or CPU.
func ExpensiveOnly(c *gin.Context) bool {
	path := c.FullPath()
	return path == "/api/v1/assistant/messages" ||
		path == "/api/v1/speech/transcriptions"
}

// StructuredLogger emits one JSON line per request with the correlation id,
// method, route, status, duration and payload size. Business audit trails stay
// in the audit_logs table; this is operational logging only (PRD §21.4).
func StructuredLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		status := c.Writer.Status()
		attrs := []any{
			"requestId", httpx.RequestID(c),
			"method", c.Request.Method,
			"route", c.FullPath(),
			"path", c.Request.URL.Path,
			"status", status,
			"latencyMs", time.Since(start).Milliseconds(),
			"bytes", c.Writer.Size(),
			"clientIp", c.ClientIP(),
		}
		if userID := httpx.UserID(c); userID != "" {
			attrs = append(attrs, "userId", userID)
		}
		if len(c.Errors) > 0 {
			attrs = append(attrs, "errors", c.Errors.String())
		}

		switch {
		case status >= http.StatusInternalServerError:
			slog.Error("http_request", attrs...)
		case status >= http.StatusBadRequest:
			slog.Warn("http_request", attrs...)
		default:
			slog.Info("http_request", attrs...)
		}
	}
}
