package objectstore

import (
	"context"
	"io"
	"time"
)

type Object struct {
	Key         string
	Size        int64
	ContentType string
	ETag        string
	Metadata    map[string]string
}

type PutRequest struct {
	Key         string
	Body        io.Reader
	Size        int64
	ContentType string
	Metadata    map[string]string
}

// Store is intentionally smaller than the S3 API. Product code depends only on
// the binary-object capabilities it owns; provider-specific features remain in
// adapters.
type Store interface {
	Put(ctx context.Context, request PutRequest) (Object, error)
	Get(ctx context.Context, key string) (io.ReadCloser, Object, error)
	Head(ctx context.Context, key string) (Object, error)
	Delete(ctx context.Context, key string) error
	SignedUploadURL(ctx context.Context, key, contentType string, expiresIn time.Duration) (string, error)
	SignedDownloadURL(ctx context.Context, key string, expiresIn time.Duration) (string, error)
}
