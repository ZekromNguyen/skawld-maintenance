# Engineering Agent Guide

This file governs contributions to `skawld-maintenance` by human developers and AI coding agents. If a repository-level `AGENTS.md` is later introduced for tool discovery, it should point to or preserve these rules.

## Mission

Build the smallest technically sound product that proves Skawld can turn normal industrial work and experienced-technician judgment into reusable company knowledge.

Optimize for a solo developer now and credible enterprise/on-premise evolution later.

## Required Reading

Before planning or changing implementation:

1. Read [Spec.md](./Spec.md).
2. Read the relevant phase/task in [Plan.md](./Plan.md).
3. Read [ChangeLogs.md](./ChangeLogs.md) and applicable ADRs.
4. Inspect the pinned `skawld-sdk-go` public API. Do not infer SDK contracts from this documentation alone.
5. Inspect working-tree changes and preserve unrelated user work.

## Git Delivery Workflow

The repository uses three protected environment branches and short-lived
feature branches:

```text
feature/<scope>-<description>
        ↓ pull request
developer
        ↓ promotion pull request
staging
        ↓ approved release pull request + tag
production
```

- Branch a new feature from `developer`; never commit a feature directly on
  `developer`, `staging`, or `production`.
- `developer` is the integration branch. `staging` is the release-candidate
  branch. `production` contains only approved, deployable releases.
- Protect all three environment branches in GitHub: require pull requests,
  required CI checks, at least one review, resolved conversations, and no
  force push or branch deletion.
- Keep each commit atomic and reviewable. A commit must describe one coherent
  change and include its tests or documentation when they are part of that
  change. Do not use a catch-all “implement everything” commit.
- Use Conventional Commits: `feat:`, `fix:`, `docs:`, `test:`, `refactor:`,
  `build:`, `ci:`, `perf:`, or `chore:`. Optional scopes are encouraged, for
  example `feat(copilot): add evidence-backed recommendations`.
- Rebase or merge the latest `developer` into a feature branch before opening
  or updating its pull request, according to the team’s chosen merge policy.
- A feature branch is deleted only after its pull request is merged. Releases
  are tagged from the exact commit promoted to `production`.

Architecture is accepted with the 2026-07-26 review amendments. The owner
explicitly authorized implementation through the Phase 5 engineering baseline
on 2026-07-26. Customer qualification, production signing, OT integration, and
any post-Phase-5 product expansion still require an explicit request.

## Non-Negotiable Product Boundaries

- Only production code in `internal/skawld` may import `skawld-sdk-go`; the
  isolated `test/contract/sdk` suite is the sole test exception. The SDK must
  never import this product.
- Maintenance concepts never enter the generic SDK.
- Maintenance domain/application structs and public ports never expose SDK types.
- RBAC permissions and `ApprovalAuthority` are separate checks.
- Every external projection declares `OWNED_BY_SKAWLD` or `EXTERNAL_REFERENCE`.
- Skawld is an intelligence layer, not a replacement CMMS/EAM, historian, reliability suite, PTW/LOTO/MOC authority, or industrial controller.
- Never implement an executable tool for PLC/SCADA/DCS control, shutdown, motor/valve operation, interlock/override/bypass, setpoint change, barrier authorization, energy isolation approval, or safety-system action.
- Raw high-frequency telemetry does not go directly to an LLM or general application table.
- AI output is always untrusted.
- Vision creates `ObservationCandidate`, never a verified observation.
- Workflow learning creates a candidate or improvement candidate, never a self-modifying production workflow.
- Published workflow versions are immutable and applicability expansion requires authorized human approval.
- `INSUFFICIENT_EVIDENCE` is a successful safe outcome.

If a request conflicts with these boundaries, stop and explain the conflict before changing code.

## Architecture Rules

### Domain first

- Domain packages contain business language, invariants, state transitions, and value objects.
- They do not import HTTP routers, SQL drivers, S3/AI/OIDC SDKs, job frameworks, or concrete clocks/ID generators.
- Do not mirror every table as a business object.

### Application layer

- Application services implement explicit commands and queries, authorization, orchestration, and transaction boundaries.
- Important processes are inspectable state machines/workflows.
- Prefer deterministic code over agentic behavior.
- Define an interface when it protects a real boundary or enables a required fake. Do not create interfaces for every struct.
- Treat API and worker as composition roles over these same services; never connect them with an internal HTTP API.

### Adapters

- PostgreSQL, S3, OIDC, AI, speech, vision, embeddings, EAM/CMMS, and edge integrations are adapters.
- HTTP handlers parse/format and call application services; they do not contain business rules.
- `internal/skawld` contains tools, observers, policies, stores, extractors, schemas/prompts, and model routing. Its tools call application services, never repositories directly.
- No module reads another module’s tables directly except an explicitly approved reporting read model.

### Processes and infrastructure

- The architecture is one modular monolith with independently scalable API and worker process roles over shared application code.
- PostgreSQL is the only V1 server database.
- Use authorization-first eligible-set PostgreSQL FTS + exact pgvector cosine + explicit RRF. Do not add a reranker/search/vector service without measured evidence and an ADR.
- Approximate vector indexes require filtered-recall evaluation against exact search; security filtering is never post-processing.
- Use PostgreSQL-backed River jobs with a dedicated worker pool, connection budget, per-type concurrency, deadlines, idempotency, and retry classification. Do not add Redis, RabbitMQ, NATS, Kafka, or Kubernetes without an approved trigger.
- Web is a Go REST client. Do not add a Next.js/backend business layer.
- Web auth uses a Go server session with secure HttpOnly cookie; never store long-lived OIDC tokens in browser `localStorage`. Flutter uses Authorization Code + PKCE and platform secure storage.
- Object storage uses the narrow product capability contract, not the complete S3 API and not one vendor.
- Production distribution remains Linux OCI by default. Native Windows/macOS
  backend archives are unsigned engineering artifacts until signing,
  notarization, service management, upgrade, and support requirements are
  explicitly approved.

## Safety Pipeline

Any AI-proposed state change follows:

```text
AI proposal
    ↓
closed schema validation
    ↓
domain validation
    ↓
workflow state and prerequisite validation
    ↓
risk policy
    ↓
ordinary permission
    ↓
matching current ApprovalAuthority when approval is required
    ↓
human approval when required
    ↓
idempotent application command/tool
    ↓
domain state + audit
```

Never skip a layer because a prompt says an action is approved. Actor, tenant, site, permission, approval, and evidence IDs come from trusted server state, not model output.

## Security Rules

- Derive organization/site scope from the authenticated principal and authorized resource.
- Parameterize every SQL query. Never execute model-generated SQL.
- Never expose shell, filesystem write, arbitrary HTTP fetch, or generic coding tools to a maintenance agent.
- Generate object keys; never use an uploaded filename as a filesystem/object path.
- Validate file size, magic bytes, MIME, checksum, and ingestion limits; quarantine before availability.
- Treat PDFs, OCR, transcripts, prompts embedded in files, EXIF, filenames, external APIs, and all model output as untrusted.
- Do not log secrets, raw tokens, sensitive full prompts, or unrestricted document content.
- Restrict egress and prevent SSRF; connectors use configured allowlisted destinations.
- Use idempotency for mobile commands, jobs, tool writes, and external integration writes.
- Audit business/security events separately from operational logs.
- Do not claim ISO/IEC/ISA compliance. Document mappings and request qualified review.

## Industrial Data Rules

- Asset hierarchy is flexible and typed; do not hard-code a fixed number of levels.
- Criticality is human-approved and records its impact dimensions and rationale.
- Measurements use a controlled type, decimal value, unit, observed time, source, quality, component, instrument reference, and verification.
- Retain original measurement value/unit when converting.
- Engineering conversion, thresholds, aggregation, and state validation are deterministic and tested.
- Store manual and low-frequency maintenance measurements only. Integrate with a historian for high-volume telemetry.
- Preserve provenance, revision, approval, applicability, and verification for every knowledge source.
- Declare and enforce source of truth. External assets/work orders/inventory/permits/telemetry are projections/references unless a configured lightweight mode explicitly owns them.
- Filter authorization, tenant, site, applicability, approval, validity, and authority in database eligibility queries before FTS/vector ranking or AI context construction.

## Offline Rules

- A field write and its outbox record commit in one local Drift transaction.
- Every mutation has a stable `client_event_id`, idempotency key, `device_id`, base server version, `created_at_device`, payload version, attempt count, and sync status; the server records `received_at_server` and the result version.
- Server processing is idempotent and returns the original result for a safe replay.
- Attachments synchronize separately from structured work.
- Never silently resolve a safety/workflow/document/approval conflict in favor of the device.
- Do not add CRDTs unless a real concurrent-editing use case proves the need.
- Test process restart, duplicate, reordered, interrupted, and stale-version sync.
- Do not assume a normal phone is permitted in a hazardous area.

## SDK Integration Rules

- Pin an exact SDK tag for CI/release. Use workspace-level `go.work` only for sibling development.
- The current verified release pin is `skawld-sdk-go v0.2.0`; upgrades require
  the same contract suite against both the candidate workspace and exact tag.
- Use the actual SDK `core.Tool`, workflow executor, observation recorder, learning compiler, policy/approval, and audit contracts.
- Do not copy SDK types/runtimes into maintenance.
- Enforce by import test that only `internal/skawld` imports SDK packages.
- Do not expose SDK coding tools.
- Product tool names and closed JSON schemas are versioned contracts.
- Wrap SDK upgrades with contract tests for validation, approval pause/resume, idempotency, audit, demonstrations, compilation, and publication.
- If a missing capability is generic, propose it in `skawld-sdk-go`. If it contains maintenance language or applicability rules, keep it here.

## Coding and Data Conventions

- Prefer small cohesive packages named for domain capabilities.
- Avoid `common`, `utils`, `helpers`, and global mutable registries.
- Use explicit constructors that validate required dependencies.
- Pass `context.Context` to I/O boundaries and honor cancellation/timeouts.
- Use UTC instants in storage and retain site timezone for human interpretation.
- Use consistent IDs and separate human-readable numbers/tags.
- Use optimistic versions for mutable aggregates.
- Use append/supersession for evidence, measurements, observations, audit, workflow versions, and approved records where history matters.
- Treat committed `domain_events` as the durable semantic-capture source.
  SDK observation processing is asynchronous and retryable; it must never
  become part of the authoritative maintenance state transition.
- Never update or delete captured `demonstration_events` or review records.
  Apply deterministic ingress sanitization and append review redaction
  overlays. A failed capture delivery remains visible and blocks trace
  completion until recovered.
- Use JSONB for variable external/provider metadata, not as an escape from modeling stable invariants.
- Migrations are forward reviewed; never edit an applied migration.
- Do not add dependencies without a concrete need, license check, maintenance check, and test strategy.

## API Rules

- Version public routes under `/api/v1`.
- Maintain OpenAPI as the external contract and generate client types where useful.
- Use explicit lifecycle commands instead of arbitrary patches for meaningful transitions.
- Require idempotency keys on creates/commands/sync.
- Use cursor pagination and stable problem details.
- Do not create a generic `/chat`; model requests are typed application use cases.
- Authorization is server-side even when UI hides an action.
- Workflow/report/safety-relevant approval requires both the ordinary permission and a matching current `ApprovalAuthority`.

## Test Expectations

Every change includes the smallest sufficient combination of:

- domain unit tests for invariants and state transitions;
- repository/adapter tests against real PostgreSQL/S3-compatible test targets;
- HTTP contract and authorization tests;
- SDK integration contract tests;
- web unit/accessibility/end-to-end tests;
- Flutter unit/widget/offline-sync tests;
- security-negative tests for tenant leakage, malformed input, and untrusted AI/content;
- approval-authority tests independent from role tests;
- source-of-truth projection/write-back tests;
- River pool/concurrency/deadline/retry/idempotency tests;
- exact-versus-approximate filtered-recall tests before enabling HNSW;
- evaluation fixtures for retrieval/recommendation behavior.

Most tests must not call real AI services. Use:

```text
FakeLLM
FakeEmbeddingProvider
FakeVisionProvider
FakeTranscriptionProvider
FakeClock
FakeObjectStore
FakeIdentity
in-memory domain repositories where appropriate
```

Provider contract tests are opt-in and secret-gated. Do not assert exact prose from nondeterministic models; validate schemas, evidence IDs, risk, invariants, and reviewer-labelled outcomes.

Before handing off code, run formatting, vet/lint, unit tests, relevant integration/contract tests, generated-code consistency, and migration checks. State what was not run.

## Change Discipline

- Work on one approved Plan task or narrowly described change at a time.
- Preserve unrelated working-tree changes.
- Do not add microservices or optional infrastructure “for later.”
- Record an ADR for a durable architecture deviation.
- Update `Spec.md` when the product contract changes, `Plan.md` when sequencing/acceptance changes, and `ChangeLogs.md` for every accepted material change.
- Lead handoff with outcome, tests, risks, and remaining decisions.

## Definition of Done

A change is done only when:

1. its acceptance criteria pass;
2. authorization/tenant scope and audit implications are handled;
3. offline/idempotency behavior is handled when relevant;
4. AI/provenance/evidence and safety behavior are handled when relevant;
5. tests prove important failure paths;
6. documentation/API/migrations are synchronized;
7. no excluded CMMS or OT-control scope was introduced.
