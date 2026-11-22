package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/fisca-app/fisca/apps/api/internal/auth"
	"github.com/fisca-app/fisca/apps/api/internal/config"
	"github.com/fisca-app/fisca/apps/api/internal/infrastructure/database"
	"github.com/fisca-app/fisca/apps/api/internal/infrastructure/envfile"
	"github.com/fisca-app/fisca/apps/api/internal/server"
)

// version is overridden at build time with -ldflags "-X main.version=...".
var version = "0.1.0-dev"

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))
	envfile.Load(".env", "../../.env")

	cfg, err := config.Load()
	if err != nil {
		slog.Error("config", "error", err)
		os.Exit(1)
	}

	rootCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := database.New(rootCtx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	verifier, err := newVerifier(rootCtx, cfg, pool)
	if err != nil {
		slog.Error("auth", "error", err)
		os.Exit(1)
	}

	router := server.New(server.Options{
		Config:   cfg,
		DB:       pool,
		Verifier: verifier,
		Version:  version,
	})

	httpServer := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	go func() {
		slog.Info("api_listening", "version", version, "port", cfg.Port, "env", cfg.Env, "auth", verifier.Mode())
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("http_server", "error", err)
			os.Exit(1)
		}
	}()

	<-rootCtx.Done()
	slog.Info("shutting_down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		slog.Error("graceful_shutdown_failed", "error", err)
	}
}

// newVerifier picks the auth path from config. "session" reads the local
// sessions table; "jwt" verifies a provider's RS256 token via JWKS. Mode "dev"
// sudah dihapus pada ISSUE-15 — tidak ada lagi token yang dipercaya apa adanya.
func newVerifier(ctx context.Context, cfg *config.Config, pool *pgxpool.Pool) (auth.Verifier, error) {
	switch cfg.AuthMode {
	case config.AuthModeSession:
		return auth.NewSessionVerifier(pool), nil
	case config.AuthModeJWT:
		return auth.NewJWTVerifier(ctx, cfg.JWKSURL, cfg.JWTIssuer, cfg.JWTAudience)
	default:
		return nil, fmt.Errorf("AUTH_MODE %q tidak dikenal", cfg.AuthMode)
	}
}
