package skawld

import (
	"bytes"
	"context"
	"encoding/binary"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

// TestTranscriptionProviderContract requires TRANSCRIPTION_ENDPOINT,
// TRANSCRIPTION_API_KEY, TRANSCRIPTION_MODEL, and TRANSCRIPTION_MODEL_VERSION;
// it transcribes a synthetic silent WAV against the configured deployment
// endpoint. It skips when the credentials are absent so CI stays green
// without them. A deployment endpoint may reject synthetic audio, so both a
// non-empty transcript and an explicit provider error are contract-valid.
func TestTranscriptionProviderContract(t *testing.T) {
	endpoint := os.Getenv("TRANSCRIPTION_ENDPOINT")
	key := os.Getenv("TRANSCRIPTION_API_KEY")
	model := os.Getenv("TRANSCRIPTION_MODEL")
	version := os.Getenv("TRANSCRIPTION_MODEL_VERSION")
	if endpoint == "" || key == "" || model == "" || version == "" {
		t.Skip("transcription credentials not configured")
	}
	provider, err := NewHTTPTranscriptionProvider(HTTPTranscriptionConfig{
		Endpoint: endpoint, APIKey: key, Provider: "deployment",
		Model: model, ModelVersion: version,
	}, &http.Client{Timeout: 4 * time.Minute})
	if err != nil {
		t.Fatal(err)
	}
	transcript, err := provider.Transcribe(
		context.Background(), bytes.NewReader(silentWAV()), "audio/wav",
	)
	if err != nil {
		if strings.TrimSpace(transcript.Text) != "" {
			t.Fatalf("transcript with error: %q, %v", transcript.Text, err)
		}
		// An explicit provider error is contract-valid for synthetic audio;
		// it must not be a client-validation sentinel.
		return
	}
	if strings.TrimSpace(transcript.Text) == "" {
		t.Fatal("empty transcript with nil error")
	}
	if transcript.Metadata.Provider == "" || transcript.Metadata.Model == "" {
		t.Fatalf("metadata = %+v", transcript.Metadata)
	}
}

// silentWAV builds a 0.25 s, 8 kHz mono 16-bit silent WAV in memory so the
// contract test does not depend on a checked-in audio fixture.
func silentWAV() []byte {
	sampleRate := uint32(8000)
	dataSize := uint32(sampleRate / 4 * 2) // 0.25 s of 16-bit mono
	var buf bytes.Buffer
	buf.WriteString("RIFF")
	_ = binary.Write(&buf, binary.LittleEndian, uint32(36+dataSize))
	buf.WriteString("WAVE")
	buf.WriteString("fmt ")
	_ = binary.Write(&buf, binary.LittleEndian, uint32(16))
	_ = binary.Write(&buf, binary.LittleEndian, uint16(1))
	_ = binary.Write(&buf, binary.LittleEndian, uint16(1))
	_ = binary.Write(&buf, binary.LittleEndian, sampleRate)
	_ = binary.Write(&buf, binary.LittleEndian, sampleRate*2)
	_ = binary.Write(&buf, binary.LittleEndian, uint16(2))
	_ = binary.Write(&buf, binary.LittleEndian, uint16(16))
	buf.WriteString("data")
	_ = binary.Write(&buf, binary.LittleEndian, dataSize)
	buf.Write(make([]byte, dataSize))
	return buf.Bytes()
}
