package keyset

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

var ErrInvalidCursor = errors.New("invalid keyset cursor")

type Key struct {
	Timestamp time.Time
	ID        string
}

type cursorPayload struct {
	T string `json:"t"`
	I string `json:"i"`
}

func Encode(timestamp time.Time, id string) string {
	payload, _ := json.Marshal(cursorPayload{
		T: timestamp.UTC().Format(time.RFC3339Nano),
		I: id,
	})
	return base64.RawURLEncoding.EncodeToString(payload)
}

func Decode(cursor string) (Key, error) {
	raw, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return Key{}, fmt.Errorf("%w: malformed encoding", ErrInvalidCursor)
	}
	var payload cursorPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return Key{}, fmt.Errorf("%w: malformed payload", ErrInvalidCursor)
	}
	if payload.T == "" || payload.I == "" {
		return Key{}, fmt.Errorf("%w: missing fields", ErrInvalidCursor)
	}
	parsed, err := time.Parse(time.RFC3339Nano, payload.T)
	if err != nil {
		return Key{}, fmt.Errorf("%w: bad timestamp", ErrInvalidCursor)
	}
	if _, err := uuid.Parse(payload.I); err != nil {
		return Key{}, fmt.Errorf("%w: bad id", ErrInvalidCursor)
	}
	return Key{Timestamp: parsed, ID: payload.I}, nil
}
