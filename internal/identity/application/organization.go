package application

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/ZekromNguyen/skawld-maintenance/internal/identity/domain"
)

var (
	ErrPermissionDenied    = errors.New("permission denied")
	ErrIdempotencyKey      = errors.New("idempotency key must contain 8 to 200 characters")
	ErrInvalidOrganization = errors.New("organization name is required")
)

type CreateOrganization struct {
	Name string `json:"name"`
}

type CreateOrganizationResult struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	SourceOfTruth string `json:"source_of_truth"`
	Version       int64  `json:"version"`
}

type AtomicOrganizationStore interface {
	Create(
		ctx context.Context,
		principal domain.Principal,
		idempotencyKey string,
		command CreateOrganization,
	) (result CreateOrganizationResult, replay bool, err error)
}

type OrganizationService struct {
	Store AtomicOrganizationStore
}

func (s OrganizationService) Create(
	ctx context.Context,
	principal domain.Principal,
	idempotencyKey string,
	command CreateOrganization,
) (CreateOrganizationResult, bool, error) {
	if !principal.Has(domain.PermissionOrganizationCreate) {
		return CreateOrganizationResult{}, false, ErrPermissionDenied
	}
	idempotencyKey = strings.TrimSpace(idempotencyKey)
	if len(idempotencyKey) < 8 || len(idempotencyKey) > 200 {
		return CreateOrganizationResult{}, false, ErrIdempotencyKey
	}
	if strings.TrimSpace(command.Name) == "" {
		return CreateOrganizationResult{}, false, ErrInvalidOrganization
	}
	if s.Store == nil {
		return CreateOrganizationResult{}, false, fmt.Errorf("organization store is required")
	}
	return s.Store.Create(ctx, principal, idempotencyKey, command)
}
