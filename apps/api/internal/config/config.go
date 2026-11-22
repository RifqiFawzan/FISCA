package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Env values.
const (
	EnvDevelopment = "development"
	EnvTest        = "test"
	EnvProduction  = "production"
)

// Auth modes.
//
// "session" adalah jalur lokal (ISSUE-15) dan menjadi default. "jwt" dipertahankan
// sebagai jalur Supabase yang dorman. Mode "dev" dihapus: tidak ada lagi token
// yang dipercaya apa adanya.
const (
	AuthModeSession = "session"
	AuthModeJWT     = "jwt"
)

// Config holds every runtime setting the API needs. Values are read from the
// environment; nothing is hardcoded so that deployment stays explicit.
type Config struct {
	Env         string
	Port        string
	DatabaseURL string

	AuthMode     string
	JWKSURL      string
	JWTIssuer    string
	JWTAudience  string
	CORSOrigins  []string
	RateLimitRPS float64

	// BudgetWarningThreshold is the fraction of a budget limit that triggers a
	// warning. Locked in plan.md (O-12) as 0.80; configurable, not hardcoded in UI.
	BudgetWarningThreshold float64

	// AI gateway (plan.md §9). When the key is empty the assistant falls back to
	// deterministic Indonesian heuristics instead of failing.
	OpenAIAPIKey  string
	OpenAIBaseURL string
	OpenAIModel   string
	STTModel      string
	// OpenAISTTBaseURL menargetkan transkripsi ke layanan terpisah (mis. speaches
	// lokal). Kosong berarti memakai OpenAIBaseURL, jadi konfigurasi lama tetap
	// berlaku tanpa perubahan.
	OpenAISTTBaseURL string
	AITimeout        time.Duration
	RateLimitBurst   int
}

// IsProduction reports whether the API runs with production safety rules.
func (c *Config) IsProduction() bool {
	return c.Env == EnvProduction
}

// Load reads configuration from the environment and validates it.
func Load() (*Config, error) {
	cfg := &Config{
		Env:                    env("APP_ENV", EnvDevelopment),
		Port:                   env("API_PORT", "8080"),
		DatabaseURL:            os.Getenv("DATABASE_URL"),
		AuthMode:               env("AUTH_MODE", AuthModeSession),
		JWKSURL:                os.Getenv("AUTH_JWKS_URL"),
		JWTIssuer:              os.Getenv("AUTH_ISSUER"),
		JWTAudience:            os.Getenv("AUTH_AUDIENCE"),
		CORSOrigins:            splitCSV(env("CORS_ORIGINS", "http://localhost:3000")),
		RateLimitRPS:           parseFloat(env("RATE_LIMIT_RPS", "10"), 10),
		BudgetWarningThreshold: parseFloat(env("BUDGET_WARNING_THRESHOLD", "0.8"), 0.8),
		OpenAIAPIKey:           os.Getenv("OPENAI_API_KEY"),
		OpenAIBaseURL:          env("OPENAI_BASE_URL", "https://api.openai.com/v1"),
		OpenAIModel:            env("OPENAI_MODEL", "gpt-4o-mini"),
		STTModel:               env("OPENAI_STT_MODEL", "whisper-1"),
		OpenAISTTBaseURL:       strings.TrimRight(os.Getenv("OPENAI_STT_BASE_URL"), "/"),
		AITimeout:              time.Duration(parseFloat(env("AI_TIMEOUT_SECONDS", "20"), 20) * float64(time.Second)),
		RateLimitBurst:         int(parseFloat(env("RATE_LIMIT_BURST", "20"), 20)),
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	if cfg.Port == "" {
		return nil, fmt.Errorf("API_PORT must not be empty")
	}
	if cfg.BudgetWarningThreshold <= 0 || cfg.BudgetWarningThreshold > 1 {
		return nil, fmt.Errorf("BUDGET_WARNING_THRESHOLD must be in (0, 1], got %v", cfg.BudgetWarningThreshold)
	}

	switch cfg.AuthMode {
	case AuthModeSession:
		// Jalur lokal selalu boleh: verifikasinya membaca tabel sessions.
	case AuthModeJWT:
		if cfg.JWKSURL == "" {
			return nil, fmt.Errorf("AUTH_JWKS_URL is required when AUTH_MODE=jwt")
		}
	default:
		return nil, fmt.Errorf("AUTH_MODE must be %q or %q, got %q", AuthModeSession, AuthModeJWT, cfg.AuthMode)
	}

	return cfg, nil
}

func env(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func splitCSV(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if v := strings.TrimSpace(p); v != "" {
			out = append(out, v)
		}
	}
	return out
}

func parseFloat(raw string, fallback float64) float64 {
	v, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
	if err != nil {
		return fallback
	}
	return v
}
