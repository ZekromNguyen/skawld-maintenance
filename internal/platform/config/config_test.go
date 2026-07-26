package config

import (
	"testing"
	"time"
)

func TestValidateRejectsPoolBudgetOverflow(t *testing.T) {
	t.Parallel()
	cfg := validConfig(RoleWorker)
	cfg.Database.APIMaxConns = 10
	cfg.Database.WorkerMaxConns = 10
	cfg.Database.MaxBudget = 19

	if err := cfg.Validate(); err == nil {
		t.Fatal("expected connection budget validation error")
	}
}

func TestAPIRequiresOIDCConfiguration(t *testing.T) {
	t.Parallel()
	cfg := validConfig(RoleAPI)
	cfg.Auth.IssuerURL = ""

	if err := cfg.Validate(); err == nil {
		t.Fatal("expected missing OIDC issuer error")
	}
}

func TestLoadRejectsMalformedEnvironmentValue(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://test")
	t.Setenv("DATABASE_API_MAX_CONNS", "not-an-integer")

	if _, err := Load(RoleWorker); err == nil {
		t.Fatal("expected malformed environment configuration error")
	}
}

func validConfig(role Role) Config {
	return Config{
		Environment: "test",
		Role:        role,
		Database: Database{
			URL:              "postgres://test",
			APIMaxConns:      5,
			WorkerMaxConns:   5,
			MaxBudget:        10,
			StatementTimeout: time.Second,
			LockTimeout:      time.Second,
		},
		Auth: Auth{
			IssuerURL:    "https://identity.example.test",
			ClientID:     "client",
			ClientSecret: "secret",
			Audience:     "api",
			RedirectURL:  "https://app.example.test/auth/callback",
			CookieName:   "session",
			SessionTTL:   1,
		},
		Jobs: Jobs{
			EmbeddingConcurrency:     1,
			ReportConcurrency:        1,
			VisionConcurrency:        1,
			TranscriptionConcurrency: 1,
		},
		ObjectStore: ObjectStore{
			Region: "us-east-1",
			Bucket: "test",
		},
	}
}
