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
	"sync"
)

type HTTPEmbeddingConfig struct {
	Endpoint     string
	APIKey       string
	Provider     string
	Model        string
	ModelVersion string
}

type HTTPEmbeddingProvider struct {
	endpoint string
	apiKey   string
	metadata ProviderMetadata
	client   *http.Client

	mu         sync.Mutex
	dimensions int
}

func NewHTTPEmbeddingProvider(
	config HTTPEmbeddingConfig,
	client *http.Client,
) (*HTTPEmbeddingProvider, error) {
	endpoint, err := url.Parse(strings.TrimSpace(config.Endpoint))
	if err != nil || !endpoint.IsAbs() ||
		(endpoint.Scheme != "http" && endpoint.Scheme != "https") {
		return nil, errors.New("embedding provider endpoint must be an absolute HTTP(S) URL")
	}
	if strings.TrimSpace(config.Provider) == "" ||
		strings.TrimSpace(config.Model) == "" ||
		strings.TrimSpace(config.ModelVersion) == "" {
		return nil, errors.New("embedding provider model metadata is required")
	}
	if client == nil {
		return nil, errors.New("embedding provider HTTP client is required")
	}
	return &HTTPEmbeddingProvider{
		endpoint: endpoint.String(), apiKey: config.APIKey, client: client,
		metadata: ProviderMetadata{
			Provider: config.Provider, Model: config.Model,
			ModelVersion: config.ModelVersion,
		},
	}, nil
}

// Model reports the configured embedding identity. Dimensions are populated
// from the first successful response; the knowledge ingest reads them after a
// successful Embed call.
func (p *HTTPEmbeddingProvider) Model() EmbeddingModel {
	p.mu.Lock()
	defer p.mu.Unlock()
	return EmbeddingModel{
		Provider: p.metadata.Provider, Model: p.metadata.Model,
		ModelVersion: p.metadata.ModelVersion, Metric: "COSINE",
		Dimensions: p.dimensions,
	}
}

func (p *HTTPEmbeddingProvider) Embed(
	ctx context.Context,
	inputs []string,
) ([][]float32, error) {
	if len(inputs) == 0 {
		return nil, errors.New("embedding provider requires at least one input")
	}
	payload, err := json.Marshal(map[string]any{
		"input": inputs,
		"model": p.metadata.Model,
	})
	if err != nil {
		return nil, err
	}
	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, p.endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	httpRequest.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		httpRequest.Header.Set("Authorization", "Bearer "+p.apiKey)
	}
	response, err := p.client.Do(httpRequest)
	if err != nil {
		return nil, fmt.Errorf("embedding provider request: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		limited, _ := io.ReadAll(io.LimitReader(response.Body, 1024))
		return nil, fmt.Errorf(
			"embedding provider status %d: %s",
			response.StatusCode, strings.TrimSpace(string(limited)),
		)
	}
	var value struct {
		Data []struct {
			Embedding []float32 `json:"embedding"`
		} `json:"data"`
	}
	decoder := json.NewDecoder(io.LimitReader(response.Body, 4<<20))
	if err := decoder.Decode(&value); err != nil {
		return nil, fmt.Errorf("decode embedding provider response: %w", err)
	}
	if len(value.Data) != len(inputs) {
		return nil, ErrInvalidOutput
	}
	result := make([][]float32, 0, len(value.Data))
	for index, item := range value.Data {
		if index > 0 && len(item.Embedding) != len(result[0]) {
			return nil, ErrInvalidOutput
		}
		if len(item.Embedding) == 0 {
			return nil, ErrInvalidOutput
		}
		result = append(result, item.Embedding)
	}
	p.mu.Lock()
	if p.dimensions == 0 {
		p.dimensions = len(result[0])
	}
	p.mu.Unlock()
	return result, nil
}
