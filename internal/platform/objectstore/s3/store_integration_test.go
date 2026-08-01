package s3_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/config"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/objectstore"
	s3store "github.com/ZekromNguyen/skawld-maintenance/internal/platform/objectstore/s3"
	"github.com/google/uuid"
)

func TestUnavailableObjectStoreReturnsAnError(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	store, err := s3store.New(ctx, config.ObjectStore{
		Endpoint: "http://127.0.0.1:1", Region: "us-east-1",
		Bucket: "unavailable", AccessKeyID: "test",
		SecretAccessKey: "test", UsePathStyle: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.Put(ctx, objectstore.PutRequest{
		Key: "must-fail", Body: bytes.NewReader([]byte("x")),
		Size: 1, ContentType: "text/plain",
	})
	if err == nil {
		t.Fatal("unavailable object storage must not be reported as success")
	}
}

func TestRequiredS3CapabilitySubset(t *testing.T) {
	endpoint := os.Getenv("TEST_S3_ENDPOINT")
	if endpoint == "" {
		t.Skip("TEST_S3_ENDPOINT is not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	store, err := s3store.New(ctx, config.ObjectStore{
		Endpoint: endpoint, Region: "us-east-1",
		Bucket: "skawld-dev", AccessKeyID: "test",
		SecretAccessKey: "test", UsePathStyle: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	key := "contract-tests/" + uuid.NewString()
	defer func() { _ = store.Delete(context.Background(), key) }()

	body := []byte("skawld object contract")
	created, err := store.Put(ctx, objectstore.PutRequest{
		Key: key, Body: bytes.NewReader(body), Size: int64(len(body)),
		ContentType: "text/plain", Metadata: map[string]string{"source": "contract-test"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.Key != key {
		t.Fatalf("created key = %q", created.Key)
	}
	head, err := store.Head(ctx, key)
	if err != nil {
		t.Fatal(err)
	}
	if head.Size != int64(len(body)) {
		t.Fatalf("head size = %d", head.Size)
	}
	reader, downloaded, err := store.Get(ctx, key)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	value, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(value, body) || downloaded.ContentType != "text/plain" {
		t.Fatalf("downloaded value or MIME differs")
	}

	signedKey := key + "-signed"
	defer func() { _ = store.Delete(context.Background(), signedKey) }()
	signedBody := []byte("signed upload")
	signed, err := store.SignedUploadURL(
		ctx, signedKey,
		objectstore.UploadConstraints{
			ContentType: "text/plain", Size: int64(len(signedBody)),
			Metadata: map[string]string{"checksum-sha256": "contract"},
		},
		time.Minute,
	)
	if err != nil {
		t.Fatal(err)
	}
	for name := range signed.Headers {
		if http.CanonicalHeaderKey(name) == "Host" {
			t.Fatal("Host must be derived from the signed URL, not set by clients")
		}
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPut, signed.URL, bytes.NewReader(signedBody))
	if err != nil {
		t.Fatal(err)
	}
	for name, value := range signed.Headers {
		request.Header.Set(name, value)
	}
	request.ContentLength = int64(len(signedBody))
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		detail, _ := io.ReadAll(response.Body)
		t.Fatalf("signed upload status = %d: %s", response.StatusCode, detail)
	}
	if _, err := store.Head(ctx, signedKey); err != nil {
		t.Fatal(err)
	}
}
