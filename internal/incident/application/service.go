package application

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	customfieldapp "github.com/ZekromNguyen/skawld-maintenance/internal/customfield/application"
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
	CustomValues    map[string]any `json:"custom_values,omitempty"`
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
	CustomValues          map[string]any `json:"custom_values,omitempty"`
	Version               int64      `json:"version"`
}

type CustomFieldRange struct {
	Min *float64
	Max *float64
}

type Filter struct {
	SiteID            string
	AssetID           string
	Status            string
	CustomFields      map[string]string
	CustomFieldRanges map[string]CustomFieldRange
	PageSize          int
	Cursor            string
}

type Store interface {
	Create(context.Context, identitydomain.Principal, string, CreateIncident) (Incident, bool, error)
	Get(context.Context, identitydomain.Principal, string) (Incident, error)
	List(context.Context, identitydomain.Principal, Filter) ([]Incident, bool, error)
	Resolve(context.Context, identitydomain.Principal, string, string, ResolveIncident) (Incident, bool, error)
	Close(context.Context, identitydomain.Principal, string, string, CloseIncident) (Incident, bool, error)
	Reopen(context.Context, identitydomain.Principal, string, string, ReopenIncident) (Incident, bool, error)
	UpdateCustomValues(context.Context, identitydomain.Principal, string, string, UpdateCustomValues) (Incident, bool, error)
}

// Fields resolves and validates custom field values against the tenant's
// field definitions before they reach the store.
type Fields interface {
	Definitions(ctx context.Context, organizationID, entityType string) ([]customfieldapp.Definition, error)
	ResolveAndValidate(ctx context.Context, organizationID, entityType string, values map[string]any) (map[string]any, error)
	ResolveKeys(ctx context.Context, organizationID, entityType string, keys []string) (map[string]customfieldapp.Definition, error)
}

// UpdateCustomValues merges the given definition-id-keyed values into the
// incident's custom_values map, subject to the optimistic lock.
type UpdateCustomValues struct {
	ExpectedVersion int64          `json:"expected_version"`
	CustomValues    map[string]any `json:"custom_values"`
}

type Service struct {
	Store  Store
	Fields Fields
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
	if len(command.CustomValues) > 0 {
		if s.Fields == nil {
			return Incident{}, false, errors.Join(ErrInvalid, errors.New("custom fields are not configured"))
		}
		normalized, err := s.Fields.ResolveAndValidate(ctx, principal.OrganizationID, "incident", command.CustomValues)
		if err != nil {
			return Incident{}, false, errors.Join(ErrInvalid, err)
		}
		command.CustomValues = normalized
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
	if len(filter.CustomFields) > 0 {
		if s.Fields == nil {
			return nil, "", errors.Join(ErrInvalid, errors.New("custom fields are not configured"))
		}
		keys := make([]string, 0, len(filter.CustomFields))
		for key := range filter.CustomFields {
			keys = append(keys, key)
		}
		resolved, err := s.Fields.ResolveKeys(ctx, principal.OrganizationID, "incident", keys)
		if err != nil {
			return nil, "", errors.Join(ErrInvalid, err)
		}
		byID := make(map[string]string, len(filter.CustomFields))
		rangesByID := make(map[string]CustomFieldRange)
		for key, value := range filter.CustomFields {
			def := resolved[key]
			if def.FieldType == "NUMBER" {
				r, isRange, err := parseNumberRange(value)
				if err != nil {
					return nil, "", errors.Join(ErrInvalid, err)
				}
				if isRange {
					rangesByID[def.ID] = r
					continue
				}
			}
			byID[def.ID] = value
		}
		filter.CustomFields = byID
		filter.CustomFieldRanges = rangesByID
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

// parseNumberRange parses "min:max" (either bound optional) into a range.
// Returns ok=false when the value is not a range at all; returns an error
// when it looks like a range but has unparseable bounds.
func parseNumberRange(value string) (CustomFieldRange, bool, error) {
	lo, hi, found := strings.Cut(value, ":")
	if !found {
		return CustomFieldRange{}, false, nil
	}
	var r CustomFieldRange
	if lo != "" {
		v, err := strconv.ParseFloat(lo, 64)
		if err != nil {
			return CustomFieldRange{}, false, fmt.Errorf("invalid range lower bound %q", lo)
		}
		r.Min = &v
	}
	if hi != "" {
		v, err := strconv.ParseFloat(hi, 64)
		if err != nil {
			return CustomFieldRange{}, false, fmt.Errorf("invalid range upper bound %q", hi)
		}
		r.Max = &v
	}
	return r, true, nil
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

func (s Service) UpdateCustomValues(
	ctx context.Context,
	principal identitydomain.Principal,
	key, incidentID string,
	command UpdateCustomValues,
) (Incident, bool, error) {
	if !principal.Has(identitydomain.PermissionIncidentCreate) {
		return Incident{}, false, ErrForbidden
	}
	if !validKey(key) || command.ExpectedVersion <= 0 {
		return Incident{}, false, ErrInvalid
	}
	if len(command.CustomValues) > 0 {
		if s.Fields == nil {
			return Incident{}, false, errors.Join(ErrInvalid, errors.New("custom fields are not configured"))
		}
		normalized, err := s.Fields.ResolveAndValidate(ctx, principal.OrganizationID, "incident", command.CustomValues)
		if err != nil {
			return Incident{}, false, errors.Join(ErrInvalid, err)
		}
		command.CustomValues = normalized
	}
	return s.Store.UpdateCustomValues(ctx, principal, key, incidentID, command)
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
