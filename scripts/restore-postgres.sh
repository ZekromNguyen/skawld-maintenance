#!/usr/bin/env bash
set -euo pipefail

: "${RESTORE_DATABASE_URL:?RESTORE_DATABASE_URL is required}"
: "${BACKUP_FILE:?BACKUP_FILE is required}"

if [[ ! -f "${BACKUP_FILE}" ]]; then
  echo "backup file does not exist: ${BACKUP_FILE}" >&2
  exit 2
fi

command -v pg_restore >/dev/null
command -v psql >/dev/null

checksum_path="${BACKUP_FILE}.sha256"
if [[ -f "${checksum_path}" ]]; then
  (
    cd "$(dirname "${BACKUP_FILE}")"
    sha256sum -c "$(basename "${checksum_path}")"
  )
else
  echo "checksum file is required beside the backup" >&2
  exit 2
fi

user_table_count="$(
  psql "${RESTORE_DATABASE_URL}" -AtX -c "
    SELECT count(*)
    FROM pg_catalog.pg_tables
    WHERE schemaname NOT IN ('pg_catalog', 'information_schema');
  "
)"
if [[ "${user_table_count}" != "0" ]]; then
  echo "restore target must be an empty database" >&2
  exit 2
fi

pg_restore \
  --dbname="${RESTORE_DATABASE_URL}" \
  --exit-on-error \
  --single-transaction \
  --no-owner \
  --no-privileges \
  "${BACKUP_FILE}"

psql "${RESTORE_DATABASE_URL}" -v ON_ERROR_STOP=1 -AtX -c "
  SELECT max(version_id)
  FROM goose_db_version
  WHERE is_applied;
"
