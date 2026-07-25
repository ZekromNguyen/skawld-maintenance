# Change Log

All notable product architecture and implementation changes are recorded here. This project uses calendar dates while unreleased. SDK dependency versions are tracked separately in `go.mod` once implementation starts.

## [Unreleased]

### 2026-07-26 — Windows and macOS engineering packages

#### Added

- Cross-platform Go packaging tool producing native API, worker, and migration
  binaries for Windows x64/ARM64, macOS Intel/Apple Silicon, and Linux
  x64/ARM64.
- ZIP archives for Windows, tar-gzip archives for macOS/Linux, embedded version
  metadata, packaged OpenAPI, operator README, and a SHA-256 checksum manifest.
- PowerShell and POSIX developer launchers plus dedicated Windows, macOS, and
  native-package runbooks.
- CI artifact job for main/tag pushes.

#### Boundary

- These are backend engineering archives, not desktop applications.
- Linux OCI remains the preferred production distribution.
- Windows artifacts are not Authenticode-signed; macOS artifacts are not signed
  or notarized. No MSI/PKG, OS service wrapper, or auto-update system was added.

### 2026-07-26 — Phase 0 foundation implemented

#### Added

- Go modular-monolith module with shared API and worker application code,
  signal-aware process lifecycle, structured logs, validated configuration,
  build metadata, IDs/clocks, and dependency import guards.
- Local Git repository initialized on `main`; no commit or remote was created.
- PostgreSQL/pgvector/goose foundation migration and River migration/worker
  role with a dedicated pool, explicit queue concurrency, timeouts, retries,
  error classification, and unique job arguments.
- Development Keycloak realm and OIDC Authorization Code + PKCE flow with
  browser-bound state, opaque server sessions, mobile bearer validation,
  principal/membership mapping, and a bootstrap installation subject.
- Separate RBAC permission and scoped/time/risk-bounded
  `ApprovalAuthority` models.
- Explicit `SourceOfTruth`, organization/site baseline, authorized site read,
  transactional idempotency, and append-oriented audit with runtime database
  privileges that deny audit mutation and schema creation.
- REST health/auth/identity endpoints, problem response model, and Phase 0
  OpenAPI contract.
- Compose dependencies with pinned PostgreSQL+pgvector and Keycloak versions;
  optional S3Mock test profile without selecting a production object-store
  server.
- Local Fedora/Linux runbook, CI checks, race/integration tests, and non-root
  OCI API/worker images.

#### Verified

- Migration from a fresh Compose volume creates pgvector and River schemas.
- Concurrent identical organization commands create exactly one organization
  and one audit event; a changed payload with the same key conflicts.
- Transaction rollback, cross-site denial, runtime least privilege, API/worker
  startup and graceful shutdown, OIDC discovery/login redirect, liveness,
  readiness, unit tests, race tests, vet, builds, Compose config, and both
  container builds.

#### Open gate

- Phase 0 remains incomplete until `skawld-sdk-go` publishes a clean compatible
  tag. The existing `v0.1.0` has the old module path and lacks the required
  workflow/observation/learning/policy/audit APIs; the required APIs only exist
  in a dirty sibling working tree. Maintenance therefore has no SDK dependency,
  `replace`, or copied SDK contracts yet. See ADR 0002.
- Repository license remains an explicit owner decision.

### 2026-07-26 — Architecture review amendments accepted

#### Changed

- Marked the primary technology stack accepted; scaffold still requires an explicit implementation request.
- Clarified that API and worker are independently scalable process roles over the same domain/application code and never separate applications connected by HTTP.
- Required a dedicated PostgreSQL pool for workers, explicit total connection budgeting, per-job-type River concurrency, deadlines, idempotency, retry classification, and an OLTP-impact trigger for reevaluating the queue architecture.
- Defined V1 hybrid retrieval as authorization-first eligible-set PostgreSQL FTS plus exact cosine pgvector search and explicit Reciprocal Rank Fusion. No model reranker is selected.
- Required exact-search recall comparisons before HNSW, with iterative scans, partial indexes, or partitioning evaluated only from measured filtered-query behavior.
- Split ordinary RBAC permissions from scoped, risk-bounded, time-valid `ApprovalAuthority`.
- Finalized browser auth as an OIDC-backed Go server session using a secure HttpOnly cookie; Flutter uses Authorization Code + PKCE.
- Narrowed `ObjectStore` to the operations Skawld needs; an S3 adapter no longer implies complete S3 compatibility or a default storage server.
- Moved every SDK import behind `internal/skawld`; maintenance domain/application packages expose no SDK types.
- Added explicit `SourceOfTruth` classification (`OWNED_BY_SKAWLD` or `EXTERNAL_REFERENCE`) and projection/write-back metadata for assets, incidents, work orders, permits, inventory, and telemetry.
- Strengthened offline mutation identity with client event/idempotency/device/device-time/server-receive fields.
- Clarified operational logs versus append-oriented accountable audit events.

#### Added

- `ApprovalAuthority` aggregate/table/API and independent authorization tests.
- Source-of-truth matrix and external projection rules.
- Import guard and SDK anti-corruption-layer contract.
- RRF and filtered-vector-recall acceptance criteria.
- River resource-governance requirements.

#### Status

- Architecture accepted with amendments.
- No scaffold, source code, dependency manifest, migration, or Compose service created.

### 2026-07-26 — Initial architecture proposal

#### Added

- Product definition as an **industrial maintenance intelligence layer**.
- Core principle: learn from normal industrial work while preserving safety, evidence, authority, and human control.
- Proposed Go modular-monolith backend with separate API and worker process roles.
- PostgreSQL as the single server database, with PostgreSQL full-text search and pgvector.
- PostgreSQL-backed River jobs; no Redis or external broker in V1.
- REST/OpenAPI public contract and Go application interfaces internally.
- React + TypeScript + Vite supervisor web recommendation.
- Tailwind/Radix/curated shadcn/TanStack UI recommendation.
- Flutter + Drift/SQLite offline-first technician application recommendation.
- S3-compatible object-storage port for cloud and on-premise deployments.
- External OIDC authentication and application-owned organization/site RBAC and approval authority.
- Evidence-first AI architecture, model capability routing, structured outputs, provider independence, and evaluation instrumentation.
- Explicit `ObservationCandidate` flow for vision and verification/provenance flow for transcription.
- Semantic demonstration, correction, candidate compilation, applicability, validity, and immutable publication model.
- Flexible typed asset hierarchy, approved criticality, deterministic measurement/unit model, and high-frequency telemetry boundary.
- Shift handover as the first enterprise-facing demo, followed by the high-vibration pump diagnosis demo.
- Conceptual schema, minimum REST API, SDK integration shape, Phases 0–5, and the first 20 tasks.
- Security threat controls, process-safety prerequisites, OT edge separation, and append-oriented audit boundary.
- Product monorepo decision for Go/web/mobile while `skawld-sdk-go` remains separate.

#### Changed from the initial hypothesis

- Selected **React + Vite**, not Next.js, because the initial product is an authenticated operations client with no SSR/SEO need; Go remains the only backend.
- Selected the **S3 API contract**, but did not select MinIO Community as the default. Its upstream repository was archived in April 2026 and its AGPL/source-only/support posture requires explicit deployment review. Development/on-premise storage must be selected and pinned after legal, security, backup, compatibility, and support assessment.
- Narrowed native work-order functionality to a lightweight `WorkReference`/draft. Enterprise work orders remain owned by EAM/CMMS integrations.
- Cut broad technician administration, analytics, vision diagnosis, full reliability engineering, preventive scheduling, inventory, permit authority, and high-volume telemetry from the MVP.
- Put document validity, evidence eligibility, and offline deterministic execution before workflow learning.

#### Confirmed from actual `skawld-sdk-go` inspection

- The SDK already provides provider/tool contracts, deterministic immutable workflow execution, semantic observation/demonstration capture, multi-demonstration learning/compiler boundaries, risk policy/approval checkpoints, audit contracts, and SQLite reference stores.
- Maintenance will implement/adapt production PostgreSQL stores and maintenance tool/observer/extractor/policy behavior instead of duplicating SDK runtimes.
- Workflow applicability, industrial prerequisites, evidence authority, and domain review remain maintenance-owned.

#### Safety

- Critical industrial actions are outside executable Skawld scope.
- No LLM output may directly reach equipment, safety systems, domain state, or tools.
- Published workflows never self-modify; corrections create improvement candidates.
- Standards are architectural references only; no compliance claim is made.

#### Status

- Documentation only.
- At this proposal point, architecture awaited owner review; superseded by the accepted amendments above.
- No application scaffold, dependency manifest, migration, Compose service, or source code has been created.
