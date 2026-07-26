package application

import (
	"context"
	"errors"
	"testing"

	evaluationdomain "github.com/ZekromNguyen/skawld-maintenance/internal/evaluation/domain"
	identitydomain "github.com/ZekromNguyen/skawld-maintenance/internal/identity/domain"
)

func TestSummaryRequiresReviewerPermission(t *testing.T) {
	t.Parallel()
	service := Service{Store: countStore{}}
	_, err := service.Summary(
		context.Background(),
		identitydomain.Principal{Permissions: map[identitydomain.Permission]struct{}{}},
	)
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("summary error = %v, want forbidden", err)
	}
}

type countStore struct{}

func (countStore) Counts(
	context.Context,
	identitydomain.Principal,
) (evaluationdomain.Counts, error) {
	return evaluationdomain.Counts{}, nil
}
