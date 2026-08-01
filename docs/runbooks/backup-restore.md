# Backup, Restore, and Disaster-Recovery Rehearsal

RPO/RTO are customer decisions. The engineering pilot target used for
rehearsal is daily database/object backup (`RPO <= 24h`) and verified service
restore within four hours (`RTO <= 4h`). This is not a contractual guarantee.

## Database backup

Run with a read-capable backup identity and an encrypted destination:

```text
BACKUP_DATABASE_URL=postgres://... \
BACKUP_DIR=/srv/skawld-backups/postgres \
scripts/backup-postgres.sh
```

The script produces a PostgreSQL custom-format dump, SHA-256 file, and metadata
containing schema version. Files are mode `0600`.

## Object storage

The S3 API does not define a portable backup mechanism. Use the selected
provider's supported versioning/replication/export tool, preserve object keys
and checksums, and record its separate recovery point. A database dump without
matching objects is not a complete Skawld recovery.

## Empty-target restore

Never rehearse into the source or production database. Create a new isolated
database, then:

```text
RESTORE_DATABASE_URL=postgres://.../skawld_restore \
BACKUP_FILE=/srv/skawld-backups/postgres/skawld-postgres-....dump \
scripts/restore-postgres.sh

SOURCE_DATABASE_URL=postgres://.../skawld \
RESTORED_DATABASE_URL=postgres://.../skawld_restore \
scripts/verify-restore.sh
```

The restore script refuses a non-empty target and verifies the adjacent
checksum. The verifier compares schema version and safety/knowledge/audit table
counts. Then perform application smoke tests against the restored database and
sample-download restored objects.

## Rehearsal record

Record start/end, backup/object versions, source and target topology, schema
version, checksum result, table/object samples, achieved RPO/RTO, operator,
reviewer, deviations, and corrective tasks. Failed verification blocks pilot
release.

## Migration and rollback

Take verified database/object backups before migration. Prefer roll-forward.
Never run a down migration merely to make old binaries start. Phase 3–5
append-only/immutable migrations intentionally refuse or avoid unsafe rollback.
If application rollback is required, verify the older binary tolerates the
newer schema in staging; otherwise restore the pre-migration backup into a new
database and switch only after verification.
