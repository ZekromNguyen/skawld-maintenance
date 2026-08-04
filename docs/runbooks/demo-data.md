# Demo Data Seed and Data Workflow

`cmd/seed` loads the repository with a complete, coherent maintenance demo for
the P-302 pump so the supervisor web, the mobile client, and the CI workflows
all have data to operate on. It drives the same application services the API
uses, so every record carries audit, idempotency, domain-event, and approval
guarantees.

## Prerequisites

Same stack as the local runbook:

```bash
cp .env.example .env
set -a && source .env && set +a
make compose-up                  # PostgreSQL + Keycloak
make compose-up-objectstore      # S3Mock (attachments)
make migrate-up
```

Start API and worker in separate shells (the worker is needed for River jobs;
the seed also processes documents synchronously, so the worker is optional for
seeding alone):

```bash
go run ./cmd/api
go run ./cmd/worker
```

## Run

```bash
make seed
# or
go run ./cmd/seed
```

The command is idempotent: it reuses the existing "Demo Maintenance Co."
organization, site, admin principal, and approval authority when present, and
creates them otherwise.

It prints the IDs of everything it created:

```text
"msg":"demo data ready","organization_id":"...","site_id":"...","principal_id":"...",
"asset_id":"...","incident_id":"...","execution_id":"...",
"recommendation_id":"...","report_id":"...","handover_id":"...",
"workflow_id":"...","workflow_version":1
```

To start from a clean demo state first (deletes only the "Demo Maintenance
Co." organization and its rows), see the SQL in the seed section below.

## What the seed creates

| Area | Record | Details |
|---|---|---|
| Identity | Organization "Demo Maintenance Co." | `OWNED_BY_SKAWLD` |
| Identity | Site "Plant A" | code `PLANT-A`, UTC |
| Identity | Principal `seed-admin` | `Administrator` membership + broad `approval_authorities` grant (risk ceiling 3) |
| Asset | `P-302` Process Pump | class `CENTRIFUGAL_PUMP`, manufacturer "Fictional Pump Co.", components `MTR-BRG` / `PUMP-BRG`, approved criticality A |
| Incident | "High vibration on P-302 motor bearing" | severity HIGH |
| Execution | "Diagnose high vibration on P-302" | started, 3 informational/advisory steps completed, `8.1 mm/s` vibration + `94 °C` temperature recorded, observation, intrusive step blocked without LOTO then unblocked after `ENERGY_ISOLATION` verification, action + decision, completed |
| Knowledge | "P-302 Bearing Lubrication SOP" (SOP, R1) | text/plain attachment uploaded + verified, ingested to READY (chunks + embeddings), revision approved |
| Copilot | Recommendation | correlated with the SOP; completes with `INSUFFICIENT_EVIDENCE` when no evidence exists |
| Copilot | Correction feedback | `CORRECTED` outcome linked to the recommendation |
| Report | Maintenance report | drafted (deterministic provider), submitted, approved |
| Handover | Shift handover | prepared, submitted, accepted |
| Demonstrations | 2 completed + APPROVED pump executions | chart the full trace: measurement, recommendation, correction, outcome |
| Workflow | "High vibration centrifugal pump inspection" | compiled from the two demonstrations, reviewed APPROVED, published, applicable to P-302 |

## Data workflow (Phase 0–5 loop)

```text
phase0.admin  (browser, Keycloak)
      │  OIDC Authorization Code + PKCE → server-side session cookie
      ▼
/api/v1/*           (Go API, application services, RBAC + ApprovalAuthority)
      │
      ├─ Asset & criticality (Phase 1)   ──  audit + domain events
      ├─ Incident & execution (Phase 1)  ──  measurements, LOTO gating
      ├─ Attachment S3 (Phase 1)         ──  generated keys, checksum/MIME gate
      ├─ Document ingestion (Phase 2)    ──  River worker → chunks + embeddings
      ├─ Search (Phase 2)                ──  eligibility → lexical + cosine + RRF
      ├─ Recommendation (Phase 2)        ──  evidence packet → structured output
      ├─ Report / handover (Phase 2)     ──  draft → submit → approve/accept
      ├─ Demonstration capture (Phase 3) ──  domain events → SDK observations
      └─ Workflow learning (Phase 4)     ──  compile → review → publish
```

The end-to-end loop:

1. **Foundation** — organization, site, memberships, approval authority.
2. **Core maintenance** — asset + criticality, incident, execution with
   measurements (`8.1 mm/s`, `94 °C` stored exactly), LOTO-blocked intrusive
   step, action + decision, completion.
3. **Copilot** — upload + ingest the SOP, hybrid search, evidence-backed
   recommendation, human correction feedback.
4. **Governance** — report draft → submit → approve (separate authority),
   handover prepare → submit → accept.
5. **Demonstrations** — two reviewed pump demonstrations from the same work.
6. **Workflow learning** — compile the two demos into a candidate, approve,
   publish, and make it applicable to P-302. The published version is
   immutable and drives `workflows/applicable`.

Every write is idempotent (replay-safe), committed atomically with its audit
event, and (for executions/documents) appends a durable domain event so
demonstrations can be captured later without rolling back the work.

## Viewing in the UI

1. Open `http://localhost:5173` (Vite dev server).
2. Sign in with the Keycloak development user:

   ```text
   username: phase0.admin
   password: phase0-admin-dev
   ```

   `make seed` also provisions one account per product role
   (`dev.supervisor` / `dev-senior-pw`, `dev.senior` / `dev-senior-pw`,
   `dev.technician` / `dev-technician-pw`, `dev.manager` / `dev-manager-pw`) —
   see the [local development runbook](./local-development.md) credential
   table. Each is bound to the demo organization and site with the matching
   `memberships` role and an approval authority.

3. The supervisor workbench boots into the "Demo Maintenance Co." organization
   and shows the P-302 asset, incident, execution timeline, SOP, search,
   recommendations, report, handover, demonstrations, and the published
   workflow. Sign out and in again if a previous session cached an older
   organization.

## Verification commands

```bash
go build ./... && go vet ./...
go test ./internal/attachment/...   # includes MIME normalization regression
cd web && npm ci && npx tsc --noEmit && npm test && npm run build
```

## Cleanup

To remove only the demo organization (safe: leave other data intact):

```sql
-- run as skawld_owner
BEGIN;
DELETE FROM demonstration_capture_deliveries WHERE demonstration_id IN (
  SELECT d.id FROM demonstrations d JOIN sites s ON s.id=d.site_id
  WHERE s.organization_id IN (SELECT id FROM organizations WHERE name='Demo Maintenance Co.'));
DELETE FROM demonstration_event_redactions WHERE demonstration_id IN (
  SELECT d.id FROM demonstrations d JOIN sites s ON s.id=d.site_id
  WHERE s.organization_id IN (SELECT id FROM organizations WHERE name='Demo Maintenance Co.'));
DELETE FROM demonstration_reviews WHERE demonstration_id IN (
  SELECT d.id FROM demonstrations d JOIN sites s ON s.id=d.site_id
  WHERE s.organization_id IN (SELECT id FROM organizations WHERE name='Demo Maintenance Co.'));
DELETE FROM demonstrations WHERE id IN (
  SELECT d.id FROM demonstrations d JOIN sites s ON s.id=d.site_id
  WHERE s.organization_id IN (SELECT id FROM organizations WHERE name='Demo Maintenance Co.'));
DELETE FROM maintenance_workflows WHERE organization_id IN (SELECT id FROM organizations WHERE name='Demo Maintenance Co.');
DELETE FROM workflow_versions WHERE organization_id IN (SELECT id FROM organizations WHERE name='Demo Maintenance Co.');
DELETE FROM workflow_applicability WHERE organization_id IN (SELECT id FROM organizations WHERE name='Demo Maintenance Co.');
DELETE FROM workflow_reviews WHERE organization_id IN (SELECT id FROM organizations WHERE name='Demo Maintenance Co.');
DELETE FROM workflow_evaluation_reports WHERE organization_id IN (SELECT id FROM organizations WHERE name='Demo Maintenance Co.');
DELETE FROM workflow_improvement_candidates WHERE organization_id IN (SELECT id FROM organizations WHERE name='Demo Maintenance Co.');
DELETE FROM retrieval_runs WHERE organization_id IN (SELECT id FROM organizations WHERE name='Demo Maintenance Co.');
DELETE FROM ai_call_records WHERE organization_id IN (SELECT id FROM organizations WHERE name='Demo Maintenance Co.');
DELETE FROM recommendation_feedback WHERE organization_id IN (SELECT id FROM organizations WHERE name='Demo Maintenance Co.');
DELETE FROM recommendations WHERE organization_id IN (SELECT id FROM organizations WHERE name='Demo Maintenance Co.');
DELETE FROM maintenance_report_edits WHERE organization_id IN (SELECT id FROM organizations WHERE name='Demo Maintenance Co.');
DELETE FROM maintenance_reports WHERE organization_id IN (SELECT id FROM organizations WHERE name='Demo Maintenance Co.');
DELETE FROM shift_handovers WHERE organization_id IN (SELECT id FROM organizations WHERE name='Demo Maintenance Co.');
DELETE FROM sync_inbox WHERE organization_id IN (SELECT id FROM organizations WHERE name='Demo Maintenance Co.');
DELETE FROM document_chunks WHERE organization_id IN (SELECT id FROM organizations WHERE name='Demo Maintenance Co.');
DELETE FROM embeddings WHERE organization_id IN (SELECT id FROM organizations WHERE name='Demo Maintenance Co.');
DELETE FROM document_applicability WHERE organization_id IN (SELECT id FROM organizations WHERE name='Demo Maintenance Co.');
DELETE FROM document_revisions WHERE organization_id IN (SELECT id FROM organizations WHERE name='Demo Maintenance Co.');
DELETE FROM attachments WHERE organization_id IN (SELECT id FROM organizations WHERE name='Demo Maintenance Co.');
DELETE FROM execution_decisions WHERE organization_id IN (SELECT id FROM organizations WHERE name='Demo Maintenance Co.');
DELETE FROM maintenance_actions WHERE organization_id IN (SELECT id FROM organizations WHERE name='Demo Maintenance Co.');
DELETE FROM observations WHERE organization_id IN (SELECT id FROM organizations WHERE name='Demo Maintenance Co.');
DELETE FROM measurements WHERE organization_id IN (SELECT id FROM organizations WHERE name='Demo Maintenance Co.');
DELETE FROM prerequisite_verifications WHERE organization_id IN (SELECT id FROM organizations WHERE name='Demo Maintenance Co.');
DELETE FROM execution_steps WHERE organization_id IN (SELECT id FROM organizations WHERE name='Demo Maintenance Co.');
DELETE FROM maintenance_executions WHERE organization_id IN (SELECT id FROM organizations WHERE name='Demo Maintenance Co.');
DELETE FROM incidents WHERE organization_id IN (SELECT id FROM organizations WHERE name='Demo Maintenance Co.');
DELETE FROM asset_components WHERE organization_id IN (SELECT id FROM organizations WHERE name='Demo Maintenance Co.');
DELETE FROM asset_criticalities WHERE organization_id IN (SELECT id FROM organizations WHERE name='Demo Maintenance Co.');
DELETE FROM asset_relationships WHERE organization_id IN (SELECT id FROM organizations WHERE name='Demo Maintenance Co.');
DELETE FROM assets WHERE organization_id IN (SELECT id FROM organizations WHERE name='Demo Maintenance Co.');
DELETE FROM domain_events WHERE organization_id IN (SELECT id FROM organizations WHERE name='Demo Maintenance Co.');
DELETE FROM audit_events WHERE organization_id IN (SELECT id FROM organizations WHERE name='Demo Maintenance Co.');
DELETE FROM documents WHERE organization_id IN (SELECT id FROM organizations WHERE name='Demo Maintenance Co.');
DELETE FROM approval_authorities WHERE organization_id IN (SELECT id FROM organizations WHERE name='Demo Maintenance Co.');
DELETE FROM memberships WHERE organization_id IN (SELECT id FROM organizations WHERE name='Demo Maintenance Co.');
DELETE FROM sites WHERE organization_id IN (SELECT id FROM organizations WHERE name='Demo Maintenance Co.');
DELETE FROM organizations WHERE name='Demo Maintenance Co.';
COMMIT;
```

## Note on attachments

`cmd/seed` demonstrates the complete attachment lifecycle: it creates the
manifest, uploads the object at the generated key, and completes the upload so
size/checksum/MIME verification runs. `verified_mime` is persisted normalized
(`text/plain`), which document ingestion requires. See the MIME regression
test in `internal/attachment/adapter/postgres/mime_test.go`.
