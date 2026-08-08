package application

import (
	"context"
	"errors"
	"strings"
	"time"

	identitydomain "github.com/ZekromNguyen/skawld-maintenance/internal/identity/domain"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/keyset"
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
	Details         string     `json:"details,omitempty"`
	Priority        string     `json:"priority"`
	Status          string     `json:"status,omitempty"`
	AssigneeID      string     `json:"assignee_id,omitempty"`
	ReporterID      string     `json:"reporter_id,omitempty"`
	TeamID          string     `json:"team_id,omitempty"`
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

type CloseIncident struct {
	ExpectedVersion int64 `json:"expected_version"`
}

type ReopenIncident struct {
	ExpectedVersion int64 `json:"expected_version"`
}

type Incident struct {
	ID                    string     `json:"id"`
	OrganizationID        string     `json:"organization_id"`
	SiteID                string     `json:"site_id"`
	AssetID               string     `json:"asset_id"`
	AssetTag              string     `json:"asset_tag,omitempty"`
	Number                string     `json:"number"`
	Summary               string     `json:"summary"`
	Details               string     `json:"details,omitempty"`
	Priority              string     `json:"priority"`
	Status                string     `json:"status"`
	AssigneeID            string     `json:"assignee_id,omitempty"`
	AssigneeName          string     `json:"assignee_name,omitempty"`
	ReporterID            string     `json:"reporter_id,omitempty"`
	ReporterName          string     `json:"reporter_name,omitempty"`
	TeamID                string     `json:"team_id,omitempty"`
	TeamName              string     `json:"team_name,omitempty"`
	SourceOfTruth         string     `json:"source_of_truth"`
	ExternalSystem        string     `json:"external_system,omitempty"`
	ExternalID            string     `json:"external_id,omitempty"`
	ExternalVersion       string     `json:"external_version,omitempty"`
	OccurredAt            *time.Time `json:"occurred_at,omitempty"`
	DetectedAt            time.Time  `json:"detected_at"`
	ResolvedAt            *time.Time `json:"resolved_at,omitempty"`
	ResolutionSummary     string     `json:"resolution_summary,omitempty"`
	TimeToCompleteSeconds *int64     `json:"time_to_complete_seconds,omitempty"`
	Version               int64      `json:"version"`
}

type Filter struct {
	SiteID   string
	AssetID  string
	Status   string
	PageSize int
	Cursor   string
}

type Store interface {
	Create(context.Context, identitydomain.Principal, string, CreateIncident) (Incident, bool, error)
	Get(context.Context, identitydomain.Principal, string) (Incident, error)
	List(context.Context, identitydomain.Principal, Filter) ([]Incident, bool, error)
	Resolve(context.Context, identitydomain.Principal, string, string, ResolveIncident) (Incident, bool, error)
	Close(context.Context, identitydomain.Principal, string, string, CloseIncident) (Incident, bool, error)
	Reopen(context.Context, identitydomain.Principal, string, string, ReopenIncident) (Incident, bool, error)
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
		strings.TrimSpace(command.Summary) == "" || command.DetectedAt.IsZero() ||
		!validPriority(command.Priority) {
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
) ([]Incident, string, error) {
	if !principal.Has(identitydomain.PermissionIncidentRead) {
		return nil, "", ErrForbidden
	}
	if filter.SiteID != "" && !principal.CanAccessSite(principal.OrganizationID, filter.SiteID) {
		return nil, "", ErrForbidden
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 25
	}
	if filter.PageSize > 100 {
		filter.PageSize = 100
	}
	if filter.Cursor != "" {
		key, err := keyset.Decode(filter.Cursor)
		if err != nil {
			return nil, "", ErrInvalid
		}
		filter.Cursor = key.Timestamp.UTC().Format(time.RFC3339Nano) + "|" + key.ID
	}
	items, hasMore, err := s.Store.List(ctx, principal, filter)
	if err != nil {
		return nil, "", err
	}
	if !hasMore || len(items) == 0 {
		return items, "", nil
	}
	last := items[len(items)-1]
	return items, keyset.Encode(last.DetectedAt, last.ID), nil
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

func (s Service) Close(
	ctx context.Context,
	principal identitydomain.Principal,
	key, incidentID string,
	command CloseIncident,
) (Incident, bool, error) {
	if !principal.Has(identitydomain.PermissionIncidentResolve) {
		return Incident{}, false, ErrForbidden
	}
	if !validKey(key) || command.ExpectedVersion <= 0 {
		return Incident{}, false, ErrInvalid
	}
	return s.Store.Close(ctx, principal, key, incidentID, command)
}

func (s Service) Reopen(
	ctx context.Context,
	principal identitydomain.Principal,
	key, incidentID string,
	command ReopenIncident,
) (Incident, bool, error) {
	if !principal.Has(identitydomain.PermissionIncidentResolve) {
		return Incident{}, false, ErrForbidden
	}
	if !validKey(key) || command.ExpectedVersion <= 0 {
		return Incident{}, false, ErrInvalid
	}
	return s.Store.Reopen(ctx, principal, key, incidentID, command)
}

func validKey(key string) bool {
	length := len(strings.TrimSpace(key))
	return length >= 8 && length <= 200
}

func validPriority(priority string) bool {
	switch priority {
	case "LOW", "MEDIUM", "HIGH", "CRITICAL":
		return true
	default:
		return false
	}
}
