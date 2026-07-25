package domain

import (
	"context"
	"strings"
	"time"
)

type Permission string

const (
	PermissionOrganizationCreate Permission = "organization:create"
	PermissionWorkflowReview     Permission = "workflow:review"
	PermissionWorkflowPublish    Permission = "workflow:publish"
	PermissionReportApprove      Permission = "report:approve"
)

type RiskLevel int

const (
	RiskInformational RiskLevel = iota
	RiskAdvisory
	RiskOperationalLow
	RiskSafetySignificant
	RiskCritical
)

type Principal struct {
	ID              string
	ExternalSubject string
	DisplayName     string
	OrganizationID  string
	SiteIDs         []string
	Permissions     map[Permission]struct{}
}

func (p Principal) Has(permission Permission) bool {
	_, ok := p.Permissions[permission]
	return ok
}

func (p Principal) CanAccessSite(organizationID, siteID string) bool {
	if p.OrganizationID == "" || p.OrganizationID != organizationID || siteID == "" {
		return false
	}
	if len(p.SiteIDs) == 0 {
		return true
	}
	for _, allowedSiteID := range p.SiteIDs {
		if allowedSiteID == siteID {
			return true
		}
	}
	return false
}

type ApprovalRequest struct {
	OrganizationID string
	SiteID         string
	ScopeKind      string
	ScopeID        string
	Competency     string
	Risk           RiskLevel
	At             time.Time
}

type ApprovalAuthority struct {
	ID                   string
	SubjectID            string
	OrganizationID       string
	SiteID               string
	ScopeKind            string
	ScopeID              string
	Competency           string
	MaximumRisk          RiskLevel
	ValidFrom            time.Time
	ValidUntil           time.Time
	RevokedAt            *time.Time
	DelegatedBySubjectID string
}

func (a ApprovalAuthority) Matches(subjectID string, request ApprovalRequest) bool {
	at := request.At.UTC()
	if a.RevokedAt != nil || a.SubjectID != subjectID || a.OrganizationID != request.OrganizationID {
		return false
	}
	if a.SiteID != "" && a.SiteID != request.SiteID {
		return false
	}
	if a.ScopeKind != "" && !strings.EqualFold(a.ScopeKind, request.ScopeKind) {
		return false
	}
	if a.ScopeID != "" && a.ScopeID != request.ScopeID {
		return false
	}
	if a.Competency != "" && !strings.EqualFold(a.Competency, request.Competency) {
		return false
	}
	if request.Risk > a.MaximumRisk {
		return false
	}
	if !a.ValidFrom.IsZero() && at.Before(a.ValidFrom.UTC()) {
		return false
	}
	if !a.ValidUntil.IsZero() && !at.Before(a.ValidUntil.UTC()) {
		return false
	}
	return true
}

func CanApprove(
	principal Principal,
	permission Permission,
	authorities []ApprovalAuthority,
	request ApprovalRequest,
) bool {
	if !principal.Has(permission) {
		return false
	}
	for _, authority := range authorities {
		if authority.Matches(principal.ID, request) {
			return true
		}
	}
	return false
}

type principalKey struct{}

func WithPrincipal(ctx context.Context, principal Principal) context.Context {
	return context.WithValue(ctx, principalKey{}, principal)
}

func PrincipalFromContext(ctx context.Context) (Principal, bool) {
	principal, ok := ctx.Value(principalKey{}).(Principal)
	return principal, ok && principal.ID != ""
}
