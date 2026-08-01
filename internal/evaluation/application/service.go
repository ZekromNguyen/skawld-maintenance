package application

import (
	"context"
	"errors"
	"time"

	evaluationdomain "github.com/ZekromNguyen/skawld-maintenance/internal/evaluation/domain"
	identitydomain "github.com/ZekromNguyen/skawld-maintenance/internal/identity/domain"
)

var ErrForbidden = errors.New("evaluation access forbidden")

type Store interface {
	Counts(
		context.Context,
		identitydomain.Principal,
	) (evaluationdomain.Counts, error)
}

type Service struct {
	Store Store
	Now   func() time.Time
}

func (s Service) Summary(
	ctx context.Context,
	principal identitydomain.Principal,
) (evaluationdomain.Summary, error) {
	if !principal.Has(identitydomain.PermissionRecommendationReview) {
		return evaluationdomain.Summary{}, ErrForbidden
	}
	counts, err := s.Store.Counts(ctx, principal)
	if err != nil {
		return evaluationdomain.Summary{}, err
	}
	now := time.Now
	if s.Now != nil {
		now = s.Now
	}
	return evaluationdomain.Build(counts, now()), nil
}
