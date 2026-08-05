package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Role string

const (
	RoleAPI    Role = "api"
	RoleWorker Role = "worker"
)

type Config struct {
	Environment   string
	Role          Role
	HTTP          HTTP
	Database      Database
	Auth          Auth
	Jobs          Jobs
	ObjectStore   ObjectStore
	Documents     Documents
	Transcription Transcription
}

type HTTP struct {
	Address string
}

type Database struct {
	URL              string
	APIMaxConns      int32
	WorkerMaxConns   int32
	MaxBudget        int32
	StatementTimeout time.Duration
	LockTimeout      time.Duration
}

type Auth struct {
	IssuerURL         string
	ClientID          string
	ClientSecret      string
	Audience          string
	RedirectURL       string
	CookieName        string
	CookieSecure      bool
	SessionTTL        time.Duration
	BootstrapSubjects []string
	// EmailDomainAllowlist restricts federated (Google / Microsoft Entra)
	// sign-in to company accounts whose email domain is listed. Empty means
	// the gate is disabled. Bootstrap and seeded principals are exempt.
	EmailDomainAllowlist []string
	// FederatedOrgID is the organization federated users are provisioned
	// into when they carry a valid skawld_role claim.
	FederatedOrgID string
}

type Jobs struct {
	EmbeddingConcurrency     int
	ReportConcurrency        int
	VisionConcurrency        int
	TranscriptionConcurrency int
}

type ObjectStore struct {
	Endpoint        string
	Region          string
	Bucket          string
	AccessKeyID     string
	SecretAccessKey string
	UsePathStyle    bool
}

type Documents struct {
	PDFToTextBinary string
}

type Transcription struct {
	Endpoint     string
	APIKey       string
	Provider     string
	Model        string
	ModelVersion string
}

func Load(role Role) (Config, error) {
	cfg := Config{
		Environment: env("APP_ENV", "development"),
		Role:        role,
		HTTP: HTTP{
			Address: env("HTTP_ADDRESS", ":8080"),
		},
		Database: Database{
			URL:              env("DATABASE_URL", ""),
			APIMaxConns:      int32(envInt("DATABASE_API_MAX_CONNS", 10)),
			WorkerMaxConns:   int32(envInt("DATABASE_WORKER_MAX_CONNS", 10)),
			MaxBudget:        int32(envInt("DATABASE_MAX_CONNECTION_BUDGET", 25)),
			StatementTimeout: envDuration("DATABASE_STATEMENT_TIMEOUT", 15*time.Second),
			LockTimeout:      envDuration("DATABASE_LOCK_TIMEOUT", 3*time.Second),
		},
		Auth: Auth{
			IssuerURL:         env("OIDC_ISSUER_URL", ""),
			ClientID:          env("OIDC_CLIENT_ID", ""),
			ClientSecret:      env("OIDC_CLIENT_SECRET", ""),
			Audience:          env("OIDC_AUDIENCE", "skawld-api"),
			RedirectURL:       env("OIDC_REDIRECT_URL", "http://localhost:8080/auth/callback"),
			CookieName:        env("SESSION_COOKIE_NAME", "skawld_session"),
			CookieSecure:      envBool("SESSION_COOKIE_SECURE", true),
			SessionTTL:        envDuration("SESSION_TTL", 8*time.Hour),
			BootstrapSubjects: envCSV("BOOTSTRAP_OIDC_SUBJECTS"),
			EmailDomainAllowlist: envCSV("EMAIL_DOMAIN_ALLOWLIST"),
			FederatedOrgID:     env("FEDERATED_ORG_ID", ""),
		},
		Jobs: Jobs{
			EmbeddingConcurrency:     envInt("JOB_CONCURRENCY_EMBEDDING", 5),
			ReportConcurrency:        envInt("JOB_CONCURRENCY_REPORT", 10),
			VisionConcurrency:        envInt("JOB_CONCURRENCY_VISION", 3),
			TranscriptionConcurrency: envInt("JOB_CONCURRENCY_TRANSCRIPTION", 3),
		},
		ObjectStore: ObjectStore{
			Endpoint:        env("S3_ENDPOINT", ""),
			Region:          env("S3_REGION", "us-east-1"),
			Bucket:          env("S3_BUCKET", ""),
			AccessKeyID:     env("S3_ACCESS_KEY_ID", ""),
			SecretAccessKey: env("S3_SECRET_ACCESS_KEY", ""),
			UsePathStyle:    envBool("S3_USE_PATH_STYLE", true),
		},
		Documents: Documents{
			PDFToTextBinary: env("PDFTOTEXT_BINARY", "pdftotext"),
		},
		Transcription: Transcription{
			Endpoint:     env("TRANSCRIPTION_ENDPOINT", ""),
			APIKey:       env("TRANSCRIPTION_API_KEY", ""),
			Provider:     env("TRANSCRIPTION_PROVIDER", "unavailable"),
			Model:        env("TRANSCRIPTION_MODEL", "unavailable"),
			ModelVersion: env("TRANSCRIPTION_MODEL_VERSION", "none"),
		},
	}
	return cfg, errors.Join(cfg.Validate(), validateEnvironmentValues())
}

func (c Config) Validate() error {
	var errs []error
	if c.Role != RoleAPI && c.Role != RoleWorker {
		errs = append(errs, fmt.Errorf("unsupported process role %q", c.Role))
	}
	if strings.TrimSpace(c.Database.URL) == "" {
		errs = append(errs, errors.New("DATABASE_URL is required"))
	}
	if c.Database.APIMaxConns <= 0 || c.Database.WorkerMaxConns <= 0 {
		errs = append(errs, errors.New("database pool sizes must be positive"))
	}
	if c.Database.MaxBudget <= 0 {
		errs = append(errs, errors.New("database connection budget must be positive"))
	}
	if c.Database.StatementTimeout <= 0 || c.Database.LockTimeout <= 0 {
		errs = append(errs, errors.New("database statement and lock timeouts must be positive"))
	}
	if c.Database.APIMaxConns+c.Database.WorkerMaxConns > c.Database.MaxBudget {
		errs = append(errs, fmt.Errorf(
			"API and worker pools (%d) exceed DATABASE_MAX_CONNECTION_BUDGET (%d)",
			c.Database.APIMaxConns+c.Database.WorkerMaxConns,
			c.Database.MaxBudget,
		))
	}
	if c.Role == RoleAPI {
		if strings.TrimSpace(c.Auth.IssuerURL) == "" {
			errs = append(errs, errors.New("OIDC_ISSUER_URL is required for API role"))
		}
		if strings.TrimSpace(c.Auth.ClientID) == "" {
			errs = append(errs, errors.New("OIDC_CLIENT_ID is required for API role"))
		}
		if strings.TrimSpace(c.Auth.ClientSecret) == "" {
			errs = append(errs, errors.New("OIDC_CLIENT_SECRET is required for API role"))
		}
		if strings.TrimSpace(c.Auth.Audience) == "" {
			errs = append(errs, errors.New("OIDC_AUDIENCE is required for API role"))
		}
		redirect, err := url.Parse(c.Auth.RedirectURL)
		if err != nil || redirect.Scheme == "" || redirect.Host == "" {
			errs = append(errs, errors.New("OIDC_REDIRECT_URL must be an absolute URL"))
		}
		if strings.TrimSpace(c.Auth.CookieName) == "" {
			errs = append(errs, errors.New("SESSION_COOKIE_NAME is required"))
		}
		if c.Environment != "development" && c.Environment != "test" && !c.Auth.CookieSecure {
			errs = append(errs, errors.New("SESSION_COOKIE_SECURE must be true outside development/test"))
		}
		if len(c.Auth.EmailDomainAllowlist) > 0 {
			if strings.TrimSpace(c.Auth.FederatedOrgID) == "" {
				errs = append(errs, errors.New("FEDERATED_ORG_ID is required when EMAIL_DOMAIN_ALLOWLIST is set"))
			} else if _, err := uuid.Parse(c.Auth.FederatedOrgID); err != nil {
				errs = append(errs, errors.New("FEDERATED_ORG_ID must be a valid UUID"))
			}
		}
	}
	if c.Auth.SessionTTL <= 0 {
		errs = append(errs, errors.New("SESSION_TTL must be positive"))
	}
	if c.Role == RoleAPI || c.Role == RoleWorker {
		if strings.TrimSpace(c.ObjectStore.Region) == "" ||
			strings.TrimSpace(c.ObjectStore.Bucket) == "" {
			errs = append(errs, errors.New("S3_REGION and S3_BUCKET are required"))
		}
		if (c.ObjectStore.AccessKeyID == "") != (c.ObjectStore.SecretAccessKey == "") {
			errs = append(errs, errors.New("S3 access key ID and secret must be configured together"))
		}
	}
	if c.Role == RoleWorker && strings.TrimSpace(c.Documents.PDFToTextBinary) == "" {
		errs = append(errs, errors.New("PDFTOTEXT_BINARY is required for worker role"))
	}
	if c.Transcription.Endpoint != "" {
		endpoint, err := url.Parse(c.Transcription.Endpoint)
		if err != nil || !endpoint.IsAbs() ||
			(endpoint.Scheme != "http" && endpoint.Scheme != "https") {
			errs = append(errs, errors.New("TRANSCRIPTION_ENDPOINT must be an absolute HTTP(S) URL"))
		}
		if strings.TrimSpace(c.Transcription.Provider) == "" ||
			strings.TrimSpace(c.Transcription.Model) == "" ||
			strings.TrimSpace(c.Transcription.ModelVersion) == "" {
			errs = append(errs, errors.New("transcription model metadata is required"))
		}
	}
	for name, value := range map[string]int{
		"embedding":     c.Jobs.EmbeddingConcurrency,
		"report":        c.Jobs.ReportConcurrency,
		"vision":        c.Jobs.VisionConcurrency,
		"transcription": c.Jobs.TranscriptionConcurrency,
	} {
		if value <= 0 {
			errs = append(errs, fmt.Errorf("%s job concurrency must be positive", name))
		}
	}
	return errors.Join(errs...)
}

func validateEnvironmentValues() error {
	var errs []error
	for _, name := range []string{
		"DATABASE_API_MAX_CONNS",
		"DATABASE_WORKER_MAX_CONNS",
		"DATABASE_MAX_CONNECTION_BUDGET",
		"JOB_CONCURRENCY_EMBEDDING",
		"JOB_CONCURRENCY_REPORT",
		"JOB_CONCURRENCY_VISION",
		"JOB_CONCURRENCY_TRANSCRIPTION",
	} {
		if raw, ok := os.LookupEnv(name); ok {
			if _, err := strconv.Atoi(raw); err != nil {
				errs = append(errs, fmt.Errorf("%s must be an integer: %w", name, err))
			}
		}
	}
	for _, name := range []string{
		"DATABASE_STATEMENT_TIMEOUT",
		"DATABASE_LOCK_TIMEOUT",
		"SESSION_TTL",
	} {
		if raw, ok := os.LookupEnv(name); ok {
			if _, err := time.ParseDuration(raw); err != nil {
				errs = append(errs, fmt.Errorf("%s must be a duration: %w", name, err))
			}
		}
	}
	for _, name := range []string{"SESSION_COOKIE_SECURE", "S3_USE_PATH_STYLE"} {
		if raw, ok := os.LookupEnv(name); ok {
			if _, err := strconv.ParseBool(raw); err != nil {
				errs = append(errs, fmt.Errorf("%s must be a boolean: %w", name, err))
			}
		}
	}
	return errors.Join(errs...)
}

func env(name, fallback string) string {
	if value, ok := os.LookupEnv(name); ok {
		return value
	}
	return fallback
}

func envInt(name string, fallback int) int {
	raw, ok := os.LookupEnv(name)
	if !ok {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return value
}

func envBool(name string, fallback bool) bool {
	raw, ok := os.LookupEnv(name)
	if !ok {
		return fallback
	}
	value, err := strconv.ParseBool(raw)
	if err != nil {
		return fallback
	}
	return value
}

func envDuration(name string, fallback time.Duration) time.Duration {
	raw, ok := os.LookupEnv(name)
	if !ok {
		return fallback
	}
	value, err := time.ParseDuration(raw)
	if err != nil {
		return fallback
	}
	return value
}

func envCSV(name string) []string {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return nil
	}
	items := strings.Split(raw, ",")
	result := make([]string, 0, len(items))
	for _, item := range items {
		if value := strings.TrimSpace(item); value != "" {
			result = append(result, value)
		}
	}
	return result
}
