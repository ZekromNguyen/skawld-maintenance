#!/usr/bin/env bash
set -euo pipefail

: "${BACKUP_DATABASE_URL:?BACKUP_DATABASE_URL is required}"
: "${BACKUP_DIR:?BACKUP_DIR is required}"

# Optional monitoring marker scope (empty = org-wide / first organization).
: "${SKAWLD_ORGANIZATION_ID:-}"
: "${SKAWLD_SITE_ID:-}"

# Record a monitoring_backup_runs marker row so the monitoring console can
# surface backup freshness (RPO) as a first-class monitor. Best-effort: a
# marker failure must never fail the backup itself.
record_backup_run() {
  local status="$1"
  local run_id
  run_id="$(uuidgen 2>/dev/null || cat /proc/sys/kernel/random/uuid 2>/dev/null || echo "skawld-$(date -u +%s)")"
  psql "${BACKUP_DATABASE_URL}" -v ON_ERROR_STOP=1 -q -c "
    INSERT INTO monitoring_backup_runs (id, organization_id, site_id, started_at, status)
    VALUES (
      '${run_id}'::uuid,
      COALESCE(NULLIF('${SKAWLD_ORGANIZATION_ID:-}', '')::uuid,
               (SELECT id FROM organizations ORDER BY created_at LIMIT 1)),
      NULLIF('${SKAWLD_SITE_ID:-}', '')::uuid,
      now(), '${status}'
    )
    ON CONFLICT (id) DO NOTHING;" >/dev/null 2>&1 || true
}

case "${BACKUP_DIR}" in
  /|/home|/home/*/..)
    echo "BACKUP_DIR is too broad" >&2
    exit 2
    ;;
esac

command -v pg_dump >/dev/null
command -v psql >/dev/null
command -v sha256sum >/dev/null

mkdir -p "${BACKUP_DIR}"
chmod 700 "${BACKUP_DIR}"

backup_timestamp="${BACKUP_TIMESTAMP:-$(date -u +%Y%m%dT%H%M%SZ)}"
backup_name="skawld-postgres-${backup_timestamp}.dump"
backup_path="${BACKUP_DIR}/${backup_name}"
temporary_path="$(mktemp "${BACKUP_DIR}/.skawld-backup.XXXXXX")"
trap 'rm -f "${temporary_path}"' EXIT

pg_dump \
  --dbname="${BACKUP_DATABASE_URL}" \
  --format=custom \
  --compress=9 \
  --no-owner \
  --no-acl \
  --file="${temporary_path}"

chmod 600 "${temporary_path}"
mv "${temporary_path}" "${backup_path}"
trap - EXIT

record_backup_run "SUCCESS"

schema_version="$(
  psql "${BACKUP_DATABASE_URL}" -AtX \
    -c "SELECT coalesce(max(version_id), 0) FROM goose_db_version WHERE is_applied"
)"
sha256sum "${backup_path}" > "${backup_path}.sha256"
chmod 600 "${backup_path}.sha256"

metadata_path="${backup_path}.metadata"
{
  echo "format=postgres-custom"
  echo "created_at=${backup_timestamp}"
  echo "schema_version=${schema_version}"
  echo "object_storage_included=false"
} > "${metadata_path}"
chmod 600 "${metadata_path}"

echo "${backup_path}"
