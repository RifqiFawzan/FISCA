package config_test

import (
	"testing"

	"github.com/fisca-app/fisca/apps/api/internal/config"
)

const testDatabaseURL = "postgres://fisca:fisca@localhost:5434/fisca?sslmode=disable"

func TestLoadDefaultsToDevelopment(t *testing.T) {
	t.Setenv("APP_ENV", "")
	t.Setenv("DATABASE_URL", testDatabaseURL)
	t.Setenv("AUTH_MODE", "")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Env != config.EnvDevelopment {
		t.Fatalf("env = %q, want %q", cfg.Env, config.EnvDevelopment)
	}
	// Jalur lokal adalah default sejak ISSUE-15 menghapus mode dev.
	if cfg.AuthMode != config.AuthModeSession {
		t.Fatalf("auth mode = %q, want %q", cfg.AuthMode, config.AuthModeSession)
	}
}

func TestLoadRequiresDatabaseURL(t *testing.T) {
	t.Setenv("APP_ENV", config.EnvDevelopment)
	t.Setenv("AUTH_MODE", config.AuthModeSession)
	t.Setenv("DATABASE_URL", "")

	if _, err := config.Load(); err == nil {
		t.Fatal("expected an error when DATABASE_URL is empty")
	}
}

// Mode dev sudah dihapus (ISSUE-15): nilai yang tidak dikenal — termasuk "dev" —
// harus ditolak, bukan diam-diam jatuh ke jalur lain.
func TestLoadRejectsUnknownAuthMode(t *testing.T) {
	for _, mode := range []string{"dev", "local", ""} {
		mode := mode
		t.Run(mode, func(t *testing.T) {
			t.Setenv("APP_ENV", config.EnvDevelopment)
			t.Setenv("AUTH_MODE", mode)
			t.Setenv("DATABASE_URL", testDatabaseURL)
			// "" kembali ke default (session) dan sah; hanya nilai eksplisit
			// yang tak dikenal yang harus ditolak.
			if mode == "" {
				if _, err := config.Load(); err != nil {
					t.Fatalf("empty AUTH_MODE falls back to the default, got %v", err)
				}
				return
			}
			if _, err := config.Load(); err == nil {
				t.Fatalf("AUTH_MODE=%q should have been rejected", mode)
			}
		})
	}
}

func TestLoadRequiresJWKSWhenJWTMode(t *testing.T) {
	t.Setenv("APP_ENV", config.EnvDevelopment)
	t.Setenv("AUTH_MODE", config.AuthModeJWT)
	t.Setenv("DATABASE_URL", testDatabaseURL)
	t.Setenv("AUTH_JWKS_URL", "")

	if _, err := config.Load(); err == nil {
		t.Fatal("expected AUTH_JWKS_URL to be required when AUTH_MODE=jwt")
	}
}

func TestLoadRejectsInvalidBudgetThreshold(t *testing.T) {
	t.Setenv("APP_ENV", config.EnvDevelopment)
	t.Setenv("AUTH_MODE", config.AuthModeSession)
	t.Setenv("DATABASE_URL", testDatabaseURL)
	t.Setenv("BUDGET_WARNING_THRESHOLD", "1.5")

	if _, err := config.Load(); err == nil {
		t.Fatal("expected an out-of-range threshold to be rejected")
	}
}
