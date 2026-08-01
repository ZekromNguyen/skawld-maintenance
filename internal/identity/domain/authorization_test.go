package domain

import (
	"testing"
	"time"
)

func TestApprovalRequiresPermissionAndAuthority(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 7, 26, 1, 0, 0, 0, time.UTC)
	request := ApprovalRequest{
		OrganizationID: "org-1",
		SiteID:         "site-a",
		ScopeKind:      "asset_class",
		ScopeID:        "centrifugal_pump",
		Competency:     "pumps",
		Risk:           RiskAdvisory,
		At:             now,
	}
	authority := ApprovalAuthority{
		SubjectID:      "principal-1",
		OrganizationID: "org-1",
		SiteID:         "site-a",
		ScopeKind:      "asset_class",
		ScopeID:        "centrifugal_pump",
		Competency:     "pumps",
		MaximumRisk:    RiskOperationalLow,
		ValidFrom:      now.Add(-time.Hour),
		ValidUntil:     now.Add(time.Hour),
	}

	noPermission := Principal{ID: "principal-1", Permissions: map[Permission]struct{}{}}
	if CanApprove(noPermission, PermissionWorkflowPublish, []ApprovalAuthority{authority}, request) {
		t.Fatal("authority without permission must not approve")
	}

	withPermission := Principal{
		ID: "principal-1",
		Permissions: map[Permission]struct{}{
			PermissionWorkflowPublish: {},
		},
	}
	if !CanApprove(withPermission, PermissionWorkflowPublish, []ApprovalAuthority{authority}, request) {
		t.Fatal("matching permission and authority should approve")
	}

	if CanApprove(withPermission, PermissionWorkflowPublish, nil, request) {
		t.Fatal("permission without authority must not approve")
	}
}

func TestApprovalAuthorityHonorsRiskAndValidity(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC()
	authority := ApprovalAuthority{
		SubjectID:      "principal-1",
		OrganizationID: "org-1",
		MaximumRisk:    RiskAdvisory,
		ValidFrom:      now.Add(-time.Hour),
		ValidUntil:     now.Add(time.Hour),
	}

	if authority.Matches("principal-1", ApprovalRequest{
		OrganizationID: "org-1",
		Risk:           RiskSafetySignificant,
		At:             now,
	}) {
		t.Fatal("authority must not exceed maximum risk")
	}

	if authority.Matches("principal-1", ApprovalRequest{
		OrganizationID: "org-1",
		Risk:           RiskAdvisory,
		At:             now.Add(2 * time.Hour),
	}) {
		t.Fatal("expired authority must not match")
	}
}

func TestCanAccessSite(t *testing.T) {
	t.Parallel()
	siteScoped := Principal{
		OrganizationID: "org-a",
		SiteIDs:        []string{"site-a"},
	}
	if !siteScoped.CanAccessSite("org-a", "site-a") {
		t.Fatal("expected matching site access")
	}
	if siteScoped.CanAccessSite("org-a", "site-b") {
		t.Fatal("wrong-site access must be denied")
	}
	if siteScoped.CanAccessSite("org-b", "site-a") {
		t.Fatal("cross-organization access must be denied")
	}
	organizationScoped := Principal{OrganizationID: "org-a"}
	if !organizationScoped.CanAccessSite("org-a", "site-b") {
		t.Fatal("organization-scoped membership should access its organization sites")
	}
}

func TestPermissionsForUnknownRoleFailsClosed(t *testing.T) {
	t.Parallel()
	if permissions := PermissionsForRole("Unknown"); len(permissions) != 0 {
		t.Fatalf("unknown role permissions = %v", permissions)
	}
	technician := PermissionsForRole(RoleTechnician)
	foundWrite := false
	foundWorkflowRead := false
	foundPublish := false
	for _, permission := range technician {
		foundWrite = foundWrite || permission == PermissionExecutionWrite
		foundWorkflowRead = foundWorkflowRead || permission == PermissionWorkflowRead
		foundPublish = foundPublish || permission == PermissionWorkflowPublish
	}
	if !foundWrite || !foundWorkflowRead || foundPublish {
		t.Fatalf("technician permissions = %v", technician)
	}
}

func TestExternalImportPermissionIsRestrictedToImportAuthorities(t *testing.T) {
	t.Parallel()
	for _, role := range []Role{RoleAdministrator, RoleMaintenanceSupervisor} {
		if !containsPermission(PermissionsForRole(role), PermissionExternalImport) {
			t.Fatalf("%s should have external import permission", role)
		}
	}
	for _, role := range []Role{
		RoleSeniorTechnician, RoleTechnician, RoleManager,
	} {
		if containsPermission(PermissionsForRole(role), PermissionExternalImport) {
			t.Fatalf("%s must not have external import permission", role)
		}
	}
}

func containsPermission(permissions []Permission, expected Permission) bool {
	for _, permission := range permissions {
		if permission == expected {
			return true
		}
	}
	return false
}
