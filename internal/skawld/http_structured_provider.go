package skawld

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type HTTPStructuredConfig struct {
	Endpoint     string
	APIKey       string
	Provider     string
	Model        string
	ModelVersion string
}

type HTTPStructuredProvider struct {
	endpoint string
	apiKey   string
	metadata ProviderMetadata
	client   *http.Client
}

func NewHTTPStructuredProvider(
	config HTTPStructuredConfig,
	client *http.Client,
) (*HTTPStructuredProvider, error) {
	endpoint, err := url.Parse(strings.TrimSpace(config.Endpoint))
	if err != nil || !endpoint.IsAbs() ||
		(endpoint.Scheme != "http" && endpoint.Scheme != "https") {
		return nil, errors.New("structured provider endpoint must be an absolute HTTP(S) URL")
	}
	if strings.TrimSpace(config.Provider) == "" ||
		strings.TrimSpace(config.Model) == "" ||
		strings.TrimSpace(config.ModelVersion) == "" {
		return nil, errors.New("structured provider model metadata is required")
	}
	if client == nil {
		return nil, errors.New("structured provider HTTP client is required")
	}
	return &HTTPStructuredProvider{
		endpoint: endpoint.String(), apiKey: config.APIKey, client: client,
		metadata: ProviderMetadata{
			Provider: config.Provider, Model: config.Model,
			ModelVersion: config.ModelVersion,
		},
	}, nil
}

func (p *HTTPStructuredProvider) Generate(
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
		"model": p.metadata.Model,
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": userPrompt},
		},
		"response_format": map[string]string{"type": "json_object"},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return GenerateResponse{}, err
	}
	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, p.endpoint, bytes.NewReader(body))
	if err != nil {
		return GenerateResponse{}, err
	}
	httpRequest.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		httpRequest.Header.Set("Authorization", "Bearer "+p.apiKey)
	}
	response, err := p.client.Do(httpRequest)
	if err != nil {
		return GenerateResponse{}, fmt.Errorf("structured provider request: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		limited, _ := io.ReadAll(io.LimitReader(response.Body, 1024))
		return GenerateResponse{}, fmt.Errorf(
			"structured provider status %d: %s",
			response.StatusCode, strings.TrimSpace(string(limited)),
		)
	}
	var value struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
		} `json:"usage"`
	}
	decoder := json.NewDecoder(io.LimitReader(response.Body, 1<<20))
	if err := decoder.Decode(&value); err != nil {
		return GenerateResponse{}, fmt.Errorf("decode structured provider response: %w", err)
	}
	if len(value.Choices) == 0 {
		return GenerateResponse{}, ErrInvalidOutput
	}
	content := strings.TrimSpace(value.Choices[0].Message.Content)
	if content == "" {
		return GenerateResponse{}, ErrInvalidOutput
	}
	if !json.Valid([]byte(content)) {
		return GenerateResponse{}, ErrInvalidOutput
	}
	return GenerateResponse{
		Output: json.RawMessage(content),
		Metadata: ProviderMetadata{
			Provider: p.metadata.Provider, Model: p.metadata.Model,
			ModelVersion: p.metadata.ModelVersion,
			TokensIn:     value.Usage.PromptTokens,
			TokensOut:    value.Usage.CompletionTokens,
		},
	}, nil
}
