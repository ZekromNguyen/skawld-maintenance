package skawld

import (
	"context"
	"errors"
	"reflect"
	"testing"

	identitydomain "github.com/ZekromNguyen/skawld-maintenance/internal/identity/domain"
	sdkcore "github.com/ZekromNguyen/skawld-sdk-go/core"
	sdkpolicy "github.com/ZekromNguyen/skawld-sdk-go/policy"
)

func TestPrincipalMappingUsesTrustedTenantActorAndRoles(t *testing.T) {
	t.Parallel()
	input := identitydomain.Principal{
		ID:             "technician-1",
		OrganizationID: "organization-1",
		Roles: []identitydomain.Role{
			identitydomain.RoleSeniorTechnician,
			identitydomain.RoleSeniorTechnician,
			"Unknown",
		},
	}
	ctx, mapped, err := AuthenticatedContext(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if mapped.TenantID != input.OrganizationID || mapped.ActorID != input.ID {
		t.Fatalf("mapped principal = %+v", mapped)
	}
	if !reflect.DeepEqual(mapped.Roles, []string{"senior_technician"}) {
		t.Fatalf("mapped roles = %v", mapped.Roles)
	}
	fromContext, ok := sdkcore.PrincipalFromContext(ctx)
	if !ok || !reflect.DeepEqual(fromContext, mapped) {
		t.Fatalf("context principal = %+v, ok=%v", fromContext, ok)
	}
}

func TestPrincipalMappingFailsClosedWithoutIsolationIdentity(t *testing.T) {
	t.Parallel()
	_, err := Principal(identitydomain.Principal{ID: "technician-1"})
	if !errors.Is(err, ErrInvalidPrincipal) {
		t.Fatalf("Principal error = %v, want %v", err, ErrInvalidPrincipal)
	}
}

func TestSDKRolePolicyUsesApplicationPermissionMapping(t *testing.T) {
	t.Parallel()
	policy, err := sdkpolicy.NewRolePolicy(sdkpolicy.RolePolicyOptions{
		RoleCapabilities: RoleCapabilities(),
	})
	if err != nil {
		t.Fatal(err)
	}
	principal := sdkcore.Principal{
		TenantID: "organization-1",
		ActorID:  "technician-1",
		Roles:    []string{RoleName(identitydomain.RoleTechnician)},
	}
	allowed, err := policy.Evaluate(context.Background(), sdkpolicy.Action{
		Principal: principal,
		Descriptor: sdkcore.ToolDescriptor{
			Risk:        sdkcore.RiskLow,
			SideEffect:  sdkcore.SideEffectNone,
			Idempotency: sdkcore.IdempotencyNotApplicable,
			Permissions: []string{string(identitydomain.PermissionExecutionWrite)},
		},
	})
	if err != nil || allowed.Kind != sdkpolicy.Allow {
		t.Fatalf("technician execution decision = %+v, err=%v", allowed, err)
	}
	denied, err := policy.Evaluate(context.Background(), sdkpolicy.Action{
		Principal: principal,
		Descriptor: sdkcore.ToolDescriptor{
			Risk:        sdkcore.RiskLow,
			SideEffect:  sdkcore.SideEffectNone,
			Idempotency: sdkcore.IdempotencyNotApplicable,
			Permissions: []string{string(identitydomain.PermissionWorkflowPublish)},
		},
	})
	if err != nil || denied.Kind != sdkpolicy.Deny {
		t.Fatalf("technician publish decision = %+v, err=%v", denied, err)
	}
}

func TestRiskMappingNeverLowersSafetySignificantWork(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input identitydomain.RiskLevel
		want  sdkcore.RiskLevel
	}{
		{identitydomain.RiskInformational, sdkcore.RiskLow},
		{identitydomain.RiskAdvisory, sdkcore.RiskLow},
		{identitydomain.RiskOperationalLow, sdkcore.RiskMedium},
		{identitydomain.RiskSafetySignificant, sdkcore.RiskHigh},
		{identitydomain.RiskCritical, sdkcore.RiskCritical},
	}
	for _, test := range tests {
		got, err := RiskLevel(test.input)
		if err != nil || got != test.want {
			t.Fatalf("RiskLevel(%d) = %q, %v; want %q", test.input, got, err, test.want)
		}
	}
}
