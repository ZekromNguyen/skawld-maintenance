package application

import (
	"context"
	"errors"

	identitydomain "github.com/ZekromNguyen/skawld-maintenance/internal/identity/domain"
)

var (
	ErrForbidden = errors.New("team operation forbidden")
	ErrInvalid   = errors.New("invalid team command")
)

type Team struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Person struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
}

type Store interface {
	ListTeams(context.Context, identitydomain.Principal) ([]Team, error)
	ListPeople(context.Context, identitydomain.Principal) ([]Person, error)
}

type Service struct {
	Store Store
}

func (s Service) ListTeams(ctx context.Context, principal identitydomain.Principal) ([]Team, error) {
	if !principal.Has(identitydomain.PermissionIncidentCreate) {
		return nil, ErrForbidden
	}
	return s.Store.ListTeams(ctx, principal)
}

func (s Service) ListPeople(ctx context.Context, principal identitydomain.Principal) ([]Person, error) {
	if !principal.Has(identitydomain.PermissionIncidentCreate) {
		return nil, ErrForbidden
	}
	return s.Store.ListPeople(ctx, principal)
}
