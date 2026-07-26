package application

import (
	"bytes"
	"context"
	"io"
	"testing"

	identitydomain "github.com/ZekromNguyen/skawld-maintenance/internal/identity/domain"
	"github.com/ZekromNguyen/skawld-maintenance/internal/skawld"
)

type fakeStore struct {
	requestCalls int
	verifyCalls  int
}

func (f *fakeStore) Request(
	context.Context,
	identitydomain.Principal,
	string,
	string,
	skawld.ProviderMetadata,
) (Transcription, bool, error) {
	f.requestCalls++
	return Transcription{ID: "transcription-1", State: "QUEUED"}, false, nil
}

func (f *fakeStore) Get(
	context.Context,
	identitydomain.Principal,
	string,
) (Transcription, error) {
	return Transcription{ID: "transcription-1"}, nil
}

func (f *fakeStore) Verify(
	context.Context,
	identitydomain.Principal,
	string,
	string,
	Verify,
) (Transcription, bool, error) {
	f.verifyCalls++
	return Transcription{ID: "transcription-1", State: "VERIFIED"}, false, nil
}

type fakeProvider struct {
	metadata skawld.ProviderMetadata
}

func (p fakeProvider) Model() skawld.ProviderMetadata {
	return p.metadata
}

func (p fakeProvider) Transcribe(
	context.Context,
	io.Reader,
	string,
) (skawld.Transcript, error) {
	return skawld.Transcript{
		Text:     "candidate",
		Metadata: p.metadata,
	}, nil
}

func TestRequestFailsClosedWithoutConfiguredProvider(t *testing.T) {
	t.Parallel()
	store := &fakeStore{}
	service := Service{Store: store}
	principal := principalWith(identitydomain.PermissionAttachmentWrite)

	_, _, err := service.Request(
		context.Background(),
		principal,
		"request-key-123",
		"attachment-1",
	)
	if err != ErrUnavailable {
		t.Fatalf("Request error = %v, want %v", err, ErrUnavailable)
	}
	if store.requestCalls != 0 {
		t.Fatalf("store request calls = %d, want 0", store.requestCalls)
	}
}

func TestRequestUsesPinnedProviderMetadata(t *testing.T) {
	t.Parallel()
	store := &fakeStore{}
	provider := fakeProvider{metadata: skawld.ProviderMetadata{
		Provider:     "private-speech",
		Model:        "whisper-compatible",
		ModelVersion: "2026-07",
	}}
	service := Service{Store: store, Provider: provider}
	principal := principalWith(identitydomain.PermissionAttachmentWrite)

	value, replay, err := service.Request(
		context.Background(),
		principal,
		"request-key-123",
		"attachment-1",
	)
	if err != nil {
		t.Fatal(err)
	}
	if replay || value.State != "QUEUED" || store.requestCalls != 1 {
		t.Fatalf("Request = (%+v, %v), calls = %d", value, replay, store.requestCalls)
	}
}

func TestVerifyRequiresHumanTranscriptWhenAccepted(t *testing.T) {
	t.Parallel()
	store := &fakeStore{}
	service := Service{Store: store}
	principal := principalWith(identitydomain.PermissionExecutionWrite)

	_, _, err := service.Verify(
		context.Background(),
		principal,
		"verify-key-1234",
		"transcription-1",
		Verify{ExpectedVersion: 2, Accept: true},
	)
	if err != ErrInvalid {
		t.Fatalf("Verify error = %v, want %v", err, ErrInvalid)
	}
	if store.verifyCalls != 0 {
		t.Fatalf("store verify calls = %d, want 0", store.verifyCalls)
	}
}

func TestFakeProviderContract(t *testing.T) {
	t.Parallel()
	provider := fakeProvider{metadata: skawld.ProviderMetadata{
		Provider:     "fake",
		Model:        "speech",
		ModelVersion: "v1",
	}}
	got, err := provider.Transcribe(
		context.Background(),
		bytes.NewBufferString("audio"),
		"audio/wav",
	)
	if err != nil || got.Metadata != provider.Model() {
		t.Fatalf("Transcribe = (%+v, %v)", got, err)
	}
}

func principalWith(permission identitydomain.Permission) identitydomain.Principal {
	return identitydomain.Principal{
		ID: "principal-1",
		Permissions: map[identitydomain.Permission]struct{}{
			permission: {},
		},
	}
}
