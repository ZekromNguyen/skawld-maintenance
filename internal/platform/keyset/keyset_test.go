package keyset

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestRoundTrip(t *testing.T) {
	now := time.Date(2026, 8, 6, 12, 0, 0, 123456789, time.UTC)
	id := "3b0f3c1e-9f4a-4f0e-8b2a-123456789abc"
	cursor := Encode(now, id)
	if cursor == "" {
		t.Fatal("cursor must not be empty")
	}
	key, err := Decode(cursor)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if !key.Timestamp.Equal(now) {
		t.Fatalf("timestamp = %v, want %v", key.Timestamp, now)
	}
	if key.ID != id {
		t.Fatalf("id = %q, want %q", key.ID, id)
	}
}

func TestDecodeRejectsGarbage(t *testing.T) {
	cases := []string{"", "not-base64!!", "e30=" /* empty json {} */, "aGVsbG8=" /* "hello" */}
	for _, cursor := range cases {
		if _, err := Decode(cursor); !errors.Is(err, ErrInvalidCursor) {
			t.Fatalf("Decode(%q) error = %v, want ErrInvalidCursor", cursor, err)
		}
	}
}

func TestEncodeProducesURLSafeBase64(t *testing.T) {
	cursor := Encode(time.Now().UTC(), "3b0f3c1e-9f4a-4f0e-8b2a-123456789abc")
	if strings.ContainsAny(cursor, "+/=") {
		t.Fatalf("cursor %q contains base64 URL-unsafe characters", cursor)
	}
}
