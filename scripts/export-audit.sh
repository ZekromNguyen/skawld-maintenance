#!/usr/bin/env bash
set -euo pipefail

: "${EXPORT_DATABASE_URL:?EXPORT_DATABASE_URL is required}"
: "${EXPORT_ORGANIZATION_ID:?EXPORT_ORGANIZATION_ID is required}"
: "${EXPORT_FILE:?EXPORT_FILE is required}"

if [[ ! "${EXPORT_ORGANIZATION_ID}" =~ ^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$ ]]; then
  echo "EXPORT_ORGANIZATION_ID must be a UUID" >&2
  exit 2
fi

command -v psql >/dev/null
command -v gzip >/dev/null
command -v sha256sum >/dev/null

mkdir -p "$(dirname "${EXPORT_FILE}")"
temporary_path="$(mktemp "$(dirname "${EXPORT_FILE}")/.audit-export.XXXXXX")"
trap 'rm -f "${temporary_path}"' EXIT

psql "${EXPORT_DATABASE_URL}" -v ON_ERROR_STOP=1 -X -At \
  -c "COPY (
    SELECT row_to_json(export_row)
    FROM (
      SELECT id, organization_id, site_id, actor_id, action, entity_kind,
             entity_id, reason, request_id, execution_id, workflow_id,
             approval_id, ai_involvement, before_value, after_value,
             attributes, occurred_at
      FROM audit_events
      WHERE organization_id = '${EXPORT_ORGANIZATION_ID}'::uuid
      ORDER BY occurred_at, id
    ) AS export_row
  ) TO STDOUT" |
  gzip -9 > "${temporary_path}"

chmod 600 "${temporary_path}"
mv "${temporary_path}" "${EXPORT_FILE}"
trap - EXIT
sha256sum "${EXPORT_FILE}" > "${EXPORT_FILE}.sha256"
chmod 600 "${EXPORT_FILE}.sha256"
echo "${EXPORT_FILE}"
