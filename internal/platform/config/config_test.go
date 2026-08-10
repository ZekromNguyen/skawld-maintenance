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
		AI: AI{
			StructuredProvider: "deterministic",
			EmbeddingProvider:  "deterministic",
		},
	}
}

func TestValidateRejectsUnknownStructuredProvider(t *testing.T) {
	t.Parallel()
	cfg := validConfig(RoleAPI)
	cfg.AI.StructuredProvider = "bogus"
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected structured provider validation error")
	}
}

func TestValidateRejectsUnknownEmbeddingProvider(t *testing.T) {
	t.Parallel()
	cfg := validConfig(RoleAPI)
	cfg.AI.EmbeddingProvider = "bogus"
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected embedding provider validation error")
	}
}

func TestValidateRequiresOpenAIModel(t *testing.T) {
	t.Parallel()
	cfg := validConfig(RoleAPI)
	cfg.AI.StructuredProvider = "openai"
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected missing AI_MODEL error")
	}
}

func TestValidateRequiresAnthropicKeyAndModel(t *testing.T) {
	t.Parallel()
	cfg := validConfig(RoleAPI)
	cfg.AI.StructuredProvider = "anthropic"
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected missing ANTHROPIC_API_KEY error")
	}
}

func TestLoadDefaultsStructuredProviderToDeterministic(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://test")
	t.Setenv("S3_BUCKET", "test-bucket")
	cfg, err := Load(RoleWorker)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.AI.StructuredProvider != "deterministic" {
		t.Fatalf("StructuredProvider = %q, want deterministic", cfg.AI.StructuredProvider)
	}
	if cfg.AI.EmbeddingProvider != "deterministic" {
		t.Fatalf("EmbeddingProvider = %q, want deterministic", cfg.AI.EmbeddingProvider)
	}
}

func TestValidateRequiresOpenAIAPIKey(t *testing.T) {
	t.Parallel()
	cfg := validConfig(RoleAPI)
	cfg.AI.StructuredProvider = "openai"
	cfg.AI.Model = "gpt-4o"
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected missing AI_API_KEY error")
	}
}

func TestValidConfigPassesValidation(t *testing.T) {
	t.Parallel()
	if err := validConfig(RoleAPI).Validate(); err != nil {
		t.Fatalf("validConfig(RoleAPI) must validate clean: %v", err)
	}
}
