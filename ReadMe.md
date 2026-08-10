# Skawld Maintenance Copilot

Skawld Maintenance Copilot is an **industrial maintenance intelligence layer** built on [`skawld-sdk-go`](https://github.com/ZekromNguyen/skawld-sdk-go).

Its purpose is to capture how experienced technicians inspect, decide, act, and verify; turn that work into reviewed reusable workflows; and provide evidence-backed assistance to other technicians while preserving human authority and industrial safety.

> Skawld should learn from normal industrial work while preserving safety, evidence, authority, and human control.

## Repository Status

**The Phase 5 pilot engineering baseline is implemented and the SDK `v0.2.0`
contract is pinned. A named customer deployment is not yet declared
pilot-ready: production IdP/S3/AI selection, agreed RPO/RTO, safety acceptance,
and signed Windows/macOS distribution remain open gates.**

The current files define the accepted product foundation:

- [Spec.md](./Spec.md) — product, architecture, stack, safety, domain/data/API, SDK integration, and MVP specification.
- [Plan.md](./Plan.md) — Phases 0–5 and the first 20 engineering tasks in dependency order.
- [ChangeLogs.md](./ChangeLogs.md) — decision/document change history.
- [Agent.md](./Agent.md) — contribution rules for human and AI engineering agents.

The implementation includes the deterministic Phase 1 maintenance slice:
asset hierarchy and approved criticality, incidents, pump inspection
executions, prerequisites, exact measurements, observations/actions/decisions,
secure attachment manifests, a React supervisor workbench, and an offline-first
Flutter/Drift technician client.

Phase 2 adds controlled document revisions and applicability, bounded PDF/text
ingestion, PostgreSQL FTS + exact pgvector cosine + explicit RRF, similar
resolved-incident evidence, typed advisory recommendations,
`INSUFFICIENT_EVIDENCE`, report draft/edit/submit/approve, shift handover
draft/edit/submit/accept/acknowledge, AI/retrieval provenance, and human
correction records. Speech is an optional configured HTTP capability; its
output remains an unverified candidate until a technician verifies it.

Phase 3 adds an SDK Observation v1 PostgreSQL store behind `internal/skawld`,
transactional domain-event capture deliveries, a single-concurrency River
capture worker with visible retry health, execution and shift-handover
demonstrations, exact recommendation-correction links, evidence-view events,
append-oriented human reviews, deterministic ingress sanitization, and
redaction-aware supervisor timelines. Demonstration review requires
`ApprovalAuthority` separately from RBAC.

Phase 4 compiles at least two compatible, approved, gap-free demonstrations
through a maintenance-specific SDK adapter. It preserves exact event evidence,
surfaces ambiguity and behavioral differences, records corrections as
improvement candidates, and requires separate scoped authority for review,
publication, applicability expansion, and retirement. Published workflow
versions are immutable and only effective, current, applicable versions can
guide work. Compilation is synchronous and bounded in the MVP; the existing
River worker remains the upgrade path when measured workload requires it.

Phase 5 adds live and frozen AI quality gates, tenant/site-scoped reviewer
metrics, backup/restore and tenant audit-export tooling, threat/incident/
retention runbooks, bounded job retry and database connection budgets, an
initial read-only EAM/CMMS snapshot adapter, an S3/OIDC qualification matrix,
a hardened single-server Compose profile, dependency SBOM, pinned CI actions,
image/dependency scan gates, tagged provenance, and Windows/macOS backend
engineering archives. These are engineering controls, not a compliance,
industrial-safety, RPO/RTO, or customer acceptance claim.

Quick start:

```bash
cp .env.example .env
set -a && source .env && set +a
make compose-up
make migrate-up
go run ./cmd/api
```

Run `go run ./cmd/worker` in another shell. The worker image includes
`pdftotext`; native worker development requires Poppler (`poppler-utils` on
Fedora/Debian-family systems). See
[the local development runbook](./docs/runbooks/local-development.md) for
Docker equivalents, the development identity, verification, and cleanup.
Run `make seed` (see
[the demo data runbook](./docs/runbooks/demo-data.md)) to load a complete
P-302 pump demo: asset, incident, execution with LOTO-gated intrusive step,
ingested SOP, recommendation + correction, approved report and handover, two
reviewed demonstrations, and a published workflow.

Run the supervisor web:

```bash
cd web
npm ci
npm run dev
```

The public marketing page lives at `/landing` (the SPA root stays the operator
console). Before building or changing any web UI, read the
[web UI design system](./docs/contributing/web-ui-design-system.md): it encodes
Skawld's tokens, dials, accessibility, and pre-flight rules so new interfaces
stay consistent with the brand.

Run the field client after installing Flutter:

```bash
cd mobile
flutter create --platforms=android --project-name skawld_maintenance_mobile .
flutter pub get
dart run build_runner build
flutter run \
  --dart-define=SKAWLD_API_URL=http://10.0.2.2:8080 \
  --dart-define=SKAWLD_OIDC_ISSUER=http://10.0.2.2:8081/realms/skawld
```

Platform-specific setup:

- [macOS development](./docs/runbooks/macos-development.md)
- [Windows development](./docs/runbooks/windows-development.md)
- [Windows/macOS desktop application packages](./docs/runbooks/desktop-app-packages.md)
- [native package contract](./docs/runbooks/native-packages.md)
- [single-server pilot installation](./docs/runbooks/pilot-installation.md)
- [backup and restore](./docs/runbooks/backup-restore.md)
- [OIDC and S3 support matrix](./docs/runbooks/identity-storage-support.md)
- [EAM/CMMS snapshot import boundary](./docs/runbooks/eam-cmms-import.md)
- [Phase 5 threat model](./docs/threat-model/pilot-v1.md)

Build Windows, macOS, and Linux archives locally:

```bash
VERSION=v0.1.0 make package
```

PowerShell:

```powershell
.\scripts\skawld.ps1 package v0.1.0
```

Those commands package the Go backend. Build the Flutter technician/workstation
desktop application on its native operating system:

```powershell
# Windows: portable ZIP + SHA-256
$env:VERSION = "v0.1.0"
make desktop-windows
```

```bash
# macOS: DMG, ZIP, and SHA-256 files
VERSION=v0.1.0 make desktop-macos
```

Artifacts are written to `mobile/dist/desktop/`. CI performs these builds on
native Windows and macOS runners rather than cross-compiling Flutter from
Linux. Each package carries a non-secret runtime configuration template, so
on-premise API/OIDC endpoints can be selected without rebuilding the app.

## What This Product Is

```text
maintenance work
      ↓
semantic observations + measurements + decisions + evidence + outcome
      ↓
reviewed demonstrations and human corrections
      ↓
versioned workflow candidates
      ↓
expert review and publication
      ↓
evidence-backed technician guidance
      ↓
new organizational knowledge
```

Skawld can provide lightweight incident and maintenance-execution records for smaller customers. In enterprise environments it should normally integrate with SAP PM, IBM Maximo, Infor EAM, Oracle, or another CMMS/EAM rather than replace them.

## What This Product Is Not

- Not a generic chatbot over manuals.
- Not a CMMS/EAM replacement.
- Not a historian or high-frequency telemetry store.
- Not an autonomous diagnostic authority.
- Not a permit-to-work, LOTO, MOC, inventory, or contractor-management platform.
- Not an OT control path.
- Not allowed to control a PLC, SCADA/DCS, motor, valve, interlock, setpoint, shutdown, bypass, or safety system.

## Accepted Stack

| Concern | Decision |
|---|---|
| Backend | Go |
| Architecture | One modular monolith; independently scalable API/worker roles over shared application code |
| Database | PostgreSQL |
| Text/semantic search | Eligible-set PostgreSQL FTS + exact pgvector cosine + explicit RRF |
| Jobs | River/PostgreSQL-backed worker with dedicated pool and per-type limits |
| External API | Versioned REST + OpenAPI |
| Supervisor web | React + TypeScript + Vite |
| UI | Tailwind + Radix + curated shadcn/ui + TanStack |
| Technician mobile | Flutter |
| Mobile persistence | SQLite through Drift |
| Binary storage | Narrow capability-level `ObjectStore`; S3 adapter; no default server product |
| Identity | External OIDC; web server session; mobile PKCE; RBAC separate from `ApprovalAuthority` |
| AI | Capability router and provider ports behind `internal/skawld`; SDK imports remain isolated there |
| Local deployment | Compose-compatible dependencies; native app processes |
| Production shape | OCI images; cloud, VPS, or on-prem capable; no Kubernetes requirement |

Important refinements:

1. React/Vite is preferred over Next.js because the dashboard has no current SSR/SEO need and Go remains the only backend.
2. The S3 API is the stable storage decision, but MinIO Community is not the default after its upstream repository was archived in April 2026. A deployment-specific supported implementation must be selected after license, security, backup, and support review.
3. Role/permission answers “may this person perform the action?”; `ApprovalAuthority` separately answers “may this person approve this subject and risk in this scope and time window?”
4. Only production code in `internal/skawld` imports `skawld-sdk-go`;
   `test/contract/sdk` is the isolated verification exception. Maintenance
   domain/application types do not expose SDK types.
5. Asset, incident, work-order, permit, inventory, and telemetry projections explicitly declare `OWNED_BY_SKAWLD` or `EXTERNAL_REFERENCE`.
6. Offline mutations carry client event/idempotency/device/device-time/server-receive metadata so retries cannot duplicate maintenance evidence.

## Repository Relationship

```text
developer workspace/
├── skawld-sdk-go/        # generic agent/workflow-learning SDK
└── skawld-maintenance/   # this industrial maintenance product
```

Dependency direction inside the product:

```text
maintenance domain/application
            ↓
internal/skawld
            ↓
skawld-sdk-go
```

Only `internal/skawld` imports the SDK. Never the reverse.

During development, a workspace-level `go.work` can point to both repositories. Release and CI builds pin an exact SDK `v0.x` tag in `go.mod` and do not depend on local replacements.

The current release dependency is
`github.com/ZekromNguyen/skawld-sdk-go v0.2.0`. Contract tests exercise its
provider, tool, workflow, policy/approval, audit, observation, learning,
evaluation, review, and publication boundaries.

## Intended Product Monorepo

This repository contains the Go product foundation. React web and Flutter mobile
directories will be added only when their first approved vertical behavior
begins. Keeping the product surfaces together allows one API contract, fixture
set, release plan, and pilot backlog. The generic SDK remains separate.

Phase 1 uses Tailwind and semantic React primitives without adding unused
Radix/shadcn/TanStack dependencies. Those libraries remain selected extension
points when accessible dialogs, complex tables, or server-state behavior
actually require them.

The exact accepted tree is in [Spec.md §12](./Spec.md#12-repository-structure).

## First Demos

### Enterprise-first: Shift Handover

Skawld gathers open work, abnormal conditions, equipment state, external permit/isolation references, temporary changes, safety concerns, and follow-up into an evidence-linked draft. Outgoing and incoming supervisors review, accept, and acknowledge it.

This demonstrates learning from normal work and preserving organizational memory with lower diagnostic risk.

### Maintenance intelligence: High Vibration Pump

For pump P-302 with vibration `8.1 mm/s` and bearing temperature `94 °C`,
Skawld retrieves asset history, approved/current documents, and similar
incidents. It guides a technician through a deterministic non-intrusive
inspection, blocks steps with missing prerequisites, records offline field
evidence, drafts a report, and captures the senior technician's work and exact
AI correction as a reviewed demonstration. A supervisor can compile multiple
reviewed executions into an evidence-linked candidate, inspect its differences,
approve and publish it, then expand applicability only through a distinct
authorized action.

No AI output controls equipment or silently changes a published workflow.

## Current Gates

- Repository license still requires the owner's explicit choice.
- The Phase 1 Flutter/Drift baseline was resolved, generated, analyzed, and
  tested with Flutter `3.44.8`. The Phase 2 copilot panel still requires the
  native/mobile CI analysis and test run because Flutter is not installed in
  the current verification shell.
- Native CI definitions produce a portable Windows ZIP and macOS DMG/ZIP with
  SHA-256 files. Those OS builds remain unsigned engineering artifacts pending
  a successful native run, Authenticode, and Apple Developer ID
  signing/notarization.
- Production object storage and deployment topology remain customer/deployment
  decisions; optional S3Mock is only a development contract target.
- The development structured/embedding providers are deterministic fakes for
  repeatable demos and tests, not production reasoning models.
- A deployment speech endpoint is optional. Without it the transcription API
  fails explicitly with `503`; no public AI service is assumed.
- There is no generic chat, autonomous diagnosis, CMMS scheduling/inventory,
  permit authority, workflow self-modification, or industrial control path.

See [Plan.md](./Plan.md) for current progress and the dependency-ordered tasks.

## Importing external EAM/CMMS data

A pilot operator can project external asset records from an EAM/CMMS export
into the database as `EXTERNAL_REFERENCE` assets (never overwriting
Skawld-owned records) with the import CLI:

```bash
go run ./cmd/import \
  -snapshot test/fixtures/import-p302.ndjson \
  -site-id <site-uuid> \
  -database-url "$DATABASE_URL" \
  -external-subject seed-admin
```

The snapshot is a newline-delimited JSON export (see
`test/fixtures/import-p302.ndjson` for the record shape). The CLI runs under
the tenant/site scope of the named administrator, rewrites the tenant-neutral
`REPLACED_BY_PRINCIPAL` / `REPLACED_BY_SITE` placeholders, and imports in
bounded pages (`-limit`, default 100, max 500). It is idempotent: records
whose external version already exists are skipped, and a later
`-cursor` resumes where a previous run stopped. The same projection is also
available to administrators through `POST /api/v1/integrations/imports`.
