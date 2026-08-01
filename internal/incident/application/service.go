package application

import (
	"context"
	"errors"
	"strings"
	"time"

	identitydomain "github.com/ZekromNguyen/skawld-maintenance/internal/identity/domain"
)

var (
	ErrForbidden       = errors.New("incident operation forbidden")
	ErrNotFound        = errors.New("incident not found")
	ErrInvalid         = errors.New("invalid incident command")
	ErrVersionConflict = errors.New("incident version conflict")
)

type CreateIncident struct {
	SiteID          string     `json:"site_id"`
	AssetID         string     `json:"asset_id"`
	Summary         string     `json:"summary"`
	Severity        string     `json:"severity"`
	SourceOfTruth   string     `json:"source_of_truth"`
	ExternalSystem  string     `json:"external_system,omitempty"`
	ExternalID      string     `json:"external_id,omitempty"`
	ExternalVersion string     `json:"external_version,omitempty"`
	OccurredAt      *time.Time `json:"occurred_at,omitempty"`
	DetectedAt      time.Time  `json:"detected_at"`
}

type ResolveIncident struct {
	ExpectedVersion   int64  `json:"expected_version"`
	ResolutionSummary string `json:"resolution_summary"`
}

type Incident struct {
	ID                string     `json:"id"`
	OrganizationID    string     `json:"organization_id"`
	SiteID            string     `json:"site_id"`
	AssetID           string     `json:"asset_id"`
	AssetTag          string     `json:"asset_tag,omitempty"`
	Number            string     `json:"number"`
	Summary           string     `json:"summary"`
	Severity          string     `json:"severity"`
	State             string     `json:"state"`
	SourceOfTruth     string     `json:"source_of_truth"`
	ExternalSystem    string     `json:"external_system,omitempty"`
	ExternalID        string     `json:"external_id,omitempty"`
	ExternalVersion   string     `json:"external_version,omitempty"`
	OccurredAt        *time.Time `json:"occurred_at,omitempty"`
	DetectedAt        time.Time  `json:"detected_at"`
	ResolvedAt        *time.Time `json:"resolved_at,omitempty"`
	ResolutionSummary string     `json:"resolution_summary,omitempty"`
	Version           int64      `json:"version"`
}

type Filter struct {
	SiteID  string
	AssetID string
	State   string
}

type Store interface {
	Create(context.Context, identitydomain.Principal, string, CreateIncident) (Incident, bool, error)
	Get(context.Context, identitydomain.Principal, string) (Incident, error)
	List(context.Context, identitydomain.Principal, Filter) ([]Incident, error)
	Resolve(context.Context, identitydomain.Principal, string, string, ResolveIncident) (Incident, bool, error)
}

type Service struct {
	Store Store
}

func (s Service) Create(
	ctx context.Context,
	principal identitydomain.Principal,
	key string,
	command CreateIncident,
) (Incident, bool, error) {
	if !principal.Has(identitydomain.PermissionIncidentCreate) ||
		!principal.CanAccessSite(principal.OrganizationID, command.SiteID) {
		return Incident{}, false, ErrForbidden
	}
	if !validKey(key) || strings.TrimSpace(command.AssetID) == "" ||
		strings.TrimSpace(command.Summary) == "" || command.DetectedAt.IsZero() {
		return Incident{}, false, ErrInvalid
	}
	return s.Store.Create(ctx, principal, key, command)
}

func (s Service) Get(
	ctx context.Context,
	principal identitydomain.Principal,
	id string,
) (Incident, error) {
	if !principal.Has(identitydomain.PermissionIncidentRead) {
		return Incident{}, ErrForbidden
	}
	return s.Store.Get(ctx, principal, id)
}

func (s Service) List(
	ctx context.Context,
	principal identitydomain.Principal,
	filter Filter,
) ([]Incident, error) {
	if !principal.Has(identitydomain.PermissionIncidentRead) {
		return nil, ErrForbidden
	}
	if filter.SiteID != "" && !principal.CanAccessSite(principal.OrganizationID, filter.SiteID) {
		return nil, ErrForbidden
	}
	return s.Store.List(ctx, principal, filter)
}

func (s Service) Resolve(
	ctx context.Context,
	principal identitydomain.Principal,
	key, incidentID string,
	command ResolveIncident,
) (Incident, bool, error) {
	if !principal.Has(identitydomain.PermissionIncidentResolve) {
		return Incident{}, false, ErrForbidden
	}
	if !validKey(key) || command.ExpectedVersion <= 0 ||
		strings.TrimSpace(command.ResolutionSummary) == "" {
		return Incident{}, false, ErrInvalid
	}
	return s.Store.Resolve(ctx, principal, key, incidentID, command)
}

func validKey(key string) bool {
	length := len(strings.TrimSpace(key))
	return length >= 8 && length <= 200
}
