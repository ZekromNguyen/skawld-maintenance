package skawld

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHTTPTranscriptionProviderUsesFixedConfiguredEndpoint(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer secret" {
			t.Error("missing provider authorization")
		}
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Error(err)
			return
		}
		if r.FormValue("model") != "speech-small" {
			t.Errorf("model = %q", r.FormValue("model"))
		}
		file, _, err := r.FormFile("file")
		if err != nil {
			t.Error(err)
			return
		}
		_ = file.Close()
		_ = json.NewEncoder(w).Encode(map[string]any{
			"text":     "Motor-side bearing is hotter than normal.",
			"language": "en", "confidence": 0.91,
		})
	}))
	defer server.Close()
	provider, err := NewHTTPTranscriptionProvider(
		HTTPTranscriptionConfig{
			Endpoint: server.URL, APIKey: "secret", Provider: "private-speech",
			Model: "speech-small", ModelVersion: "2026-07",
		},
		server.Client(),
	)
	if err != nil {
		t.Fatal(err)
	}
	value, err := provider.Transcribe(
		context.Background(), strings.NewReader("audio"), "audio/wav",
	)
	if err != nil {
		t.Fatal(err)
	}
	if value.Text == "" || value.Confidence != 0.91 ||
		value.Metadata != provider.Model() {
		t.Fatalf("unexpected transcript: %+v", value)
	}
}
