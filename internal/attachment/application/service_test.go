package application

import (
	"context"
	"errors"
	"testing"

	identitydomain "github.com/ZekromNguyen/skawld-maintenance/internal/identity/domain"
)

type fakeStore struct {
	command CreateManifest
}

func TestCompleteRejectsMalformedIdempotencyKeyAsInvalidInput(t *testing.T) {
	service := Service{Store: &fakeStore{}}
	principal := identitydomain.Principal{
		ID: "user", OrganizationID: "org",
		Permissions: map[identitydomain.Permission]struct{}{
			identitydomain.PermissionAttachmentWrite: {},
		},
	}

	_, _, err := service.Complete(
		context.Background(), principal, "short", "attachment",
	)
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("expected invalid command, got %v", err)
	}
}

func (f *fakeStore) Create(
	_ context.Context,
	_ identitydomain.Principal,
	_ string,
	command CreateManifest,
) (Attachment, bool, error) {
	f.command = command
	return Attachment{}, false, nil
}

func (*fakeStore) Complete(
	context.Context,
	identitydomain.Principal,
	string,
	string,
	CompleteUpload,
) (Attachment, bool, error) {
	return Attachment{}, false, nil
}

func TestCreateSanitizesFilename(t *testing.T) {
	store := &fakeStore{}
	service := Service{Store: store}
	principal := identitydomain.Principal{
		ID: "user", OrganizationID: "org", SiteIDs: []string{"site"},
		Permissions: map[identitydomain.Permission]struct{}{
			identitydomain.PermissionAttachmentWrite: {},
		},
	}
	_, _, err := service.Create(context.Background(), principal, "request-123", CreateManifest{
		SiteID: "site", EntityKind: "EXECUTION", EntityID: "execution",
		ClientEventID: "event", OriginalFilename: "../../bearing.jpg",
		DeclaredMIME: "image/jpeg", SizeBytes: 42,
		ChecksumSHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
	})
	if err != nil {
		t.Fatal(err)
	}
	if store.command.OriginalFilename != "bearing.jpg" {
		t.Fatalf("filename was not sanitized: %q", store.command.OriginalFilename)
	}
}
