# ADR 0001: Accepted Phase 0 Foundation

Status: Accepted  
Date: 2026-07-26

## Decision

Skawld Maintenance is one Go modular monolith. `cmd/api` and `cmd/worker` are independently scalable process roles over shared `internal/` domain and application packages. They never communicate through a private HTTP API.

The server database is PostgreSQL with pgvector. River uses PostgreSQL through a separately budgeted worker pool. The public application API is REST/OpenAPI. Web is React/Vite and mobile is Flutter/Drift, introduced only when their phase starts.

Authentication is external OIDC. Web uses a Go-terminated authorization-code flow and opaque server session cookie. Mobile uses Authorization Code + PKCE. RBAC permissions and `ApprovalAuthority` are separate domain checks.

Only `internal/skawld` may import `skawld-sdk-go`. External projections explicitly declare `OWNED_BY_SKAWLD` or `EXTERNAL_REFERENCE`.

## Consequences

- API and worker can scale independently without becoming services.
- PostgreSQL connection budgets and worker concurrency are configuration invariants.
- Domain/application packages remain stable while the SDK is pre-v1.
- No application code may infer that a synchronized entity is Skawld-owned.

