# Design: Company-only Google + Microsoft Entra login via Keycloak federation

Date: 2026-08-05
Status: Approved for implementation

## Problem

Skawld is sold as B2B industrial software. Customers should be able to sign in
with their **company Google Workspace** or **company Microsoft Entra (work or
school)** account, and **only company accounts** may authenticate. Federated
users need product roles without manual per-user DB work.

## Decisions (from brainstorming)

- **Option A: Keycloak identity-provider federation.** Keycloak remains the
  single IdP and session authority; Google and Microsoft are added as
  brokered identity providers. The API OIDC flow is unchanged.
- **Microsoft scope:** work/school accounts only, pinned to the customer's
  Entra **tenant ID** endpoint (never `/organizations`, which admits every
  company worldwide).
- **Google scope:** pinned to the customer's Google Workspace **hosted
  domain**.
- **App-level allowlist (defense in depth):** new `EMAIL_DOMAIN_ALLOWLIST`
  env; federated users whose email domain is not allowlisted are rejected
  (401). Bootstrap and seeded demo accounts are exempt.
- **Role provisioning:** IdP attribute `skawld_role` set per user in
  Keycloak; a mapper adds it to the id_token. The API grants the product
  role by creating an idempotent membership in the configured
  `FEDERATED_ORG_ID` organization (site NULL).
- **Credentials:** placeholders in the dev realm JSON; real client secrets
  are entered in the Keycloak admin console. Documented step-by-step.
  No code change needed when secrets are later supplied.

## Architecture

```text
User clicks "Continue with Google" / "Continue with Microsoft"
  -> Keycloak login page (identity providers configured)
  -> Google (hd=<workspace domain>) | Microsoft (tenant-id pinned)
  -> Keycloak creates/loads linked user, issues id_token (includes skawld_role)
  -> /auth/callback (existing flow) -> web session (existing)
  -> API /api/v1/* -> ResolvePrincipal:
       - email domain not in EMAIL_DOMAIN_ALLOWLIST (federated) -> 401
       - allowlisted, no membership -> create membership
         (org=FEDERATED_ORG_ID, site=NULL, role=skawld_role claim)
       - bootstrap/seed subjects -> unchanged
```

No changes to `oidc.go` OIDC exchange, PKCE, nonce, session creation, or
RP-initiated logout. The id_token stored per session is Keycloak's, so logout
continues to work for federated sessions.

## Changes

### 1. Keycloak realm (`deployments/compose/keycloak/skawld-realm.json`)

Add a top-level `identityProviders` array (a known, accepted realm field):

- `google`:
  - `clientId` / `clientSecret`: placeholders (`dev-google-client-id` /
    `dev-google-client-secret`)
  - `hostedDomain`: `example.com` placeholder
  - `authorizationUrl` / `tokenUrl` default Google endpoints
- `microsoft`:
  - `clientId` / `clientSecret`: placeholders
  - authorization URL pinned to
    `https://login.microsoftonline.com/<tenant-id>/oauth2/v2.0/authorize`
    (placeholder tenant id `00000000-0000-0000-0000-000000000000`)
  - `tokenUrl`: `https://login.microsoftonline.com/<tenant-id>/oauth2/v2.0/token`
- A `protocolMapper` on the `skawld-web` client to surface the `skawld_role`
  user attribute as an id_token claim.

Realm import uses `IGNORE_EXISTING`; applying IdP changes to an existing realm
requires a fresh Keycloak database (documented in the runbook).

### 2. API (`internal/platform/config/config.go`, `repository.go`)

- New `Auth` fields:
  - `EmailDomainAllowlist []string` from `EMAIL_DOMAIN_ALLOWLIST` (CSV)
  - `FederatedOrgID string` from `FEDERATED_ORG_ID`
- `ResolvePrincipal`:
  - Resolve the principal (existing upsert).
  - If the subject is a bootstrap subject, or the principal already has
    memberships, or the principal is a seeded demo account -> unchanged.
  - Otherwise (federated first login): if the email domain is not in the
    allowlist -> return an authorization error (authenticate -> 401).
    If allowlisted: read `skawld_role` from claims (extend
    `IdentityClaims`), and if present, idempotently create a membership
    (org = `FEDERATED_ORG_ID`, site NULL, role = claim).

Note: the role claim must be carried through `claimsFromToken` (web id_token
path) so `ResolvePrincipal` receives it.

### 3. Tests

- Integration tests (mirroring `repository_integration_test.go`,
  `TEST_DATABASE_URL`):
  - allowlist reject: federated user with disallowed domain -> error
  - allowlist + `skawld_role` attribute -> membership created, permissions
    granted
  - re-login idempotency: second resolution does not duplicate membership
  - seeded/bootstrap users unaffected when allowlist is set
- Realm JSON remains valid JSON (python json.load in verification).
- Keycloak boot + realm import smoke test.

### 4. Docs

- `docs/runbooks/local-development.md`: new section "Company SSO (Google /
  Microsoft Entra)":
  - Google Cloud Console: OAuth consent screen (Internal), OAuth client
    (Web), authorized redirect URI
    `http://localhost:8081/realms/skawld/broker/google/endpoint`
  - Entra: app registration, redirect URI
    `http://localhost:8081/realms/skawld/broker/microsoft/endpoint`,
    client secret
  - Keycloak admin: fill real secrets in Identity Providers > google /
    microsoft; set `skawld_role` user attribute per user
  - Env vars: `EMAIL_DOMAIN_ALLOWLIST`, `FEDERATED_ORG_ID`
  - Placeholders-to-secrets swap requires no code changes

## Out of scope (future)

- Per-tenant allowlist / multi-tenant org-per-domain provisioning
- SCIM / directory group sync for roles
- Custom Keycloak login theme with provider logos
- Google `hd` multi-domain handling (single domain per IdP config today)
