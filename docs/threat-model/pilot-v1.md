# Pilot Threat Model

Status: engineering baseline for pilot qualification
Owner: product/security owner
Review required: before every customer pilot and material architecture change

This is not an IEC 62443, ISO 27001, ISO 55001, or ISO 14224 compliance claim.
Customer OT/security, safety, legal, and data owners must review the actual
deployment.

## Scope and trust boundaries

```text
Technician device ── HTTPS/OIDC ── Reverse proxy ── Web/API
                                                │
                                  ┌─────────────┼─────────────┐
                                  │             │             │
                              PostgreSQL    S3 contract    AI endpoint
                                  │
                           worker/River jobs

EAM/CMMS ── allowlisted read-only connector ── projection boundary

OT/PLC/DCS/SCADA ── NOT CONNECTED in the pilot
```

Protected assets are tenant/site authorization, technician identity and
approval authority, official/current documents, measurements and work
evidence, demonstrations/corrections, published workflows, audit history,
object content, credentials, and AI/provider provenance.

## Threats, controls, and verification

| Threat or abuse case | Preventive/detective control | Verification |
|---|---|---|
| Cross-tenant/site read or mutation | Scope derives from authenticated principal; repository predicates; connector record validation; generated object keys | race-enabled repository/search/attachment/workflow/connector tests |
| RBAC used as approval authority | Application services require permission and a current scoped `ApprovalAuthority` | approval and workflow publication tests |
| Prompt/document injection reaches a tool | Retrieved material is evidence only; strict structured output; evidence-ID validation; read-only tool catalog | invented-evidence and unsafe-tool tests; frozen evaluation |
| Unsafe or critical recommendation | Advisory risk allowlist; critical controls have no tool; unsafe candidates must fail closed | `cmd/eval` gate requires zero escaped unsafe candidates |
| Workflow self-modification | Corrections create improvement candidates; review and publication are explicit; published payload DB trigger is immutable | Phase 4 integration and DB privilege tests |
| Malicious upload/path traversal | Server-generated keys, signed constrained URL, checksum/size/MIME checks, quarantine state, no user filename as path | object-store contract and attachment tests |
| SSRF through provider/connector | Configured fixed endpoints only; no arbitrary URL in user/model input; deployment egress allowlist | provider fixed-endpoint tests and deployment review |
| SQL/command injection | Parameterized SQL; no LLM-generated SQL/shell; fixed `pdftotext` binary and arguments | tests, `go vet`, code review |
| Job amplification/starvation | Dedicated worker pool, per-queue concurrency, timeout, max attempts, bounded retry, permanent/retry classification | job policy tests and OLTP load rehearsal |
| Offline duplicate/reorder/stale write | stable client event/idempotency IDs, unique outbox operation/event, deterministic ordering, bounded retry, visible 409/412 conflict | Flutter sync-policy tests |
| Audit erasure | separate append-oriented events and feedback; runtime role cannot update/delete | migration privilege tests and checksum export |
| Backup theft or unusable restore | restricted files, checksum, encrypted backup location, empty-target restore, count/schema verification | restore rehearsal record |
| Supply-chain compromise | pinned modules/images, `govulncheck`, SBOM, container scan, provenance attestation gate | GitHub Actions and release checklist |
| Secret leakage | environment/secret injection, no secrets in prompts/logs/artifacts, rotation runbook | configuration review and incident drill |

## Data classification

- Restricted: credentials, tokens, private model keys, identity-provider
  secrets. Never store in application tables, logs, prompts, or backups.
- Confidential industrial: asset history, procedures, measurements, images,
  voice, reports, demonstrations, workflow evidence, contractor identity.
- Controlled operational: audit/evaluation metadata and system configuration.
- Public: only explicitly approved product documentation and release metadata.

Encryption in transit and encrypted database/object/backup volumes are pilot
deployment requirements. The application does not make disk encryption true by
itself.

## Residual risks and explicit non-goals

- The deterministic development AI provider is not a production model
  qualification.
- Malware scanning and metadata stripping depend on the selected object-storage
  and upload-processing deployment; unscanned objects must remain quarantined.
- Mobile SQLite encryption and managed-device controls require customer device
  management qualification.
- Availability, RPO/RTO, support hours, retention, and geographic residency
  remain customer-specific.
- No PLC, SCADA, DCS, historian protocol, PTW/LOTO authority, barrier bypass,
  machine control, or direct OT connection exists in the pilot.

## Pilot sign-off

Record named owners for product, operations, customer IT, customer OT security,
maintenance authority, and process safety. Open high/critical findings block
deployment. A safety-success metric cannot compensate for any escaped critical
action.
