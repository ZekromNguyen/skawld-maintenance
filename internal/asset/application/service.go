package application

import (
	"context"
	"errors"
	"strings"
	"time"

	assetdomain "github.com/ZekromNguyen/skawld-maintenance/internal/asset/domain"
	identitydomain "github.com/ZekromNguyen/skawld-maintenance/internal/identity/domain"
	integrationdomain "github.com/ZekromNguyen/skawld-maintenance/internal/integration/domain"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/keyset"
)

var (
	ErrForbidden       = errors.New("asset operation forbidden")
	ErrNotFound        = errors.New("asset not found")
	ErrInvalid         = errors.New("invalid asset command")
	ErrVersionConflict = errors.New("asset version conflict")
)

type ExternalReference struct {
	System  string `json:"system"`
	ID      string `json:"id"`
	Version string `json:"version,omitempty"`
}

type ComponentInput struct {
	Code string `json:"code"`
	Name string `json:"name"`
	Type string `json:"type"`
}

type CreateAsset struct {
	SiteID            string             `json:"site_id"`
	ParentAssetID     string             `json:"parent_asset_id,omitempty"`
	Tag               string             `json:"tag"`
	Name              string             `json:"name"`
	Class             string             `json:"class"`
	Manufacturer      string             `json:"manufacturer,omitempty"`
	Model             string             `json:"model,omitempty"`
	SourceOfTruth     string             `json:"source_of_truth"`
	ExternalReference *ExternalReference `json:"external_reference,omitempty"`
	Components        []ComponentInput   `json:"components,omitempty"`
}

type Asset struct {
	ID                string             `json:"id"`
	OrganizationID    string             `json:"organization_id"`
	SiteID            string             `json:"site_id"`
	ParentAssetID     string             `json:"parent_asset_id,omitempty"`
	Tag               string             `json:"tag"`
	Name              string             `json:"name"`
	Class             string             `json:"class"`
	Manufacturer      string             `json:"manufacturer,omitempty"`
	Model             string             `json:"model,omitempty"`
	Status            string             `json:"status"`
	SourceOfTruth     string             `json:"source_of_truth"`
	ExternalReference *ExternalReference `json:"external_reference,omitempty"`
	Version           int64              `json:"version"`
	CreatedAt         time.Time          `json:"created_at"`
	Components        []Component        `json:"components"`
	Criticality       *Criticality       `json:"criticality,omitempty"`
}

type Component struct {
	ID   string `json:"id"`
	Code string `json:"code"`
	Name string `json:"name"`
	Type string `json:"type"`
}

type Criticality struct {
	Rating              string    `json:"rating"`
	SafetyImpact        int       `json:"safety_impact"`
	ProductionImpact    int       `json:"production_impact"`
	EnvironmentalImpact int       `json:"environmental_impact"`
	FinancialImpact     int       `json:"financial_impact"`
	Redundancy          string    `json:"redundancy"`
	Rationale           string    `json:"rationale"`
	ApprovedBy          string    `json:"approved_by"`
	ApprovedAt          time.Time `json:"approved_at"`
}

type ApproveCriticality struct {
	Rating              string `json:"rating"`
	SafetyImpact        int    `json:"safety_impact"`
	ProductionImpact    int    `json:"production_impact"`
	EnvironmentalImpact int    `json:"environmental_impact"`
	FinancialImpact     int    `json:"financial_impact"`
	Redundancy          string `json:"redundancy"`
	Rationale           string `json:"rationale"`
}

type Filter struct {
	SiteID   string
	Query    string
	PageSize int
	Cursor   string
}

type Store interface {
	Create(
		ctx context.Context,
		principal identitydomain.Principal,
		key string,
		command CreateAsset,
	) (Asset, bool, error)
	Get(ctx context.Context, principal identitydomain.Principal, id string) (Asset, error)
	List(ctx context.Context, principal identitydomain.Principal, filter Filter) ([]Asset, bool, error)
	ApproveCriticality(
		ctx context.Context,
		principal identitydomain.Principal,
		idempotencyKey, assetID string,
		command ApproveCriticality,
	) (Criticality, bool, error)
}

type AuthorityReader interface {
	ForSubject(
		ctx context.Context,
		subjectID, organizationID, siteID string,
	) ([]identitydomain.ApprovalAuthority, error)
}

type Service struct {
	Store       Store
	Authorities AuthorityReader
}

func (s Service) Create(
	ctx context.Context,
	principal identitydomain.Principal,
	key string,
	command CreateAsset,
) (Asset, bool, error) {
	if !principal.Has(identitydomain.PermissionAssetCreate) ||
		!principal.CanAccessSite(principal.OrganizationID, command.SiteID) {
		return Asset{}, false, ErrForbidden
	}
	if !validKey(key) || strings.TrimSpace(command.Tag) == "" ||
		strings.TrimSpace(command.Name) == "" || strings.TrimSpace(command.Class) == "" {
		return Asset{}, false, ErrInvalid
	}
	source := integrationdomain.SourceOfTruth(command.SourceOfTruth)
	hasExternal := command.ExternalReference != nil &&
		strings.TrimSpace(command.ExternalReference.System) != "" &&
		strings.TrimSpace(command.ExternalReference.ID) != ""
	if err := source.Validate(hasExternal); err != nil {
		return Asset{}, false, errors.Join(ErrInvalid, err)
	}
	return s.Store.Create(ctx, principal, key, command)
}

func (s Service) Get(
	ctx context.Context,
	principal identitydomain.Principal,
	id string,
) (Asset, error) {
	if !principal.Has(identitydomain.PermissionAssetRead) {
		return Asset{}, ErrForbidden
	}
	return s.Store.Get(ctx, principal, id)
}

func (s Service) List(
	ctx context.Context,
	principal identitydomain.Principal,
	filter Filter,
) ([]Asset, string, error) {
	if !principal.Has(identitydomain.PermissionAssetRead) {
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
	return items, keyset.Encode(last.CreatedAt, last.ID), nil
}

func (s Service) ApproveCriticality(
	ctx context.Context,
	principal identitydomain.Principal,
	idempotencyKey, assetID string,
	command ApproveCriticality,
) (Criticality, bool, error) {
	if !validKey(idempotencyKey) || s.Authorities == nil {
		return Criticality{}, false, ErrInvalid
	}
	asset, err := s.Store.Get(ctx, principal, assetID)
	if err != nil {
		return Criticality{}, false, err
	}
	authorities, err := s.Authorities.ForSubject(
		ctx,
		principal.ID,
		principal.OrganizationID,
		asset.SiteID,
	)
	if err != nil {
		return Criticality{}, false, err
	}
	if !identitydomain.CanApprove(
		principal,
		identitydomain.PermissionCriticalityApprove,
		authorities,
		identitydomain.ApprovalRequest{
			OrganizationID: principal.OrganizationID,
			SiteID:         asset.SiteID,
			ScopeKind:      "asset",
			ScopeID:        assetID,
			Risk:           identitydomain.RiskAdvisory,
			At:             time.Now().UTC(),
		},
	) {
		return Criticality{}, false, ErrForbidden
	}
	value := assetdomain.Criticality{
		ID:                  "validated-by-store",
		OrganizationID:      principal.OrganizationID,
		AssetID:             assetID,
		Rating:              assetdomain.CriticalityRating(command.Rating),
		SafetyImpact:        command.SafetyImpact,
		ProductionImpact:    command.ProductionImpact,
		EnvironmentalImpact: command.EnvironmentalImpact,
		FinancialImpact:     command.FinancialImpact,
		Redundancy:          command.Redundancy,
		Rationale:           command.Rationale,
		ApprovedBy:          principal.ID,
		ApprovedAt:          time.Now(),
	}
	if _, err := assetdomain.NewCriticality(value); err != nil {
		return Criticality{}, false, errors.Join(ErrInvalid, err)
	}
	return s.Store.ApproveCriticality(ctx, principal, idempotencyKey, assetID, command)
}

func validKey(key string) bool {
	length := len(strings.TrimSpace(key))
	return length >= 8 && length <= 200
}
