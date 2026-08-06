package domain

import (
	"slices"
	"sort"
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

func TestPermissionsForRoleMatrix(t *testing.T) {
	t.Parallel()
	read := []string{
		"asset:read", "demonstration:read", "execution:read",
		"incident:read", "knowledge:read", "workflow:read",
	}
	expected := map[Role][]string{
		RoleAdministrator: matrixSet(read,
			"asset:create", "asset:criticality:approve", "attachment:write",
			"demonstration:capture", "demonstration:review", "execution:read:all",
			"execution:write", "handover:accept", "handover:write",
			"incident:create", "incident:resolve", "integration:external:import",
			"knowledge:approve", "knowledge:write", "organization:create",
			"execution:prerequisite:verify",
			"recommendation:review", "recommendation:run", "report:approve",
			"report:write", "workflow:publish", "workflow:review",
		),
		RoleMaintenanceSupervisor: matrixSet(read,
			"asset:create", "asset:criticality:approve", "attachment:write",
			"demonstration:capture", "demonstration:review", "execution:read:all",
			"execution:write", "handover:accept", "handover:write",
			"incident:create", "incident:resolve", "integration:external:import",
			"knowledge:approve", "knowledge:write", "execution:prerequisite:verify",
			"recommendation:review", "recommendation:run", "report:approve",
			"report:write", "workflow:review",
		),
		RoleSeniorTechnician: matrixSet(read,
			"attachment:write", "demonstration:capture", "demonstration:review",
			"execution:write", "handover:write", "incident:create",
			"execution:prerequisite:verify", "recommendation:run", "report:write",
			"workflow:review",
		),
		RoleTechnician: matrixSet(read,
			"attachment:write", "demonstration:capture", "execution:write",
			"handover:write", "recommendation:run", "report:write",
		),
		RoleManager: matrixSet(read,
			"demonstration:review", "handover:accept", "handover:write",
			"recommendation:review", "recommendation:run",
		),
	}
	for role, want := range expected {
		got := permissionSet(PermissionsForRole(role))
		if !slices.Equal(got, want) {
			t.Errorf("PermissionsForRole(%s) = %v, want %v", role, got, want)
		}
	}
}

// matrixSet builds a sorted, deduplicated permission set from the universal
// read set plus role-specific extras, matching permissionSet's output shape.
func matrixSet(read []string, extra ...string) []string {
	set := append(slices.Clone(read), extra...)
	sort.Strings(set)
	return set
}

// permissionSet returns the role's permissions as a sorted, deduplicated
// list of strings, so expected sets can be written literally.
func permissionSet(permissions []Permission) []string {
	unique := make(map[Permission]struct{}, len(permissions))
	for _, p := range permissions {
		unique[p] = struct{}{}
	}
	set := make([]string, 0, len(unique))
	for p := range unique {
		set = append(set, string(p))
	}
	sort.Strings(set)
	return set
}
