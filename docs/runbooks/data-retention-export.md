# Data Retention and Export

No default deletion job is enabled because industrial retention and legal-hold
requirements are deployment-specific. The pilot owner must approve durations
for application logs, sessions/idempotency records, attachments, voice,
documents/revisions, maintenance records, AI calls, demonstrations,
corrections, workflows, and audit.

Rules:

- official/superseded documents and published workflows remain auditable;
- audit, human review, workflow review, and correction rows are append-oriented;
- legal hold overrides normal expiry;
- deletion must remove database metadata and object content consistently;
- external projections follow the authoritative-system agreement;
- model/provider credentials never enter retained product data;
- export and deletion operations are themselves audited or change-controlled.

## Audit export

Use an authorized read-only database identity:

```text
EXPORT_DATABASE_URL=postgres://... \
EXPORT_ORGANIZATION_ID=00000000-0000-0000-0000-000000000000 \
EXPORT_FILE=/secure/export/audit.jsonl.gz \
scripts/export-audit.sh
```

Output is tenant-scoped, chronological JSON Lines compressed with gzip and
accompanied by SHA-256. Store it in the customer-approved immutable archive if
required.

## Customer data export

For the pilot, use a verified PostgreSQL custom dump plus provider-native S3
object export. A future portable product export must preserve IDs, provenance,
revisions, applicability, evidence references, checksums, approvals, and audit;
it must not be improvised as a vector-database dump.
