package auth

import (
	"context"
	"crypto/sha256"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/ZekromNguyen/skawld-maintenance/internal/identity/domain"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/clock"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/id"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	Pool  *pgxpool.Pool
	IDs   id.Generator
	Clock clock.Clock
	// EmailDomainAllowlist restricts federated sign-in to company email
	// domains. Empty disables the gate. Bootstrap and already-provisioned
	// principals are exempt.
	EmailDomainAllowlist []string
	// FederatedOrgID is the organization federated users with a valid
	// skawld_role claim are provisioned into (site-less membership).
	FederatedOrgID string
}

type Flow struct {
	Nonce        string
	PKCEVerifier string
	ReturnTo     string
	ExpiresAt    time.Time
}

type IdentityClaims struct {
	Subject     string
	DisplayName string
	Email       string
	// Role is the federated role attribute (skawld_role claim) used to
	// provision first-login company accounts. Empty when absent.
	Role string
}

func (r Repository) SaveFlow(
	ctx context.Context,
	state string,
	flow Flow,
) error {
	hash := sha256.Sum256([]byte(state))
	_, err := r.Pool.Exec(ctx, `
		INSERT INTO auth_flows (
			state_hash, nonce, pkce_verifier, return_to, expires_at, created_at
		) VALUES ($1, $2, $3, $4, $5, $6)
	`, hash[:], flow.Nonce, flow.PKCEVerifier, flow.ReturnTo, flow.ExpiresAt.UTC(), r.Clock.Now())
	if err != nil {
		return fmt.Errorf("save OIDC flow: %w", err)
	}
	return nil
}

func (r Repository) ConsumeFlow(ctx context.Context, state string) (Flow, error) {
	hash := sha256.Sum256([]byte(state))
	var flow Flow
	err := r.Pool.QueryRow(ctx, `
		DELETE FROM auth_flows
		WHERE state_hash = $1
		RETURNING nonce, pkce_verifier, return_to, expires_at
	`, hash[:]).Scan(&flow.Nonce, &flow.PKCEVerifier, &flow.ReturnTo, &flow.ExpiresAt)
	if err != nil {
		return Flow{}, fmt.Errorf("consume OIDC flow: %w", err)
	}
	if !r.Clock.Now().Before(flow.ExpiresAt) {
		return Flow{}, fmt.Errorf("OIDC flow expired")
	}
	return flow, nil
}

func (r Repository) ResolvePrincipal(
	ctx context.Context,
	claims IdentityClaims,
	bootstrapSubjects map[string]struct{},
) (domain.Principal, error) {
	if claims.Subject == "" {
		return domain.Principal{}, fmt.Errorf("OIDC subject is required")
	}
	now := r.Clock.Now()
	principalID := r.IDs.New()
	err := r.Pool.QueryRow(ctx, `
		INSERT INTO principals (
			id, external_subject, display_name, email, created_at, updated_at
		) VALUES ($1::uuid, $2, $3, nullif($4, ''), $5, $5)
		ON CONFLICT (external_subject) DO UPDATE
		SET display_name = EXCLUDED.display_name,
		    email = EXCLUDED.email,
		    updated_at = EXCLUDED.updated_at
		RETURNING id::text
	`, principalID, claims.Subject, claims.DisplayName, claims.Email, now).Scan(&principalID)
	if err != nil {
		return domain.Principal{}, fmt.Errorf("resolve principal: %w", err)
	}

	principal := domain.Principal{
		ID:              principalID,
		ExternalSubject: claims.Subject,
		DisplayName:     claims.DisplayName,
		Permissions:     make(map[domain.Permission]struct{}),
	}
	rows, err := r.Pool.Query(ctx, `
		SELECT organization_id::text, coalesce(site_id::text, ''), role
		FROM memberships
		WHERE principal_id = $1::uuid
		ORDER BY created_at
	`, principalID)
	if err != nil {
		return domain.Principal{}, fmt.Errorf("load principal memberships: %w", err)
	}
	defer rows.Close()
	siteSet := make(map[string]struct{})
	roleSet := make(map[domain.Role]struct{})
	for rows.Next() {
		var organizationID, siteID, role string
		if err := rows.Scan(&organizationID, &siteID, &role); err != nil {
			return domain.Principal{}, fmt.Errorf("scan principal membership: %w", err)
		}
		if principal.OrganizationID == "" {
			principal.OrganizationID = organizationID
		}
		// The current product session is scoped to one organization. Do not
		// merge roles or sites from another tenant into that session.
		if organizationID != principal.OrganizationID {
			continue
		}
		if siteID != "" {
			siteSet[siteID] = struct{}{}
		}
		trustedRole := domain.Role(role)
		permissions := domain.PermissionsForRole(trustedRole)
		if len(permissions) == 0 {
			continue
		}
		roleSet[trustedRole] = struct{}{}
		for _, permission := range permissions {
			principal.Permissions[permission] = struct{}{}
		}
	}
	if err := rows.Err(); err != nil {
		return domain.Principal{}, fmt.Errorf("iterate principal memberships: %w", err)
	}
	for siteID := range siteSet {
		principal.SiteIDs = append(principal.SiteIDs, siteID)
	}
	sort.Strings(principal.SiteIDs)
	for role := range roleSet {
		principal.Roles = append(principal.Roles, role)
	}
	sort.Slice(principal.Roles, func(i, j int) bool {
		return principal.Roles[i] < principal.Roles[j]
	})
	if _, ok := bootstrapSubjects[claims.Subject]; ok {
		principal.Permissions[domain.PermissionOrganizationCreate] = struct{}{}
	}
	// Federated (Google / Microsoft Entra) first-login provisioning. Only
	// runs for principals with no memberships that are not bootstrap
	// subjects. Existing seeded/provisioned accounts are untouched.
	if len(roleSet) == 0 {
		if _, isBootstrap := bootstrapSubjects[claims.Subject]; !isBootstrap {
			if err := r.provisionFederatedPrincipal(ctx, &principal, claims); err != nil {
				return domain.Principal{}, err
			}
		}
	}
	return principal, nil
}

// provisionFederatedPrincipal enforces the company-account allowlist and
// provisions a site-less membership from the IdP skawld_role attribute.
// The gate is inactive when no allowlist is configured (legacy behavior).
func (r Repository) provisionFederatedPrincipal(
	ctx context.Context,
	principal *domain.Principal,
	claims IdentityClaims,
) error {
	if len(r.EmailDomainAllowlist) == 0 {
		return nil
	}
	email := strings.ToLower(strings.TrimSpace(claims.Email))
	at := strings.LastIndex(email, "@")
	if at <= 0 {
		return fmt.Errorf("federated principal %q has no valid email for allowlist check", claims.Subject)
	}
	domainName := email[at+1:]
	allowed := false
	for _, allowedDomain := range r.EmailDomainAllowlist {
		if strings.EqualFold(strings.TrimSpace(allowedDomain), domainName) {
			allowed = true
			break
		}
	}
	if !allowed {
		return fmt.Errorf("email domain %q is not allowlisted for federated sign-in", domainName)
	}
	if r.FederatedOrgID == "" {
		return nil
	}
	role := domain.Role(strings.TrimSpace(claims.Role))
	if len(domain.PermissionsForRole(role)) == 0 {
		// Unknown or absent role attribute: authenticate with no access
		// rather than failing the request.
		return nil
	}
	now := r.Clock.Now()
	_, err := r.Pool.Exec(ctx, `
		INSERT INTO memberships (
			id, principal_id, organization_id, site_id, role, created_at
		)
		SELECT $1::uuid, $2::uuid, $3::uuid, NULL, $4, $5
		WHERE NOT EXISTS (
			SELECT 1 FROM memberships
			WHERE principal_id = $2::uuid
			  AND organization_id = $3::uuid
			  AND site_id IS NULL
			  AND role = $4
		)
	`, r.IDs.New(), principal.ID, r.FederatedOrgID, string(role), now)
	if err != nil {
		return fmt.Errorf("provision federated membership: %w", err)
	}
	principal.OrganizationID = r.FederatedOrgID
	principal.Roles = append(principal.Roles, role)
	for _, permission := range domain.PermissionsForRole(role) {
		principal.Permissions[permission] = struct{}{}
	}
	return nil
}

func (r Repository) CreateSession(
	ctx context.Context,
	token string,
	principalID string,
	expiresAt time.Time,
	idToken string,
) error {
	hash := sha256.Sum256([]byte(token))
	now := r.Clock.Now()
	_, err := r.Pool.Exec(ctx, `
		INSERT INTO web_sessions (
			token_hash, principal_id, expires_at, created_at, last_seen_at, id_token
		) VALUES ($1, $2::uuid, $3, $4, $4, nullif($5, ''))
	`, hash[:], principalID, expiresAt.UTC(), now, idToken)
	if err != nil {
		return fmt.Errorf("create web session: %w", err)
	}
	return nil
}

func (r Repository) PrincipalForSession(
	ctx context.Context,
	token string,
	bootstrapSubjects map[string]struct{},
) (domain.Principal, error) {
	hash := sha256.Sum256([]byte(token))
	var claims IdentityClaims
	err := r.Pool.QueryRow(ctx, `
		SELECT p.external_subject, p.display_name, coalesce(p.email, '')
		FROM web_sessions s
		JOIN principals p ON p.id = s.principal_id
		WHERE s.token_hash = $1
		  AND s.revoked_at IS NULL
		  AND s.expires_at > $2
		  AND p.status = 'ACTIVE'
	`, hash[:], r.Clock.Now()).Scan(&claims.Subject, &claims.DisplayName, &claims.Email)
	if err != nil {
		return domain.Principal{}, fmt.Errorf("load web session: %w", err)
	}
	return r.ResolvePrincipal(ctx, claims, bootstrapSubjects)
}

// SessionIDToken returns the OIDC id_token bound to an active web session so
// the API can pass it to the provider's end_session_endpoint as id_token_hint.
func (r Repository) SessionIDToken(ctx context.Context, token string) (string, error) {
	hash := sha256.Sum256([]byte(token))
	var idToken string
	err := r.Pool.QueryRow(ctx, `
		SELECT coalesce(id_token, '')
		FROM web_sessions
		WHERE token_hash = $1
		  AND revoked_at IS NULL
		  AND expires_at > $2
	`, hash[:], r.Clock.Now()).Scan(&idToken)
	if err != nil {
		return "", fmt.Errorf("load web session id_token: %w", err)
	}
	return idToken, nil
}

func (r Repository) RevokeSession(ctx context.Context, token string) error {
	hash := sha256.Sum256([]byte(token))
	_, err := r.Pool.Exec(ctx, `
		UPDATE web_sessions SET revoked_at = $2
		WHERE token_hash = $1 AND revoked_at IS NULL
	`, hash[:], r.Clock.Now())
	if err != nil {
		return fmt.Errorf("revoke web session: %w", err)
	}
	return nil
}
