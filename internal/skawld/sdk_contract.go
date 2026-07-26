package skawld

import (
	"context"
	"errors"
	"sort"

	identitydomain "github.com/ZekromNguyen/skawld-maintenance/internal/identity/domain"
	sdkcore "github.com/ZekromNguyen/skawld-sdk-go/core"
)

var ErrInvalidPrincipal = errors.New("maintenance principal cannot be mapped to SDK identity")

// Principal maps trusted application identity into the SDK isolation and
// policy identity. Tenant/actor/roles always come from authenticated server
// state, never workflow input or model output.
func Principal(value identitydomain.Principal) (sdkcore.Principal, error) {
	if value.ID == "" || value.OrganizationID == "" {
		return sdkcore.Principal{}, ErrInvalidPrincipal
	}
	roles := make([]string, 0, len(value.Roles))
	seen := make(map[identitydomain.Role]struct{}, len(value.Roles))
	for _, role := range value.Roles {
		if _, exists := seen[role]; exists ||
			len(identitydomain.PermissionsForRole(role)) == 0 {
			continue
		}
		seen[role] = struct{}{}
		roles = append(roles, RoleName(role))
	}
	sort.Strings(roles)
	return sdkcore.Principal{
		TenantID: value.OrganizationID,
		ActorID:  value.ID,
		Roles:    roles,
	}, nil
}

// AuthenticatedContext binds the exact mapped identity to context because SDK
// workflow operations reject an argument/context identity mismatch.
func AuthenticatedContext(
	ctx context.Context,
	value identitydomain.Principal,
) (context.Context, sdkcore.Principal, error) {
	principal, err := Principal(value)
	if err != nil {
		return ctx, sdkcore.Principal{}, err
	}
	return sdkcore.WithPrincipal(ctx, principal), principal, nil
}

// RoleCapabilities keeps SDK role policy aligned with the application's
// single role-to-permission source. ApprovalAuthority remains a separate
// product check and is deliberately not represented by this map.
func RoleCapabilities() map[string][]string {
	roles := []identitydomain.Role{
		identitydomain.RoleAdministrator,
		identitydomain.RoleMaintenanceSupervisor,
		identitydomain.RoleSeniorTechnician,
		identitydomain.RoleTechnician,
		identitydomain.RoleManager,
	}
	output := make(map[string][]string, len(roles))
	for _, role := range roles {
		permissions := identitydomain.PermissionsForRole(role)
		capabilities := make([]string, 0, len(permissions))
		for _, permission := range permissions {
			capabilities = append(capabilities, string(permission))
		}
		sort.Strings(capabilities)
		output[RoleName(role)] = capabilities
	}
	return output
}

// RoleName converts product display roles into the SDK's policy-safe stable
// identifiers without changing the maintenance role vocabulary.
func RoleName(role identitydomain.Role) string {
	switch role {
	case identitydomain.RoleAdministrator:
		return "administrator"
	case identitydomain.RoleMaintenanceSupervisor:
		return "maintenance_supervisor"
	case identitydomain.RoleSeniorTechnician:
		return "senior_technician"
	case identitydomain.RoleTechnician:
		return "technician"
	case identitydomain.RoleManager:
		return "manager"
	default:
		return ""
	}
}

// RiskLevel is the explicit application-to-SDK safety mapping. The mapping
// never lowers SafetySignificant or Critical work.
func RiskLevel(value identitydomain.RiskLevel) (sdkcore.RiskLevel, error) {
	switch value {
	case identitydomain.RiskInformational, identitydomain.RiskAdvisory:
		return sdkcore.RiskLow, nil
	case identitydomain.RiskOperationalLow:
		return sdkcore.RiskMedium, nil
	case identitydomain.RiskSafetySignificant:
		return sdkcore.RiskHigh, nil
	case identitydomain.RiskCritical:
		return sdkcore.RiskCritical, nil
	default:
		return "", errors.New("unknown maintenance risk level")
	}
}
