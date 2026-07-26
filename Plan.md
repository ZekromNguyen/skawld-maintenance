# Skawld Maintenance Copilot — Implementation Plan

Status: **Phases 0–5 engineering baseline implemented; SDK v0.2.0 pinned; customer pilot qualification and native signing gates remain open**
Architecture source: [Spec.md](./Spec.md)  
Planning rule: Complete vertical behavior before adding infrastructure. Do not interpret a later phase as authorization to build it early.

## Implementation Phases

### Phase 0 — Foundation

**Goal**

Create a production-shaped but small Go product skeleton whose dependency rules, REST contract, persistence, identity boundary, and quality gates are executable.

**Deliverables**

- Go module pinned to an approved `skawld-sdk-go` `v0.x` tag; local sibling development through workspace-level `go.work`.
- API and worker composition roots over the same internal application packages; no API-to-worker HTTP boundary.
- Module boundary tests/lint rules and initial ADRs.
- PostgreSQL/pgvector migrations, transaction helper, health endpoints, configuration, structured logging, IDs, and clock.
- OIDC web-session/mobile-PKCE strategy, principal mapping, organization/site-scoped RBAC, and separate approval authority.
- `internal/skawld` anti-corruption boundary and import guard.
- Explicit `SourceOfTruth` and external projection contract.
- OpenAPI conventions, error model, idempotency support, audit sink, and Compose dependencies.
- CI for Go checks and container build; web/mobile directories only when their first behavior begins.

**Dependencies**

- Explicit user authorization to begin scaffold/implementation.
- Confirmed SDK tag and its actual store/tool/workflow APIs.
- Chosen local OIDC and S3 test implementations after license/support review.

**Acceptance Criteria**

- A clean checkout can start dependencies, migrate the database, run API/worker, and pass checks using documented commands.
- A protected endpoint maps an OIDC identity to a principal and refuses wrong-site access.
- One transactional command writes domain state and an audit event, and safely replays with the same idempotency key.
- CI does not rely on a local `replace` directive or public AI credentials.
- There is no domain import of SQL/HTTP/provider packages.
- No domain/application package outside `internal/skawld` imports `skawld-sdk-go`.
- API and worker use shared application services rather than calling one another.
- Role permission and approval authority are evaluated separately.

**Implementation status — 2026-07-26**

Implemented and verified:

- shared Go API/worker composition, configuration, graceful shutdown, structured
  logs, build metadata, dependency guard, and `internal/skawld` boundary;
- PostgreSQL/pgvector/goose and River migrations from an empty database,
  dedicated runtime pool budgets/timeouts, River schema/queues, and
  least-privilege runtime roles;
- Keycloak development realm, web Authorization Code + PKCE flow with
  browser-bound state, server-side opaque sessions, mobile PKCE contract,
  principal/membership mapping, RBAC, and independent `ApprovalAuthority`;
- `SourceOfTruth`, authorized site query, idempotent organization command, and
  domain state + membership + audit + replay result in one transaction;
- REST health/auth/identity slice, problem responses, OpenAPI, optional S3Mock
  contract-test profile, local runbook, CI, and non-root OCI API/worker images;
- native backend archives for Windows x64/ARM64, macOS Intel/Apple Silicon, and
  Linux x64/ARM64, with embedded build metadata and SHA-256 manifests;
- unit, architecture, race, concurrent-idempotency, transaction rollback,
  wrong-site, migration, and database-privilege checks.

SDK gate completed:

- `skawld-sdk-go v0.2.0` is pinned without `replace` and resolves through the
  standard Go module path;
- the anti-corruption layer maps trusted tenant/actor/roles and maintenance
  risk without leaking SDK types into product domain/application packages;
- SDK contract tests cover provider streaming, tool validation and
  idempotency, deterministic workflow execution, approval pause/resume and
  separation of duties, audit, semantic demonstrations, candidate
  compilation, review, deterministic evaluation, and publication.

### Phase 1 — Core Maintenance

**Goal**

Deliver the deterministic maintenance backbone for the pump demo, including a narrow offline field path.

**Deliverables**

- Organization/site membership, approval authority, flexible asset hierarchy, components, and approved criticality.
- `OWNED_BY_SKAWLD` versus `EXTERNAL_REFERENCE` on asset/incident/work projections.
- Incident aggregate and lifecycle.
- Maintenance execution aggregate with prerequisites, steps, measurements, observations, decisions, actions, outcome, and report state.
- Controlled engineering measurement/unit catalog and deterministic conversion.
- Attachment manifest, S3 transfer, quarantine/validation state.
- Minimal React supervisor screens and Flutter assigned-execution flow.
- Drift local schema, transactional outbox/inbox sync with client event/device/server receive timestamps, idempotency, and conflict UI.
- Append-oriented audit views for important actions.

**Dependencies**

- Phase 0.
- Product decisions for pump measurement types and fictional demo taxonomy.
- Approved mobile device/emulator development setup.

**Acceptance Criteria**

- Supervisor creates/imports P-302 and an incident, then assigns an execution.
- Technician downloads it, records steps/measurements/notes/photo metadata offline, and later syncs without duplication or data loss.
- Intrusive step is blocked when isolation verification is absent.
- Vibration and temperature values retain original units and pass deterministic validation/conversion tests.
- Attachment failure does not roll back structured work.
- No comprehensive work-order planning, inventory, permit, or telemetry subsystem exists.
- External asset/work/permit/telemetry projections cannot silently become Skawld-owned records.

**Implementation status — 2026-07-26**

Implemented:

- source-aware asset hierarchy/components and human-approved criticality;
- incident lifecycle and transactional domain-event/audit outbox;
- deterministic pump execution with optimistic versions, assigned-execution
  projection, safety-significant step blocking, and separately privileged
  external prerequisite verification;
- exact decimal vibration/temperature measurement validation and rational unit
  conversion, plus semantic observations, maintenance actions, and decisions;
- S3 capability adapter, server-generated object keys, constrained signed
  uploads, independent manifests, streaming SHA-256/size/magic-byte validation,
  and rejection states;
- Phase 1 REST/OpenAPI, React/Vite/Tailwind supervisor asset/incident/execution
  workbench, and production web build;
- Flutter/Drift source for assigned work, checklist, measurements, notes,
  photo/audio metadata, atomic outbox writes, stable client/idempotency/device
  identity, retry/conflict state, attachment store-and-forward, OIDC PKCE, and
  connectivity-triggered push/pull;
- Windows and macOS Flutter engineering-package jobs on native CI runners,
  producing a portable Windows ZIP and macOS DMG/ZIP with SHA-256 files.

Verified on the current Fedora host:

- migrations `00001`–`00003`, runtime least privilege, all Go tests/vet/build;
- PostgreSQL pump vertical integration including exact `8.1 mm/s`, offline
  mutation metadata, and LOTO blocking;
- required S3 operation subset against S3Mock 5;
- TypeScript check, Vitest, Vite production build, and valid OpenAPI lint;
- pinned Flutter `3.44.8`, resolved/locked dependencies, Drift generation,
  clean Flutter analysis, and Flutter contract tests.

Open verification gate:

- Windows/macOS Flutter builds require their native runner toolchains. Native
  build/package jobs are defined in CI but have not been executed in this
  Fedora session. Do not call the desktop distribution pilot-ready until those
  jobs, code signing, and macOS notarization pass.

### Phase 2 — Copilot

**Implementation status: core Phase 2 completed on 2026-07-26.**

**Goal**

Add evidence-backed document/history retrieval, safe recommendations, transcription, and approved report drafting.

**Deliverables**

- Revisioned document metadata, approval/applicability/supersession, object ingestion, extraction, chunking, FTS, and embeddings.
- Authorization-first eligible-set FTS and cosine search, explicit RRF fusion, and inspectable score/evidence results.
- Capability/model router and versioned structured-output schemas.
- `TranscriptionProvider`, `EmbeddingProvider`, LLM provider, and optional vision contracts with fakes.
- Contextual incident recommendation endpoint; `INSUFFICIENT_EVIDENCE`.
- Structured maintenance report draft, edits, submit/approve flow.
- Recommendation/model/retrieval/evaluation instrumentation.
- Shift handover draft and human accept/acknowledge flow.

**Dependencies**

- Phase 1 verified records and attachments.
- A small approved fictional document/history corpus.
- AI provider selected for development only; provider independence contract approved.

**Acceptance Criteria**

- Obsolete/inapplicable documents are excluded before ranking.
- Cross-tenant/site/applicability content is excluded in database queries before FTS/vector ranking or AI context construction.
- Every material recommendation cites resolvable evidence IDs and lists unknowns/risk.
- Invalid model JSON, invented evidence IDs, and cross-site references fail closed.
- Most automated tests use fakes; secret-gated provider contract tests are optional.
- Report and handover drafts retain exact model/prompt/evidence provenance and human edits.
- No generic chat endpoint or autonomous action path exists.

Implemented and verified:

- migrations `00004`–`00005` add controlled document, chunk/embedding,
  retrieval, recommendation/feedback, report provenance/edit, handover,
  transcription-candidate, and AI-call records;
- document ingestion commits revision state and River job atomically, uses a
  dedicated worker pool, bounded Poppler/text extraction, stable chunks,
  versioned embeddings, and safe re-ingestion job identity;
- a PostgreSQL integration test proves draft, expired, wrong-class, and
  cross-site documents are excluded before ranking while current applicable
  documents and same-site resolved incident history remain available;
- closed structured-output decoders reject unknown fields, invented evidence
  IDs, duplicated evidence IDs, and risks above advisory for recommendations;
- report and handover PostgreSQL vertical tests preserve model/prompt/input/
  output/evidence provenance and human edits/lifecycle transitions;
- optional speech uses a configured HTTP provider adapter and asynchronous
  River worker; no provider is assumed by default and every transcript remains
  an unverified candidate until a human verifies or rejects it;
- the complete schema migrates from an empty PostgreSQL database to version 5
  using a non-superuser owner after the DBA/container installs `pgvector`;
- backend engineering archives cross-build for Windows amd64/arm64 and macOS
  amd64/arm64 with verified SHA-256 manifests;
- reachable Go vulnerability scanning reports no findings and both API and
  non-root Poppler-equipped worker OCI images build;
- React supervisor surfaces cover controlled knowledge, evidence search,
  recommendations, report drafts, and shift handover; Flutter desktop/field UI
  exposes online-only advisory recommendations while preserving offline writes.

Open Phase 2 deployment gates:

- replace deterministic development structured/embedding providers with
  selected deployment adapters and run secret-gated contract/evaluation tests;
- provide an approved fictional document/history corpus and scoped approval
  authorities for a polished customer demo;
- execute native Windows/macOS Flutter package CI, code signing, and
  notarization;
- run Flutter analysis/tests for the Phase 2 desktop copilot panel; the current
  Fedora shell does not have Flutter on `PATH`.

### Phase 3 — Demonstration Capture

**Status: Implemented and verified on 2026-07-26.**

**Goal**

Capture how senior technicians perform both incident and normal work as trustworthy semantic demonstrations.

**Deliverables**

- PostgreSQL implementation/adaptation for the SDK observation store.
- Mapping from maintenance domain events to versioned semantic SDK events.
- Start/capture/complete demonstration use cases and UI.
- Decision, correction, evidence-view, step, and outcome capture.
- Demonstration review timeline with provenance and redaction controls.
- Shift handover and pump execution demonstration fixtures.

**Dependencies**

- Phase 1 domain events.
- Phase 2 evidence/recommendation identifiers.
- Confirmed SDK observation contracts.

**Acceptance Criteria**

- A full pump execution produces a coherent SDK demonstration without mouse-coordinate semantics.
- A normal shift handover also produces a demonstration.
- Events include actor, context, trust/verification, source IDs, and timestamps.
- Corrections link the exact recommendation/context to the human choice and later outcome.
- Capture failure cannot fail the authoritative maintenance transaction; a durable retry is visible and audited.

Implemented and verified:

- migrations `00006`–`00008` add subject-aware domain events, the SDK
  demonstration/event store, durable capture deliveries, append-oriented
  reviews/redactions, one active recording per subject, and hardened runtime
  privileges for immutable events/reviews;
- `internal/skawld` contains the only SDK Observation store, recorder adapter,
  semantic mapper, sanitizer, and retry processor; maintenance application
  types expose no SDK types;
- execution lifecycle, steps, prerequisites, measurements, observations,
  actions, decisions, recommendations/corrections, outcomes, and shift
  handovers emit versioned semantic source events;
- River runs a two-second semantic-capture sweep on a dedicated queue with one
  worker, bounded batches, idempotent event identity, stale-claim recovery,
  exponential retry, and visible per-demonstration capture health;
- REST endpoints and a dense supervisor timeline expose actor, source, trust,
  sensitivity, provenance, domain-event identity, correction links, review,
  redaction, and structured payload;
- review and redaction require both RBAC and scoped `ApprovalAuthority`;
  captured events and review records cannot be updated/deleted by the runtime
  role;
- PostgreSQL fixtures prove a coherent P-302 trace, a normal shift-handover
  trace, exact human correction linkage, later outcome capture, and a failed
  delivery that remains retryable without rolling back its domain event;
- clean migration through version 8, Go vet/race/integration suite,
  React/TypeScript/Vitest/Vite build, and valid OpenAPI Phase 3 contract pass.

### Phase 4 — Workflow Learning

**Status: Implemented and verified on 2026-07-26. Phase 5 has not started.**

**Goal**

Turn multiple reviewed demonstrations into a safe, evidence-linked, human-published workflow version.

**Deliverables**

- Maintenance learning extractor implementing the SDK boundary.
- Domain tool catalog and strict schemas.
- Deterministic multi-demonstration analysis and candidate compilation.
- Review UI showing supported steps, variants, ambiguity, evidence, and behavioral diff.
- Workflow applicability, prerequisites, competencies, effective/review dates, approval, publication, retirement, and `REVIEW_REQUIRED`.
- Improvement candidates from corrections; no self-modification.

**Dependencies**

- Phase 3 reviewed demonstrations.
- At least two coherent fictional demonstrations for one workflow.
- Phase 2 applicability/evidence and Phase 0 approval/audit.

**Acceptance Criteria**

- Compiler rejects unsupported tools, fabricated event references, invalid schema, and unsafe/unclassified actions.
- Ambiguous transitions are surfaced rather than invented away.
- Candidate cannot guide production execution until an authorized human publishes it.
- Applicability expansion requires a separate human approval.
- Published versions are immutable, effective-dated, supersedable, and reproducible.
- A human correction creates an improvement candidate only.

Implemented and verified:

- migration `00009` adds workflow identities, immutable versions,
  applicability, append-oriented reviews/evaluations, and improvement
  candidates while preserving the exact SDK candidate payload and digest;
- only completed, approved, gap-free demonstrations for the same site, asset
  class, and workflow key can be compiled, with at least two distinct source
  demonstrations required;
- `internal/skawld` owns the maintenance extractor, read-only guidance tool
  catalog, SDK stores, compiler, deterministic evaluation gates, publisher,
  and query mapping; product domain/application packages expose no SDK types;
- every common compiled step is backed by exact source event IDs, unsupported
  or unclassified actions are rejected, variants/conflicts remain visible, and
  human corrections create improvement candidates rather than executable
  steps;
- review and publication require RBAC and a separate scoped
  `ApprovalAuthority`; initial review cannot expand site/asset-class scope,
  while later applicability expansion is a separately authorized operation;
- published and retired executable payloads are database-immutable, versions
  carry effective/review dates and applicability, and retirement immediately
  removes a workflow from applicable guidance;
- REST/OpenAPI and the React supervisor workbench support compile, inspect,
  approve/reject/review-required, evaluate/publish, expand applicability, query
  applicable versions, and retire;
- Phase 4 compilation is deliberately synchronous and bounded for the MVP.
  Move it to the existing River worker only when measured latency or workload
  justifies asynchronous execution;
- migration upgrade through version 9, Go vet/race/integration suite,
  React/TypeScript/Vitest/Vite build, valid OpenAPI contract, and
  Windows/macOS backend packaging pass.

### Phase 5 — Pilot Readiness

**Goal**

Make the two demos deployable, supportable, measurable, and safe for a real maintenance-company pilot.

**Deliverables**

- Threat model, backup/restore, disaster-recovery rehearsal, migration/rollback runbook, data retention/export, and incident response.
- Hardened container images, SBOM, dependency/image scanning, secret rotation, and signed release plan.
- Load/concurrency/offline soak tests and object/database failure tests.
- Dedicated API/worker database pool budgets, per-job-type River concurrency/deadlines/retry classification, and OLTP-impact load tests.
- OIDC/SSO deployment guide and customer-selected S3/on-prem support matrix.
- Initial EAM/CMMS read/import adapter contract; no broad connector marketplace.
- Evaluation dataset and review dashboard for evidence, unsafe/unsupported outputs, correction, latency, and cost.
- Installation and operations documentation for a single-server/private deployment.

**Dependencies**

- Phases 0–4.
- Named pilot customer, actual deployment topology, data classification, identity provider, support expectations, and safety review.

**Acceptance Criteria**

- Restore rehearsal meets agreed pilot RPO/RTO.
- Cross-tenant/site authorization suite passes at API, application, repository, search, sync, attachment, and tool boundaries.
- Offline sync survives duplicate, reordered, interrupted, and stale-version operations.
- Pilot reviewer can trace every recommendation/report/workflow step to evidence and AI/human provenance.
- Unsafe recommendation test set fails closed; critical actions have no tool implementation.
- Installation does not require Kubernetes, Redis, Kafka, Elasticsearch, or public AI access as an architectural assumption.

**Implementation status — 2026-07-26**

Implemented and verified as the engineering baseline:

- migration 10 adds append-only reviewer quality labels and evidence/retrieval
  counts; live tenant/site-scoped quality metrics and a frozen deterministic
  evaluation suite fail closed on unsafe candidates;
- the supervisor dashboard exposes review and AI quality/safety metrics without
  creating a separate evaluation platform;
- API/worker PostgreSQL pools have an explicit shared connection budget; River
  jobs have per-kind concurrency, deadlines, bounded retry schedules,
  idempotency, and permanent/transient failure classification;
- mobile sync policy covers duplicates, deterministic reordering,
  interruption/backoff, stale-version conflicts, and independent attachment
  manifest/upload/completion retry behavior;
- a pull-only EAM/CMMS connector contract and immutable NDJSON snapshot adapter
  preserve tenant/site scope, provenance, source-of-truth, path confinement,
  and cursor/digest consistency; no write-back or connector marketplace exists;
- backup, checksum, empty-database restore, schema/count verification, tenant
  audit export, retention, incident response, threat model, OIDC/S3 support
  matrix, and single-server pilot runbooks exist;
- non-root/read-only OCI roles, Compose pilot profile, CycloneDX dependency
  SBOM, pinned GitHub Actions, dependency/image scan gates, tagged provenance,
  and release/signing gates are defined;
- local rehearsal restored schema version 10 and verified key-table counts;
  the local contention check completed 1,600 operations with zero errors;
- full Go race/PostgreSQL/S3 tests, vet/build, web lint/Vitest/Vite,
  OpenAPI validation, all four OCI image builds, frozen safety gates, SBOM, and
  Windows/macOS backend package checks pass.

Still requires a named customer before the word “pilot-ready” may be used:

- agreed topology, data classification, IdP/S3 selections, support contacts,
  RPO/RTO, retention, safety review, and customer acceptance;
- a restore drill measured against that agreed RPO/RTO and an object-storage
  backup/restore drill for the selected implementation;
- Windows Authenticode and macOS Developer ID/notarization, protected signing
  identities, and clean-machine installation tests;
- native Flutter analysis/build/package execution on Windows/macOS CI. Flutter
  is not installed in this Fedora verification environment.

## First 20 Engineering Tasks

### Task 1 — Record the accepted architecture baseline

**Goal**

Turn the accepted review into ADRs and identify remaining deployment-specific decisions without reopening the primary stack.

**Why now**

Code would otherwise freeze accidental assumptions about CMMS scope, identity, safety, or the SDK boundary.

**Files/modules**

`Spec.md`, `Plan.md`, `docs/adr/0001-architecture.md`, issue tracker.

**Implementation**

Record the accepted stack plus the `ApprovalAuthority`, `internal/skawld`, and `SourceOfTruth` boundaries. Decide module path, license, deployment target for the first demo, initial OIDC provider, S3 test target, and SDK tag. Create narrowly scoped follow-up ADRs for unresolved deployment choices.

**Acceptance Criteria**

- ADR marks the primary stack accepted; it is not reopened without new evidence.
- Safety boundary and “not a CMMS” scope are signed off.
- Approval authority, SDK isolation, and source-of-truth boundaries are explicit.
- Open decisions have owner/date, not implicit TODOs.
- No application scaffold is created by the architecture-document task; implementation starts only after an explicit request.

**Tests**

Document-link and Markdown lint check; manual decision checklist.

### Task 2 — Verify and pin the SDK integration contract

**Implementation status: completed on 2026-07-26 against
`skawld-sdk-go v0.2.0`.**

**Goal**

Prove the product can use the real SDK workflow, observation, learning, policy, approval, audit, provider, and tool APIs.

**Why now**

The SDK is `v0.x`; invented or unstable contracts would contaminate every later module.

**Files/modules**

`go.mod`, `internal/skawld/`, `test/contract/sdk/`, `docs/adr/0002-sdk-contract.md`; developer workspace `go.work` outside the repository.

**Implementation**

Pin an exact SDK tag. Write compile-time and behavioral contract tests for a maintenance-shaped fake tool, deterministic workflow execution, approval pause/resume, semantic demonstration recording, candidate compilation, and audit emission. Document gaps to fix generically in the SDK rather than patching around them in product code.

**Acceptance Criteria**

- CI resolves the tagged module with no `replace`.
- Local `go work` can use the sibling SDK.
- Contract tests pass against both local candidate and pinned tag before upgrade.
- No maintenance concept is added to the SDK.
- No domain/application package outside `internal/skawld` imports an SDK package or exposes an SDK type.

**Tests**

SDK contract suite with only fakes/in-memory SDK stores.

### Task 3 — Create the Go composition skeleton and dependency guardrails

**Goal**

Establish the minimal modular monolith layout and runnable process lifecycle.

**Why now**

All implementation needs stable dependency direction, configuration, cancellation, and process composition.

**Files/modules**

`cmd/api/`, `cmd/worker/`, shared `internal/` modules, `internal/platform/`, `internal/skawld/`, `go.mod`, `Makefile`.

**Implementation**

Create API/worker composition roots with signal-aware contexts and graceful shutdown over shared application packages. Add configuration parsing/validation, `slog`, clock/ID ports, build metadata, and import rules. Do not introduce HTTP between the process roles. Create only packages required for the first health path.

**Acceptance Criteria**

- Both binaries start and stop cleanly.
- Invalid configuration fails before serving.
- Domain packages have no infrastructure imports.
- Worker and API share application services; neither imports the other command nor calls the other over HTTP.
- Only `internal/skawld` may import the SDK.
- `make check` runs format, vet, tests, and dependency guard.

**Tests**

Configuration table tests, shutdown test, architecture/import test.

### Task 4 — Establish local dependencies and CI baseline

**Goal**

Make a clean Fedora/Linux checkout reproducible.

**Why now**

Database, identity, and object contracts need reliable integration environments before domain work.

**Files/modules**

`compose.yaml`, `deployments/compose/`, `.env.example`, `.github/workflows/ci.yml`, `scripts/`.

**Implementation**

Define PostgreSQL+pgvector and a local OIDC provider; select and pin an S3-compatible test target after license/security review. Add health checks and named volumes. CI runs Go checks and PostgreSQL integration tests; add dependency/security scanning without deployment automation.

**Acceptance Criteria**

- One documented command starts dependencies on Podman Compose and Docker Compose.
- No default credentials are suitable for production.
- CI is green on a clean checkout.
- Images are pinned deliberately and update ownership is documented.

**Tests**

Compose config validation, container health smoke test, CI dry run where available.

### Task 5 — Build PostgreSQL migration and transaction foundation

**Goal**

Create repeatable schema evolution and transaction boundaries.

**Why now**

Tenant scope, idempotency, audit, jobs, and aggregates all depend on correct transactions.

**Files/modules**

`migrations/`, `internal/platform/database/`, `internal/platform/database/testdb/`.

**Implementation**

Choose one migration tool; create extensions and baseline organization/site tables. Build `pgx` API/worker pool configurations with an explicit total connection budget, transaction runner, migration command, and isolated integration-test database helper. Set statement/lock timeouts and least-privilege runtime/migration roles.

**Acceptance Criteria**

- Migrations apply from empty and roll forward predictably.
- Runtime role cannot modify schema.
- Transaction rollback is verified.
- API and worker pool limits are independent and cannot jointly exceed configured database capacity.
- Tests run independently and clean up.

**Tests**

Migration-from-empty, schema smoke, commit/rollback, timeout, and role-permission integration tests.

### Task 6 — Implement OIDC, scoped RBAC, and approval authority

**Goal**

Authenticate through OIDC, authorize organization/site actions, and evaluate approval authority independently.

**Why now**

Every repository query, sync mutation, tool, and search result must start with trusted scope.

**Files/modules**

`internal/identity/domain/`, `internal/identity/application/`, `internal/platform/auth/`, migrations, HTTP middleware.

**Implementation**

Validate issuer/audience/signatures and map external subject to principal/membership. For web, terminate the OIDC authorization-code flow in Go and issue a short-lived server-side session cookie (`Secure`, `HttpOnly`, scoped `SameSite`); never store long-lived tokens in browser `localStorage`. Define the Flutter Authorization Code + PKCE contract. Implement named permissions/roles and a separate `ApprovalAuthority` grant with scope, risk ceiling, delegation, and validity. Build `/api/v1/me`.

**Acceptance Criteria**

- Valid local OIDC user receives correct scoped identity.
- Missing/invalid token and wrong issuer/audience fail.
- Role names are mapped to permissions centrally.
- Web session and Flutter PKCE strategies have separate contract tests.
- A role permission alone cannot approve; a current matching authority grant is also required.
- No handler trusts tenant/site IDs without resource authorization.

**Tests**

OIDC/session/PKCE contract tests, permission and approval-authority matrices, expiry/revocation/delegation, cross-site denial HTTP tests.

### Task 7 — Add idempotency and append-oriented audit

**Goal**

Make commands replay-safe and important actions traceable.

**Why now**

Mobile retry, worker retry, and AI/tool execution require these guarantees before writes proliferate.

**Files/modules**

`internal/platform/idempotency/`, `internal/platform/audit/`, migrations, HTTP command middleware.

**Implementation**

Store principal/scope/key/request hash/status/result atomically with commands. Reject key reuse with a different request. Implement audit sink with actor/action/entity/reason/provenance IDs and database privileges that deny update/delete to runtime. Keep operational failures/retries in structured logs and accountable domain/AI/view/approval/publication actions in the append-oriented audit stream.

**Acceptance Criteria**

- Duplicate identical command returns original result.
- Same key with different payload returns conflict.
- Domain state, audit, and idempotency result commit or roll back together.
- Audit records cannot be changed by the runtime role.
- An operational log entry is never treated as the authoritative audit record.

**Tests**

Concurrent duplicate integration test, payload mismatch, rollback, and audit privilege tests.

### Task 8 — Map SDK persistence and policy adapters to PostgreSQL

**Goal**

Provide durable SDK workflow/demonstration/approval/audit operation without duplicate authority.

**Why now**

The product must decide exactly which SDK-shaped data is stored before workflow-related tables spread.

**Files/modules**

`internal/skawld/stores/`, migrations, `test/contract/sdkstore/`, ADR.

**Implementation**

Inside `internal/skawld`, map actual SDK store interfaces to PostgreSQL. Reuse product audit/approval records through adapters where semantics match; otherwise keep clearly named SDK tables and cross-reference them. Implement tenant-safe wrappers even if the generic SDK interface does not know tenants. Translate product approval authority into SDK policy/approval decisions without leaking SDK types outward.

**Acceptance Criteria**

- Store contract suite passes.
- Tenant scope cannot be omitted by product callers.
- One canonical record owns each approval/audit fact.
- SDK SQLite storage is not used by production API/worker.

**Tests**

SDK store contracts, transaction, concurrency, tenant leakage, publish/version immutability tests.

### Task 9 — Implement asset hierarchy and criticality

**Goal**

Represent P-302 and flexible industrial hierarchy without building an asset-management suite.

**Why now**

Incidents, evidence, applicability, and permissions all need stable asset context.

**Files/modules**

`internal/asset/domain/`, `application/`, `adapter/postgres/`, migrations, asset REST handlers.

**Implementation**

Build Asset aggregate/value objects, typed relationships/components, human-approved criticality, and explicit `SourceOfTruth`. Enterprise assets are projections with external system/ID/version, sync state, mapping version, and controlled local enrichment; lightweight native mode is explicitly `OWNED_BY_SKAWLD`. Support list/search/get/create only for configured small-customer mode. Detect cycles for containment relationships.

**Acceptance Criteria**

- P-302 path and components are represented.
- Customer-defined hierarchy depth works.
- Cross-organization relation and containment cycle fail.
- LLM cannot assign approved criticality.
- An external projection cannot be mutated as if Skawld were its source of truth.

**Tests**

Domain invariants, cycle detection, repository, tenant scope, HTTP contract tests.

### Task 10 — Implement incident lifecycle

**Goal**

Create the minimal abnormal-condition record for the pump demo.

**Why now**

The first work execution and evidence context begins with an incident.

**Files/modules**

`internal/incident/domain/`, `application/`, `adapter/postgres/`, migrations, API.

**Implementation**

Define states and allowed transitions, severity, asset/site linkage, explicit source-of-truth/external version, occurrence/detection/resolution fields, and commands to create/update/resolve. Emit in-process domain events and audit. External incidents follow their integration contract rather than native mutation rules.

**Acceptance Criteria**

- Create/list/get/resolve P-302 incident works.
- Invalid transitions and wrong-site asset fail.
- Resolution requires an authorized actor and outcome summary/reference.
- Domain events occur only after successful commit through a post-commit/outbox mechanism.

**Tests**

State table tests, repository integration, authorization, idempotent HTTP tests.

### Task 11 — Implement measurement and unit model

**Goal**

Safely record maintenance measurements with deterministic engineering semantics.

**Why now**

The two demo measurements are central evidence and cannot be modeled as arbitrary float/string pairs.

**Files/modules**

`internal/execution/domain/measurement*.go`, unit catalog/conversion package, migrations.

**Implementation**

Use decimal values, controlled measurement type/unit compatibility, original value/unit, observed time, source, quality, component, instrument reference, and verification. Implement only vibration velocity and temperature conversions needed for the demo, with extension points based on catalog data.

**Acceptance Criteria**

- `8.1 mm/s` and `94 °C` persist exactly with source/verification.
- Incompatible units and non-finite/out-of-policy values fail.
- Conversion is deterministic and round-trip tolerances are defined.
- No LLM performs conversion or final validation.

**Tests**

Property/table conversion tests, precision/round-trip, invalid unit/type, repository tests.

### Task 12 — Implement maintenance execution aggregate and prerequisites

**Goal**

Represent actual inspection/troubleshooting work and block unsafe progression.

**Why now**

This is the deterministic spine for field UX, reports, demonstrations, and workflows.

**Files/modules**

`internal/execution/domain/`, `application/`, `adapter/postgres/`, migrations, execution API.

**Implementation**

Create execution, participant, step, prerequisite verification, observation, decision, action, and outcome models. Define state machine and commands. Model work order as optional `WorkReference` whose source of truth and write-back state are explicit. Add in-process domain event collection.

**Acceptance Criteria**

- Execution can start, record evidence/actions, complete steps, and finish.
- Intrusive step remains blocked without current isolation verification.
- Stale workflow version/prerequisite fails safely.
- No inventory, scheduling, permit authority, or machinery action model appears.

**Tests**

Aggregate transition tests, prerequisite matrix, concurrency/version conflict, repository/HTTP tests.

### Task 13 — Implement S3 attachment lifecycle

**Goal**

Store photos/audio/documents without coupling binary transfer to business transactions.

**Why now**

Field capture and document ingestion need the same secure object boundary.

**Files/modules**

`internal/platform/objectstore/`, `internal/execution/adapter/attachment/`, migrations, attachment API.

**Implementation**

Define the narrow capability-level `ObjectStore` port (`Put`, `Get`, `Delete`, `Head`, signed upload/download); generated object keys; initiate/complete upload; checksums, size/MIME/magic-byte validation, and quarantine/scan states. Use fake and one selected S3-compatible test adapter without making it the product default. Add multipart/versioning/retention only through a later requirement. Never use user filename as a path.

**Acceptance Criteria**

- Photo/audio manifest persists independently from upload.
- Interrupted retry does not duplicate attachment.
- MIME mismatch/oversize/checksum failure is rejected and audited.
- Cross-tenant object access and arbitrary object key access fail.
- Contract tests cover only the required operation subset and document adapter deviations.

**Tests**

Fake unit tests, required-operation S3 adapter contract tests, malicious filename/MIME/size, interrupted upload/retry, authorization tests.

### Task 14 — Define OpenAPI and build the minimal supervisor web

**Goal**

Provide a usable supervisor path without duplicating Go business logic.

**Why now**

Core maintenance behavior needs a human review surface before AI is introduced.

**Files/modules**

`api/openapi.yaml`, `web/`, generated TypeScript types/client.

**Implementation**

Document endpoints and problem details; generate types. Create React/Vite shell using only the Go server-session cookie (no token in `localStorage`), site context, asset/incident lists, incident detail, execution timeline, status/severity system, and accessible form/table primitives.

**Acceptance Criteria**

- Supervisor can perform the Phase 1 web flow.
- UI hides nothing as a substitute for server authorization.
- No Next.js server or backend business logic exists.
- Loading/error/empty/stale states and keyboard navigation are explicit.

**Tests**

OpenAPI lint/compatibility, Vitest component tests, Playwright happy/denied/error flows.

### Task 15 — Build Flutter/Drift offline execution slice

**Goal**

Let a technician execute the pump inspection through connectivity loss.

**Why now**

Offline behavior must shape the data/API design before copilot complexity.

**Files/modules**

`mobile/lib/core/`, `data/`, `sync/`, `features/execution/`, Drift migrations.

**Implementation**

Cache assigned execution/asset/workflow snapshot; write local rows and outbox atomically. Every mutation carries `client_event_id`, idempotency key, `device_id`, base server version, `created_at_device`, payload version, attempt count, and sync status; the server records `received_at_server` and result version. Implement push/pull cursors, optimistic conflicts, attachment queue, and sync status. Support checklist, measurements, notes, voice/photo capture metadata.

**Acceptance Criteria**

- Airplane-mode work survives process/device restart.
- Reconnect syncs exactly once and reports rejected/conflicted changes.
- Timeout followed by retry cannot duplicate a measurement or observation.
- Attachment retry is independent.
- Server-authoritative workflow/document/approval changes are not overwritten.

**Tests**

Drift migration/unit tests, offline integration scenario, duplicate/reorder/stale mutations, widget tests.

### Task 16 — Implement document validity and ingestion

**Goal**

Create an authoritative, revision-aware knowledge corpus.

**Why now**

RAG is unsafe until document currency, authority, and applicability exist.

**Files/modules**

`internal/knowledge/domain/`, `application/ingest/`, adapters, jobs, migrations.

**Implementation**

Build document/revision/applicability models, upload-to-quarantine flow, extraction interface, cleaning/chunking, stable locators, content hashes, FTS, and ingestion jobs. Start with text-bearing PDFs; mark OCR/scanned complexity explicitly. Keep Python optional behind the extractor port.

**Acceptance Criteria**

- Approved current revision is searchable.
- Obsolete/unapproved revision remains auditable but ineligible for guidance.
- Re-ingestion is idempotent by object/content/version.
- Malformed/encrypted/oversized document fails visibly and safely.

**Tests**

Validity/applicability domain tests, parser fixtures, job retry/idempotency, malicious PDF limits, retrieval eligibility tests.

### Task 17 — Implement embeddings and hybrid evidence retrieval

**Goal**

Find similar incidents and relevant approved knowledge with inspectable provenance.

**Why now**

The copilot must retrieve evidence before it can reason.

**Files/modules**

`internal/skawld/routing/`, `internal/knowledge/application/search/`, embedding adapter, migrations, eval fixtures.

**Implementation**

Define embedding port/model metadata and re-embedding state. Generate embeddings asynchronously through River using the dedicated worker pool and per-job concurrency/deadline/retry/idempotency policy. Build the eligible set using authorization, tenant, site, validity, authority, and applicability database predicates; run lexical and exact cosine retrieval; combine ranks with explicit RRF; return evidence locators and component ranks. Add a small judged fixture set. Benchmark HNSW filtered recall before enabling it.

**Acceptance Criteria**

- Query returns eligible evidence with lexical/vector/domain score components.
- Cross-site and obsolete evidence never appears.
- Dimension/model upgrade can coexist and resume.
- Similar-incident result explains matching factors without claiming causality.
- Security filtering occurs before ranked retrieval and AI context construction.
- If HNSW is enabled, exact-search recall comparison and iterative-scan/partial-index/partition decision are documented.

**Tests**

Fake embedding contract, PostgreSQL FTS/vector integration, eligibility security suite, deterministic ranking fixtures, re-embedding tests.

### Task 18 — Implement evidence-backed recommendations and report drafts

**Goal**

Deliver the safe copilot and report value loop.

**Why now**

Structured facts and trustworthy retrieval now exist.

**Files/modules**

`internal/skawld/schemas/`, `internal/skawld/prompts/`, `internal/skawld/routing/`, `internal/skawld/tools/`, recommendation/report application services and API.

**Implementation**

Build capability router, provider adapters, strict recommendation/report schemas, bounded evidence packet, evidence ID verification, risk/policy handling, and `INSUFFICIENT_EVIDENCE`. Persist retrieval/model/prompt provenance and feedback. Expose only maintenance tools; report generation produces a draft.

**Acceptance Criteria**

- Pump recommendation cites resolvable current evidence and unknowns.
- Invented citation, invalid JSON, timeout, or provider failure fails without domain mutation.
- Report draft can be edited/submitted/approved with full provenance.
- No critical or generic shell/control tool is registered.

**Tests**

Fake provider golden/semantic assertions, schema fuzzing, prompt-injection fixtures, evidence validation, policy/approval, provider contract tests opt-in.

### Task 19 — Capture semantic demonstrations and corrections

**Goal**

Record senior work as SDK demonstrations during normal product use.

**Why now**

The executed maintenance and recommendation contexts provide meaningful semantic events.

**Files/modules**

`internal/skawld/observers/`, demonstration application service/API, SDK PostgreSQL adapter, web review timeline.

**Implementation**

Map committed domain events to versioned SDK observations; durably retry capture; implement start/complete/review. Record accessed evidence, inspected measurements/history, decisions, actions, recommendation corrections, reason, and outcome. Add pump and handover fixtures.

**Acceptance Criteria**

- Demonstration timeline reconstructs domain meaning and provenance.
- Capture contains no primary mouse/keyboard coordinate semantics.
- Failed observation persistence is retryable and visible without rolling back maintenance work.
- Human correction links exact AI proposal, context, choice, and outcome.

**Tests**

Mapping contracts, ordering/idempotency, failure/retry, privacy/redaction, complete-trace fixtures.

### Task 20 — Compile, review, and publish the first workflow candidate

**Goal**

Prove the full differentiating loop from demonstrations to a reusable governed workflow.

**Why now**

All safety, evidence, domain, and SDK foundations are finally present.

**Files/modules**

`internal/skawld/extractors/`, `internal/skawld/tools/`, `internal/workflow/`, workflow review API/web, evaluation fixtures.

**Implementation**

Provide at least two reviewed pump demonstrations; run SDK deterministic analysis and compiler with maintenance extractor/catalog. Add applicability, prerequisites, competency, effective/review dates, behavioral diff, review, publication, retirement, and improvement-candidate paths.

**Acceptance Criteria**

- Candidate steps cite real demonstration events and registered safe tools.
- Ambiguity/conflicting sequence is shown to reviewer.
- Unauthorized/unreviewed candidate cannot publish or guide execution.
- Published version is immutable and only applicable contexts receive it.
- Correction proposes improvement without changing production workflow.
- Publication requires both workflow-publish permission and a current matching `ApprovalAuthority`; role title alone is insufficient.

**Tests**

Compiler/tool catalog contracts, fabricated evidence/unsafe tool rejection, review authorization, applicability matrix, immutable version, end-to-end pump loop.

## Planning Exit Rule

After Task 20, evaluate the product with a real maintenance company before expanding scope. The next backlog should be driven by observed workflow friction, retrieval/evidence quality, offline reliability, and safety review—not by adding infrastructure or generic CMMS features.
