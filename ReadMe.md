# Skawld Maintenance Copilot

Skawld Maintenance Copilot is an **industrial maintenance intelligence layer** built on [`skawld-sdk-go`](https://github.com/ZekromNguyen/skawld-sdk-go).

Its purpose is to capture how experienced technicians inspect, decide, act, and verify; turn that work into reviewed reusable workflows; and provide evidence-backed assistance to other technicians while preserving human authority and industrial safety.

> Skawld should learn from normal industrial work while preserving safety, evidence, authority, and human control.

## Repository Status

**Phase 0 foundation is operational. The SDK release gate remains open.**

The current files define the accepted product foundation:

- [Spec.md](./Spec.md) — product, architecture, stack, safety, domain/data/API, SDK integration, and MVP specification.
- [Plan.md](./Plan.md) — Phases 0–5 and the first 20 engineering tasks in dependency order.
- [ChangeLogs.md](./ChangeLogs.md) — decision/document change history.
- [Agent.md](./Agent.md) — contribution rules for human and AI engineering agents.

The initial implementation includes runnable Go API/worker roles, migrations,
OIDC identity, scoped authorization foundations, idempotency, append-oriented
audit, Compose dependencies, OpenAPI, CI, and OCI images. Phase 1 domain work
has not started.

Quick start:

```bash
cp .env.example .env
set -a && source .env && set +a
make compose-up
make migrate-up
go run ./cmd/api
```

Run `go run ./cmd/worker` in another shell. See
[the local development runbook](./docs/runbooks/local-development.md) for
Docker equivalents, the development identity, verification, and cleanup.

Platform-specific setup:

- [macOS development](./docs/runbooks/macos-development.md)
- [Windows development](./docs/runbooks/windows-development.md)
- [native package contract](./docs/runbooks/native-packages.md)

Build Windows, macOS, and Linux archives locally:

```bash
VERSION=v0.1.0 make package
```

PowerShell:

```powershell
.\scripts\skawld.ps1 package v0.1.0
```

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
| AI | `skawld-sdk-go` behind the product’s `internal/skawld` integration boundary |
| Local deployment | Compose-compatible dependencies; native app processes |
| Production shape | OCI images; cloud, VPS, or on-prem capable; no Kubernetes requirement |

Important refinements:

1. React/Vite is preferred over Next.js because the dashboard has no current SSR/SEO need and Go remains the only backend.
2. The S3 API is the stable storage decision, but MinIO Community is not the default after its upstream repository was archived in April 2026. A deployment-specific supported implementation must be selected after license, security, backup, and support review.
3. Role/permission answers “may this person perform the action?”; `ApprovalAuthority` separately answers “may this person approve this subject and risk in this scope and time window?”
4. Only `internal/skawld` imports `skawld-sdk-go`; maintenance domain/application types do not expose SDK types.
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

## Intended Product Monorepo

This repository contains the Go product foundation. React web and Flutter mobile
directories will be added only when their first approved vertical behavior
begins. Keeping the product surfaces together allows one API contract, fixture
set, release plan, and pilot backlog. The generic SDK remains separate.

The exact accepted tree is in [Spec.md §12](./Spec.md#12-repository-structure).

## First Demos

### Enterprise-first: Shift Handover

Skawld gathers open work, abnormal conditions, equipment state, external permit/isolation references, temporary changes, safety concerns, and follow-up into an evidence-linked draft. Outgoing and incoming supervisors review, accept, and acknowledge it.

This demonstrates learning from normal work and preserving organizational memory with lower diagnostic risk.

### Maintenance intelligence: High Vibration Pump

For pump P-302 with vibration `8.1 mm/s` and bearing temperature `94 °C`, Skawld retrieves asset history, approved/current documents, similar incidents, and an applicable validated workflow. It guides a technician through non-intrusive inspection, blocks steps with missing prerequisites, records offline field evidence, drafts a report, and stores corrections as workflow-improvement evidence.

No AI output controls equipment or silently changes a published workflow.

## Remaining Phase 0 Gates

- Repository license still requires the owner's explicit choice.
- A clean compatible `skawld-sdk-go` `v0.x.y` tag and contract-test baseline.
- Production object storage and deployment topology remain customer/deployment
  decisions; optional S3Mock is only a development contract target.
- Fictional demo data/documents and first pilot customer assumptions belong to
  Phase 1 planning.

See [Plan.md](./Plan.md) for current progress and the dependency-ordered tasks.
