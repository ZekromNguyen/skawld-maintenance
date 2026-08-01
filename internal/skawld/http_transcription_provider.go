package skawld

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
)

type HTTPTranscriptionConfig struct {
	Endpoint     string
	APIKey       string
	Provider     string
	Model        string
	ModelVersion string
}

type HTTPTranscriptionProvider struct {
	endpoint string
	apiKey   string
	metadata ProviderMetadata
	client   *http.Client
}

func NewHTTPTranscriptionProvider(
	config HTTPTranscriptionConfig,
	client *http.Client,
) (*HTTPTranscriptionProvider, error) {
	endpoint, err := url.Parse(strings.TrimSpace(config.Endpoint))
	if err != nil || !endpoint.IsAbs() ||
		(endpoint.Scheme != "http" && endpoint.Scheme != "https") {
		return nil, errors.New("transcription endpoint must be an absolute HTTP(S) URL")
	}
	if strings.TrimSpace(config.Provider) == "" ||
		strings.TrimSpace(config.Model) == "" ||
		strings.TrimSpace(config.ModelVersion) == "" {
		return nil, errors.New("transcription provider model metadata is required")
	}
	if client == nil {
		return nil, errors.New("transcription HTTP client is required")
	}
	return &HTTPTranscriptionProvider{
		endpoint: endpoint.String(), apiKey: config.APIKey, client: client,
		metadata: ProviderMetadata{
			Provider: config.Provider, Model: config.Model,
			ModelVersion: config.ModelVersion,
		},
	}, nil
}

func (p *HTTPTranscriptionProvider) Model() ProviderMetadata {
	return p.metadata
}

func (p *HTTPTranscriptionProvider) Transcribe(
	ctx context.Context,
	audio io.Reader,
	mimeType string,
) (Transcript, error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "evidence-audio")
	if err != nil {
		return Transcript{}, err
	}
	written, err := io.Copy(part, io.LimitReader(audio, (50<<20)+1))
	if err != nil {
		return Transcript{}, err
	}
	if written > 50<<20 {
		return Transcript{}, errors.New("transcription audio exceeds 50 MiB")
	}
	if err := writer.WriteField("model", p.metadata.Model); err != nil {
		return Transcript{}, err
	}
	if err := writer.WriteField("response_format", "json"); err != nil {
		return Transcript{}, err
	}
	if err := writer.Close(); err != nil {
		return Transcript{}, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, p.endpoint, &body)
	if err != nil {
		return Transcript{}, err
	}
	request.Header.Set("Content-Type", writer.FormDataContentType())
	request.Header.Set("X-Skawld-Source-MIME", mimeType)
	if p.apiKey != "" {
		request.Header.Set("Authorization", "Bearer "+p.apiKey)
	}
	response, err := p.client.Do(request)
	if err != nil {
		return Transcript{}, fmt.Errorf("transcription provider request: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		limited, _ := io.ReadAll(io.LimitReader(response.Body, 1024))
		return Transcript{}, fmt.Errorf(
			"transcription provider status %d: %s",
			response.StatusCode, strings.TrimSpace(string(limited)),
		)
	}
	var value struct {
		Text       string  `json:"text"`
		Language   string  `json:"language"`
		Confidence float64 `json:"confidence"`
	}
	decoder := json.NewDecoder(io.LimitReader(response.Body, 1<<20))
	if err := decoder.Decode(&value); err != nil {
		return Transcript{}, fmt.Errorf("decode transcription response: %w", err)
	}
	value.Text = strings.TrimSpace(value.Text)
	if value.Text == "" || value.Confidence < 0 || value.Confidence > 1 {
		return Transcript{}, ErrInvalidOutput
	}
	return Transcript{
		Text: value.Text, Language: value.Language,
		Confidence: value.Confidence, Metadata: p.metadata,
	}, nil
}
