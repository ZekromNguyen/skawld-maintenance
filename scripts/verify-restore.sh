#!/usr/bin/env bash
set -euo pipefail

: "${SOURCE_DATABASE_URL:?SOURCE_DATABASE_URL is required}"
: "${RESTORED_DATABASE_URL:?RESTORED_DATABASE_URL is required}"

command -v psql >/dev/null

tables=(
  organizations
  sites
  assets
  incidents
  maintenance_executions
  documents
  recommendations
  demonstrations
  maintenance_workflows
  workflow_versions
  audit_events
)

source_version="$(
  psql "${SOURCE_DATABASE_URL}" -AtX \
    -c "SELECT max(version_id) FROM goose_db_version WHERE is_applied"
)"
restored_version="$(
  psql "${RESTORED_DATABASE_URL}" -AtX \
    -c "SELECT max(version_id) FROM goose_db_version WHERE is_applied"
)"
if [[ "${source_version}" != "${restored_version}" ]]; then
  echo "schema version mismatch: ${source_version} != ${restored_version}" >&2
  exit 3
fi

for table_name in "${tables[@]}"; do
  source_count="$(
    psql "${SOURCE_DATABASE_URL}" -AtX \
      -c "SELECT count(*) FROM ${table_name}"
  )"
  restored_count="$(
    psql "${RESTORED_DATABASE_URL}" -AtX \
      -c "SELECT count(*) FROM ${table_name}"
  )"
  if [[ "${source_count}" != "${restored_count}" ]]; then
    echo "${table_name}: ${source_count} != ${restored_count}" >&2
    exit 3
  fi
  echo "${table_name}: ${source_count}"
done

echo "restore verification passed at schema version ${source_version}"
