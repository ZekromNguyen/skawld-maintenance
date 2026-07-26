# Change Log

All notable product architecture and implementation changes are recorded here. This project uses calendar dates while unreleased. SDK dependency versions are tracked separately in `go.mod` once implementation starts.

## [Unreleased]

### 2026-07-26 — Go standard-library security update

- Upgraded the minimum, CI, and container build toolchain from Go `1.25.7` to
  Go `1.25.12`.
- Cleared reachable standard-library findings reported by `govulncheck`,
  including `crypto/tls`, `crypto/x509`, `net/http`, `net/textproto`,
  `net/url`, `net`, and `os`.
- Verified the full race-enabled PostgreSQL/S3 suite and `govulncheck` with the
  exact `go1.25.12` toolchain.

### 2026-07-26 — Phase 5 pilot engineering baseline implemented

#### Added

- Tenant/site-scoped AI quality summaries, append-only reviewer labels,
  evidence coverage/retrieval precision counters, supervisor quality dashboard,
  and a frozen deterministic pilot evaluation CLI/dataset with fail-closed
  unsafe recommendation gates.
- Separate API/worker PostgreSQL connection budgets, per-job River concurrency,
  deadlines, bounded retry schedules, permanent/transient classification, and
  an OLTP contention load-check tool.
- Offline sync policy tests for duplicate, reordered, interrupted, retryable,
  stale-version, and independent attachment transfer behavior.
- Read-only EAM/CMMS connector/import boundary plus a path-confined,
  digest-cursored NDJSON snapshot adapter; no external write-back.
- PostgreSQL backup, checksum, empty-target restore, count/schema verification,
  tenant audit export, data retention, incident response, threat model, release,
  identity/object-store support, and single-server pilot runbooks.
- Pilot Compose profile, migrator and static web images, dependency SBOM tool,
  pinned GitHub Actions, dependency/image scan gates, tagged artifact
  provenance, and Windows/macOS backend evaluation packages.

#### Verified

- Migration upgrade to version 10 and append-only feedback privileges.
- Backup/restore rehearsal with schema and key-table count parity; the temporary
  restore database was removed after verification.
- Frozen safety evaluation passed with zero escaped unsafe candidates.
- Local API/worker contention test completed 1,600 operations with zero errors.
- Full race-enabled Go/PostgreSQL/S3 suite, vet/build, React lint/Vitest/Vite,
  OpenAPI validation, npm high-severity audit, and all four OCI builds passed.

#### Open qualification gates

- A named customer must still approve topology, data class, OIDC/S3, support,
  RPO/RTO, retention, and safety acceptance. Object-store recovery must be
  rehearsed against the selected product.
- Windows/macOS Flutter CI execution, code signing/notarization, protected
  signing identities, and clean-machine installation remain release gates.
- The result is a pilot engineering baseline, not an industrial standards,
  compliance, safety-authority, or autonomous-control claim.

### 2026-07-26 — Phase 4 governed workflow learning implemented

#### Added

- PostgreSQL workflow identities, immutable versions, applicability,
  append-oriented reviews/evaluations, and correction-derived improvement
  candidates in migration `00009`.
- A maintenance learning gateway contained in `internal/skawld`: approved
  demonstration loader with redaction overlays, deterministic extractor,
  read-only guidance tool catalog, SDK v0.2.0 stores/compiler/evaluation/
  publication adapters, and product-owned query DTOs.
- Multi-demonstration compilation with exact event evidence, source digest,
  sequence consistency, visible conflicts, behavioral diff, prerequisites,
  competencies, effective/review dates, and reproducible SDK payload.
- REST/OpenAPI and supervisor UI for candidate compilation, evidence review,
  approval/rejection, evaluation/publication, applicability expansion,
  applicable-workflow lookup, and retirement.

#### Safety and boundaries

- Only completed, approved, gap-free demonstrations from the same workflow,
  site, and asset class can compile; at least two distinct demonstrations are
  required.
- Unsupported or unclassified semantic actions are rejected. The tool catalog
  contains read-only maintenance guidance tools and no industrial-control
  capability.
- RBAC and `ApprovalAuthority` remain separate. Initial approval cannot expand
  applicability outside the learned site/asset class; expansion is a distinct
  authorized operation.
- Corrections create improvement candidates only. Published/retired executable
  payloads are database-immutable and production workflows never self-modify.
- Compilation remains synchronous and bounded for MVP volume; no new queue or
  service was introduced without measured need.

#### Verified

- PostgreSQL migration upgrade through version 9 and database immutability
  enforcement for published workflow payloads.
- Full Go vet and race-enabled PostgreSQL/S3 suite, including compile-review-
  evaluate-publish-expand-retire integration behavior.
- React TypeScript check, Vitest, Vite production build, and valid OpenAPI
  Phase 4 contract.
- Windows and macOS backend archives for amd64/arm64 with SHA-256 manifests.

### 2026-07-26 — Phase 3 semantic demonstration capture implemented

#### Added

- PostgreSQL implementation of SDK Observation v1 contained inside
  `internal/skawld`, with product-owned demonstration DTOs and services.
- Subject-aware domain-event outbox and durable capture deliveries. River
  processes them on a dedicated single-concurrency queue with bounded batches,
  stale-claim recovery, idempotent IDs, exponential retry, and visible health.
- Semantic mapping for execution work, measurements, decisions, evidence,
  recommendations, exact human corrections, outcomes, and shift handovers.
- Start/list/get/evidence-view/complete/review/redaction REST routes and a React
  timeline showing actor, trust, provenance, corrections, payload, and retry
  failures.
- Deterministic ingress sanitization plus append-oriented redaction overlays;
  governance requires scoped `ApprovalAuthority` separately from RBAC.
- P-302, shift-handover, and capture-failure PostgreSQL fixtures; migrations
  through version 8.
- Platform-selectable backend packaging, including a Windows/macOS-only target
  that produces x64/ARM64 archives and a complete SHA-256 manifest without
  requiring Linux builds.

#### Safety and boundaries

- Capture processing failure cannot roll back authoritative maintenance work;
  a gap remains visible and blocks demonstration completion.
- Runtime role cannot update/delete semantic events or reviews. No
  mouse-coordinate recorder, self-modification, CMMS expansion, or industrial
  control was introduced in Phase 3.

#### Verified

- Empty-database migration to version 8 on PostgreSQL/pgvector.
- Full Go vet and race-enabled PostgreSQL/S3 suite.
- React TypeScript check, Vitest, Vite build, and valid OpenAPI Phase 3 lint.
- Fedora Podman bind mounts now carry SELinux relabel options.
- Windows and macOS backend archives for amd64/arm64 with all checksums
  verified.

### 2026-07-26 — SDK v0.2.0 integration gate closed

#### Added

- Exact `github.com/ZekromNguyen/skawld-sdk-go v0.2.0` dependency with normal
  Go checksum verification and no committed `replace`.
- Trusted maintenance-principal to SDK tenant/actor/role mapping, canonical
  policy-safe SDK role identifiers, centralized product role permissions, and
  explicit maintenance-to-SDK risk mapping.
- Repository-wide import guard allowing SDK imports only in
  `internal/skawld` and the isolated SDK contract suite.
- Behavioral contracts for provider streaming, a strict maintenance-shaped
  idempotent tool, role/risk policy, approval pause/resume, requester/approver
  separation, deterministic workflow execution, audit emission, semantic
  demonstration recording, multi-demonstration learning, exact candidate
  review, evaluation gates, and publication.

#### Changed

- Closed ADR 0002 and Phase 0's SDK release gate.
- Principal resolution now retains trusted roles and does not merge roles or
  sites from another organization into the current single-organization
  session.
- Product role names containing spaces are translated only at the SDK boundary
  (`maintenance_supervisor`, `senior_technician`, and so on); maintenance
  vocabulary and stored role names remain unchanged.

#### Verified

- `go mod verify` and `go list -m` resolve the exact tag without replacement.
- Full Go vet/build and PostgreSQL/S3 race suite, including the new SDK
  contract package.
- Reachable vulnerability scanning reports no findings; API/worker OCI images
  and Windows/macOS amd64/arm64 backend packages still build after the SDK
  integration, with all package checksums verified.

### 2026-07-26 — Phase 2 evidence-first copilot implemented

#### Added

- Controlled industrial document metadata and revision lifecycle with
  authority, effective/expiry dates, applicability, supersession, retirement,
  attachment provenance, and approval-authority enforcement.
- Bounded PDF/text ingestion using a fixed Poppler command, stable chunk
  locators/checksums, versioned embeddings, transactional River enqueue,
  re-ingestion identity, failure classification, and a worker container with
  the required extractor.
- Authorization-first PostgreSQL eligible-set search combining FTS, exact
  pgvector cosine, explicit RRF, and same-site resolved incident history.
- Provider-neutral capability router, deterministic structured and embedding
  development providers, strict versioned output contracts, optional
  configured HTTP speech adapter, transcription candidate/verification flow,
  and a vision-candidate-only port.
- Typed incident recommendation API with evidence snapshots, assumptions,
  unknowns, confidence/risk, `INSUFFICIENT_EVIDENCE`, human feedback, and AI
  call/retrieval instrumentation.
- Structured maintenance report draft/edit/submit/approve lifecycle and shift
  handover draft/edit/submit/accept/acknowledge lifecycle with exact
  model/prompt/hash/evidence provenance and append-oriented audit/edit records.
- React supervisor views for controlled knowledge ingestion/status, evidence
  search, advisory recommendations, report drafts, and shift handover.
- Online-only evidence-backed recommendation panel in the Flutter
  Windows/macOS/field client; deterministic offline maintenance writes remain
  independent and durable.
- OpenAPI Phase 2 routes and migrations through version 5.

#### Safety

- Tenant, site, document approval/validity, and asset applicability are applied
  in SQL before ranking or AI context construction.
- Invalid JSON, unknown fields, invented evidence IDs, duplicated evidence,
  and recommendation risk above advisory fail closed without domain mutation.
- AI produces proposals/drafts/candidates only. Report, handover, document, and
  transcript authority remain explicit human transitions.
- No generic chat, machinery-control tool, PTW/LOTO authority, or autonomous
  workflow mutation was added.

#### Verified

- Go vet/build and full tests, including real PostgreSQL vertical tests for
  ingestion, filtered hybrid retrieval, report provenance/edit/approval, and
  shift handover provenance.
- S3 capability contract, React TypeScript/Vitest/production build, valid
  OpenAPI lint, and an empty-database migration through version 5 using the
  non-superuser schema owner after admin extension bootstrap.
- Windows amd64/arm64 and macOS amd64/arm64 backend engineering archives,
  archive contents, and SHA-256 manifests. Flutter Phase 2 source validation
  remains a native/mobile CI gate because the current shell has no Flutter SDK.
- Reachable Go vulnerability scan reported no findings; API and non-root
  Poppler-equipped worker OCI images build successfully.

#### Open gate

- Deterministic development providers must be replaced by deployment-selected
  adapters and evaluated before a pilot. Speech returns `503` unless a
  configured endpoint is provided.
- Native Windows/macOS CI artifacts still require a native CI run, signing, and
  macOS notarization before external distribution. The same CI run must execute
  Flutter analysis/tests for the new copilot panel.

### 2026-07-26 — Phase 1 core maintenance implemented

#### Added

- Source-aware asset hierarchy/components, approved criticality, incident
  lifecycle, and PostgreSQL repositories with tenant/site authorization,
  optimistic concurrency, idempotency, audit, and domain events.
- Pump inspection execution aggregate with assigned work, fixed inspectable
  steps, exact vibration/temperature measurements, observations, actions,
  semantic decisions, and external prerequisite evidence.
- Hard blocking of intrusive bearing inspection until current energy-isolation
  evidence is recorded by a separately privileged actor; Skawld does not own
  the permit or isolation.
- Narrow AWS SDK v2 S3 adapter and attachment lifecycle with server-generated
  keys, constrained signed upload, independent retries, SHA-256/size/magic-byte
  verification, and rejected/available states.
- Phase 1 REST handlers and OpenAPI 3.1 contract.
- React 19 + TypeScript + Vite + Tailwind supervisor workbench for native asset
  and incident creation, incident queue, pump execution, steps, and exact
  measurement entry.
- Flutter/Drift offline client source with assigned-execution pull, checklist,
  measurement/note capture, photo/audio manifest queue, atomic outbox,
  client/device/time identity, retry/conflict state, secure OIDC PKCE
  credentials, and connectivity-triggered synchronization.
- Native Windows and macOS Flutter build jobs and packaging scripts producing a
  portable Windows ZIP plus macOS DMG/ZIP with SHA-256 files. These are
  technician/workstation apps and remain unsigned engineering artifacts.
- Desktop runtime configuration loaded from a managed file or environment,
  plus macOS network/loopback/Keychain entitlements required by OIDC PKCE and
  secure credential storage.

#### Verified

- Goose migrations through version 3 on PostgreSQL/pgvector.
- Race-enabled PostgreSQL vertical integration for P-302 execution and safety
  prerequisite blocking.
- Required S3 adapter capability subset against Adobe S3Mock 5 after updating
  its v5 environment/health configuration.
- Go test/vet/build, TypeScript check, Vitest, Vite production build, and valid
  Redocly OpenAPI lint.
- Flutter `3.44.8` dependency resolution and lock, Drift generation, clean
  Flutter analysis, and Flutter contract tests.

#### Open gate

- Native Windows/macOS builds require their OS toolchains and remain CI gates;
  signing/notarization are deliberately not configured without release
  identities and protected secrets.

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
