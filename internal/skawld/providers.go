package skawld

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const (
	ProviderDeterministic = "deterministic"
	ProviderOpenAI        = "openai"
	ProviderAnthropic     = "anthropic"
)

// AIConfig carries the env-selected provider choices and credentials from the
// composition roots. It mirrors config.AI without coupling the boundary
// package to the configuration package.
type AIConfig struct {
	StructuredProvider     string
	EmbeddingProvider      string
	StructuredEndpoint     string
	StructuredAPIKey       string
	StructuredModel        string
	StructuredModelVersion string
	AnthropicAPIKey        string
	AnthropicModel         string
	AnthropicModelVersion  string
	EmbeddingEndpoint      string
	EmbeddingModel         string
	EmbeddingModelVersion  string
}

// BuildProviders selects the structured and embedding providers from the
// environment-driven configuration. Deterministic providers are the default
// so CI, dev, and seed run without AI credentials; unknown selections and
// missing credentials fail closed at startup.
func BuildProviders(
	cfg AIConfig,
	client *http.Client,
) (map[Capability]StructuredProvider, EmbeddingProvider, error) {
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	structuredProvider, err := buildStructuredProvider(cfg, client)
	if err != nil {
		return nil, nil, err
	}
	embeddingProvider, err := buildEmbeddingProvider(cfg, client)
	if err != nil {
		return nil, nil, err
	}
	return map[Capability]StructuredProvider{
		CapabilityRecommendation: structuredProvider,
		CapabilityReportDraft:    structuredProvider,
		CapabilityShiftHandover:  structuredProvider,
	}, embeddingProvider, nil
}

func buildStructuredProvider(cfg AIConfig, client *http.Client) (StructuredProvider, error) {
	switch strings.ToLower(strings.TrimSpace(cfg.StructuredProvider)) {
	case "", ProviderDeterministic:
		return DeterministicProvider{}, nil
	case ProviderOpenAI:
		if strings.TrimSpace(cfg.StructuredModel) == "" ||
			strings.TrimSpace(cfg.StructuredModelVersion) == "" {
			return nil, errors.New("openai structured provider requires model and model version")
		}
		return NewHTTPStructuredProvider(HTTPStructuredConfig{
			Endpoint: cfg.StructuredEndpoint, APIKey: cfg.StructuredAPIKey,
			Provider: ProviderOpenAI, Model: cfg.StructuredModel,
			ModelVersion: cfg.StructuredModelVersion,
		}, client)
	case ProviderAnthropic:
		return NewAnthropicStructuredProvider(AnthropicStructuredConfig{
			APIKey: cfg.AnthropicAPIKey, Model: cfg.AnthropicModel,
			ModelVersion: cfg.AnthropicModelVersion,
		}, client)
	default:
		return nil, fmt.Errorf("unsupported structured provider %q", cfg.StructuredProvider)
	}
}

func buildEmbeddingProvider(cfg AIConfig, client *http.Client) (EmbeddingProvider, error) {
	switch strings.ToLower(strings.TrimSpace(cfg.EmbeddingProvider)) {
	case "", ProviderDeterministic:
		return DeterministicEmbeddingProvider{Dimensions: 64}, nil
	case ProviderOpenAI:
		if strings.TrimSpace(cfg.EmbeddingModel) == "" ||
			strings.TrimSpace(cfg.EmbeddingModelVersion) == "" {
			return nil, errors.New("openai embedding provider requires model and model version")
		}
		return NewHTTPEmbeddingProvider(HTTPEmbeddingConfig{
			Endpoint: cfg.EmbeddingEndpoint, APIKey: cfg.StructuredAPIKey,
			Provider: ProviderOpenAI, Model: cfg.EmbeddingModel,
			ModelVersion: cfg.EmbeddingModelVersion,
		}, client)
	default:
		return nil, fmt.Errorf("unsupported embedding provider %q", cfg.EmbeddingProvider)
	}
}
