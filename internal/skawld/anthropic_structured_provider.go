package skawld

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const anthropicAPIVersion = "2023-06-01"

// anthropicMessagesEndpoint is a package-level variable so tests can redirect
// the provider to a local httptest server; production uses the public API.
var anthropicMessagesEndpoint = "https://api.anthropic.com/v1/messages"

type AnthropicStructuredConfig struct {
	APIKey       string
	Model        string
	ModelVersion string
}

type AnthropicStructuredProvider struct {
	apiKey   string
	metadata ProviderMetadata
	client   *http.Client
}

func NewAnthropicStructuredProvider(
	config AnthropicStructuredConfig,
	client *http.Client,
) (*AnthropicStructuredProvider, error) {
	if strings.TrimSpace(config.APIKey) == "" ||
		strings.TrimSpace(config.Model) == "" ||
		strings.TrimSpace(config.ModelVersion) == "" {
		return nil, errors.New("anthropic provider requires api key, model, and model version")
	}
	if client == nil {
		return nil, errors.New("anthropic provider HTTP client is required")
	}
	return &AnthropicStructuredProvider{
		apiKey: config.APIKey, client: client,
		metadata: ProviderMetadata{
			Provider: "anthropic", Model: config.Model,
			ModelVersion: config.ModelVersion,
		},
	}, nil
}

func (p *AnthropicStructuredProvider) Generate(
	ctx context.Context,
	request GenerateRequest,
) (GenerateResponse, error) {
	systemPrompt := fmt.Sprintf(
		"Return exactly one JSON object for capability %q at schema version %s. "+
			"Do not include markdown fences or commentary.",
		request.Capability, request.SchemaVersion,
	)
	userPrompt := fmt.Sprintf(
		"Prompt version: %s\nEvidence:\n%s\nContext:\n%s",
		request.PromptVersion, joinEvidence(request.Evidence), string(request.Context),
	)
	payload := map[string]any{
		"model":      p.metadata.Model,
		"max_tokens": 2048,
		"system":     systemPrompt,
		"messages": []map[string]string{
			{"role": "user", "content": userPrompt},
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return GenerateResponse{}, err
	}
	httpRequest, err := http.NewRequestWithContext(
		ctx, http.MethodPost, anthropicMessagesEndpoint, bytes.NewReader(body),
	)
	if err != nil {
		return GenerateResponse{}, err
	}
	httpRequest.Header.Set("Content-Type", "application/json")
	httpRequest.Header.Set("x-api-key", p.apiKey)
	httpRequest.Header.Set("anthropic-version", anthropicAPIVersion)
	response, err := p.client.Do(httpRequest)
	if err != nil {
		return GenerateResponse{}, fmt.Errorf("anthropic provider request: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		limited, _ := io.ReadAll(io.LimitReader(response.Body, 1024))
		return GenerateResponse{}, fmt.Errorf(
			"anthropic provider status %d: %s",
			response.StatusCode, strings.TrimSpace(string(limited)),
		)
	}
	var value struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		Usage struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}
	decoder := json.NewDecoder(io.LimitReader(response.Body, 1<<20))
	if err := decoder.Decode(&value); err != nil {
		return GenerateResponse{}, fmt.Errorf("decode anthropic provider response: %w", err)
	}
	if len(value.Content) == 0 {
		return GenerateResponse{}, ErrInvalidOutput
	}
	text := strings.TrimSpace(value.Content[0].Text)
	if text == "" {
		return GenerateResponse{}, ErrInvalidOutput
	}
	if !json.Valid([]byte(text)) {
		return GenerateResponse{}, ErrInvalidOutput
	}
	return GenerateResponse{
		Output: json.RawMessage(text),
		Metadata: ProviderMetadata{
			Provider: p.metadata.Provider, Model: p.metadata.Model,
			ModelVersion: p.metadata.ModelVersion,
			TokensIn:     value.Usage.InputTokens,
			TokensOut:    value.Usage.OutputTokens,
		},
	}, nil
}
