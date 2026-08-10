# EAM/CMMS Snapshot Import Boundary

The first pilot connector is deliberately pull-only. It consumes an immutable
newline-delimited JSON snapshot exported by SAP, Maximo, or another EAM/CMMS.
It cannot create or approve work orders, permits, isolations, inventory
movements, or operational actions in the external system.

## Contract

Each line is one `ExternalRecord`:

```json
{"kind":"ASSET","organization_id":"<uuid>","site_id":"<uuid>","external_system":"SAP","external_id":"P-302","external_version":"42","observed_at":"2026-07-26T00:00:00Z","attributes":{"tag":"P-302","name":"Process Pump","asset_class":"centrifugal_pump"}}
```

Allowed kinds are `ASSET`, `WORK_REFERENCE`, `MAINTENANCE_HISTORY`, and
`DOCUMENT_METADATA`. Every record retains external system, ID, version,
observation time, organization, and site provenance. Imported business context
is always `EXTERNAL_REFERENCE`; the adapter does not silently convert it to
`OWNED_BY_SKAWLD`.

The configured snapshot path must resolve inside an explicit allowed root.
Symlink/path escapes, non-regular files, unknown JSON fields, records larger
than 1 MiB, invalid cursors, cross-tenant records, and records outside the
principal's site scope fail closed.

## Pagination and change detection

The adapter cursor combines the SHA-256 digest of the complete snapshot and the
next line offset. If an operator replaces the export during a paged import, the
old cursor is rejected. Complete one snapshot before atomically replacing it
with the next export.

## Projection rule

`internal/integration/application.Importer` validates the complete returned
page before calling a product-specific `ProjectionSink`. The sink must apply
the page transactionally and idempotently, preserve source-of-truth metadata,
and audit the import. Do not wire a customer sink until the real EAM field
mapping and ownership rules are reviewed.

Write-back is a separate future adapter and requires its own risk, permission,
approval, idempotency, and customer-system contract. It is not part of Phase 5.
