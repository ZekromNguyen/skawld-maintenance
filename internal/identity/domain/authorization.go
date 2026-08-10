package domain

import (
	"context"
	"strings"
	"time"
)

type Permission string
type Role string

const (
	RoleAdministrator         Role = "Administrator"
	RoleMaintenanceSupervisor Role = "Maintenance Supervisor"
	RoleSeniorTechnician      Role = "Senior Technician"
	RoleTechnician            Role = "Technician"
	RoleManager               Role = "Manager"
)

const (
	PermissionOrganizationCreate   Permission = "organization:create"
	PermissionAssetRead            Permission = "asset:read"
	PermissionAssetCreate          Permission = "asset:create"
	PermissionCriticalityApprove   Permission = "asset:criticality:approve"
	PermissionIncidentRead         Permission = "incident:read"
	PermissionIncidentCreate       Permission = "incident:create"
	PermissionIncidentResolve      Permission = "incident:resolve"
	PermissionExecutionRead        Permission = "execution:read"
	PermissionExecutionReadAll     Permission = "execution:read:all"
	PermissionExecutionWrite       Permission = "execution:write"
	PermissionPrerequisiteVerify   Permission = "execution:prerequisite:verify"
	PermissionAttachmentWrite      Permission = "attachment:write"
	PermissionKnowledgeRead        Permission = "knowledge:read"
	PermissionKnowledgeWrite       Permission = "knowledge:write"
	PermissionKnowledgeApprove     Permission = "knowledge:approve"
	PermissionRecommendationRun    Permission = "recommendation:run"
	PermissionRecommendationReview Permission = "recommendation:review"
	PermissionReportWrite          Permission = "report:write"
	PermissionHandoverWrite        Permission = "handover:write"
	PermissionHandoverAccept       Permission = "handover:accept"
	PermissionDemonstrationRead    Permission = "demonstration:read"
	PermissionDemonstrationCapture Permission = "demonstration:capture"
	PermissionDemonstrationReview  Permission = "demonstration:review"
	PermissionWorkflowRead         Permission = "workflow:read"
	PermissionWorkflowReview       Permission = "workflow:review"
	PermissionWorkflowPublish      Permission = "workflow:publish"
	PermissionReportApprove        Permission = "report:approve"
	PermissionExternalImport       Permission = "integration:external:import"
	PermissionFieldManage          Permission = "field:manage"
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
	Roles           []Role
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

// PermissionsForRole is the single trusted role-to-capability mapping used by
// HTTP authorization and the SDK policy adapter. Unknown roles fail closed.
func PermissionsForRole(role Role) []Permission {
	read := []Permission{
		PermissionAssetRead,
		PermissionIncidentRead,
		PermissionExecutionRead,
		PermissionKnowledgeRead,
		PermissionDemonstrationRead,
		PermissionWorkflowRead,
	}
	switch role {
	case RoleAdministrator:
		return append(read,
			PermissionOrganizationCreate,
			PermissionAssetCreate,
			PermissionCriticalityApprove,
			PermissionIncidentCreate,
			PermissionIncidentResolve,
			PermissionExecutionWrite,
			PermissionExecutionReadAll,
			PermissionPrerequisiteVerify,
			PermissionAttachmentWrite,
			PermissionKnowledgeWrite,
			PermissionKnowledgeApprove,
			PermissionRecommendationRun,
			PermissionRecommendationReview,
			PermissionReportWrite,
			PermissionHandoverWrite,
			PermissionHandoverAccept,
			PermissionDemonstrationCapture,
			PermissionDemonstrationReview,
			PermissionWorkflowReview,
			PermissionWorkflowPublish,
			PermissionReportApprove,
			PermissionExternalImport,
			PermissionFieldManage,
		)
	case RoleMaintenanceSupervisor:
		return append(read,
			PermissionAssetCreate,
			PermissionCriticalityApprove,
			PermissionIncidentCreate,
			PermissionIncidentResolve,
			PermissionExecutionWrite,
			PermissionExecutionReadAll,
			PermissionPrerequisiteVerify,
			PermissionAttachmentWrite,
			PermissionKnowledgeWrite,
			PermissionKnowledgeApprove,
			PermissionRecommendationRun,
			PermissionRecommendationReview,
			PermissionReportWrite,
			PermissionHandoverWrite,
			PermissionHandoverAccept,
			PermissionDemonstrationCapture,
			PermissionDemonstrationReview,
			PermissionWorkflowReview,
			PermissionReportApprove,
			PermissionExternalImport,
		)
	case RoleSeniorTechnician:
		return append(read,
			PermissionIncidentCreate,
			PermissionExecutionWrite,
			PermissionPrerequisiteVerify,
			PermissionAttachmentWrite,
			PermissionRecommendationRun,
			PermissionReportWrite,
			PermissionHandoverWrite,
			PermissionDemonstrationCapture,
			PermissionDemonstrationReview,
			PermissionWorkflowReview,
		)
	case RoleTechnician:
		return append(read,
			PermissionExecutionWrite,
			PermissionAttachmentWrite,
			PermissionRecommendationRun,
			PermissionReportWrite,
			PermissionHandoverWrite,
			PermissionDemonstrationCapture,
		)
	case RoleManager:
		return append(read,
			PermissionRecommendationRun,
			PermissionRecommendationReview,
			PermissionHandoverWrite,
			PermissionHandoverAccept,
			PermissionDemonstrationReview,
		)
	default:
		return nil
	}
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
