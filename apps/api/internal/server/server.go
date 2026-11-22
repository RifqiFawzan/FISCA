// Package server wires the HTTP router. It contains no business rules: handlers
// call services, services call repositories (plan.md §4.2).
package server

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/fisca-app/fisca/apps/api/internal/ai"
	"github.com/fisca-app/fisca/apps/api/internal/api"
	"github.com/fisca-app/fisca/apps/api/internal/auth"
	"github.com/fisca-app/fisca/apps/api/internal/config"
	"github.com/fisca-app/fisca/apps/api/internal/httpx"
	"github.com/fisca-app/fisca/apps/api/internal/infrastructure/database"
	"github.com/fisca-app/fisca/apps/api/internal/middleware"
	"github.com/fisca-app/fisca/apps/api/internal/service"
)

// Options are the dependencies required to build the router.
type Options struct {
	Config   *config.Config
	DB       *pgxpool.Pool
	Verifier auth.Verifier
	Version  string
}

// New builds the gin engine with the full middleware chain.
func New(opts Options) *gin.Engine {
	if opts.Config.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(
		middleware.RequestID(),
		middleware.StructuredLogger(),
		gin.CustomRecovery(func(c *gin.Context, _ any) {
			httpx.Fail(c, http.StatusInternalServerError, httpx.CodeInternal, "internal server error", nil)
		}),
		middleware.CORS(opts.Config.CORSOrigins),
	)

	router.GET("/health", health(opts))

	svc := service.New(opts.DB, opts.Config)
	svc.AI = ai.New(opts.Config.OpenAIAPIKey, opts.Config.OpenAIBaseURL, opts.Config.OpenAISTTBaseURL, opts.Config.OpenAIModel, opts.Config.STTModel, opts.Config.OpenAISessionID, opts.Config.AITimeout)

	v1 := router.Group("/api/v1")
	v1.Use(middleware.Auth(opts.Verifier))
	// General per-user budget for every v1 call, plus a tighter budget for AI
	// and speech routes (PRD §21.3).
	v1.Use(middleware.RateLimit(opts.Config.RateLimitRPS, opts.Config.RateLimitBurst, "v1"))
	expensiveRPS := opts.Config.RateLimitRPS / 2
	if expensiveRPS < 1 {
		expensiveRPS = 1
	}
	expensiveBurst := opts.Config.RateLimitBurst / 2
	if expensiveBurst < 2 {
		expensiveBurst = 2
	}
	v1.Use(middleware.RateLimit(expensiveRPS, expensiveBurst, "expensive", middleware.ExpensiveOnly))

	handler := api.New(svc)

	// Rute auth publik: mustahil meminta token untuk mendapatkan token, jadi
	// berada di luar middleware.Auth, dengan bucket rate limit tersendiri agar
	// percobaan masuk/daftar tidak menghabiskan anggaran v1 — dan sebaliknya.
	//
	// Bucket ini sengaja LEBIH KETAT dari RATE_LIMIT_RPS. Argon2id (64 MiB)
	// membatasi throughput login ke ~10 percobaan/detik; dengan laju yang sama,
	// konsumsi dan refill seimbang sehingga 429 tidak pernah terjadi — terukur
	// pada 10 Nov 2026: 30 percobaan login salah berturut-turut semuanya 401.
	// Laju di bawah throughput itu membuat pelindungnya benar-benar mengena,
	// sekaligus menahan spam login yang memaksa alokasi 64 MiB per percobaan.
	authRate := opts.Config.RateLimitRPS / 4
	if authRate < 2 {
		authRate = 2
	}
	authBurst := opts.Config.RateLimitBurst / 4
	if authBurst < 5 {
		authBurst = 5
	}
	authPublic := router.Group("/api/v1/auth")
	authPublic.Use(middleware.RateLimit(authRate, authBurst, "auth"))
	handler.RegisterPublic(authPublic)

	handler.Register(v1)

	return router
}

func health(opts Options) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := database.Ping(c.Request.Context(), opts.DB); err != nil {
			c.JSON(http.StatusServiceUnavailable, httpx.Envelope{
				Data: gin.H{"status": "degraded", "db": "down", "version": opts.Version, "env": opts.Config.Env},
				Meta: &httpx.Meta{RequestID: httpx.RequestID(c)},
			})
			return
		}

		httpx.OK(c, gin.H{
			"status":    "ok",
			"db":        "ok",
			"version":   opts.Version,
			"env":       opts.Config.Env,
			"checkedAt": time.Now().UTC().Format(time.RFC3339),
		})
	}
}
