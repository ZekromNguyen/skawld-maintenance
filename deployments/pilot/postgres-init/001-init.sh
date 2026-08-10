#!/usr/bin/env bash
set -euo pipefail

: "${SKAWLD_DB_OWNER_PASSWORD:?SKAWLD_DB_OWNER_PASSWORD is required}"
: "${SKAWLD_DB_APP_PASSWORD:?SKAWLD_DB_APP_PASSWORD is required}"

psql -v ON_ERROR_STOP=1 \
  --username "${POSTGRES_USER}" \
  --dbname "${POSTGRES_DB}" \
  --set=owner_password="${SKAWLD_DB_OWNER_PASSWORD}" \
  --set=app_password="${SKAWLD_DB_APP_PASSWORD}" <<'SQL'
SELECT format(
  'CREATE ROLE skawld_owner LOGIN PASSWORD %L',
  :'owner_password'
) \gexec
SELECT format(
  'CREATE ROLE skawld_app LOGIN PASSWORD %L NOSUPERUSER NOCREATEDB NOCREATEROLE',
  :'app_password'
) \gexec
CREATE DATABASE skawld OWNER skawld_owner;
SQL

psql -v ON_ERROR_STOP=1 \
  --username "${POSTGRES_USER}" \
  --dbname skawld <<'SQL'
CREATE EXTENSION vector;
REVOKE CREATE ON SCHEMA public FROM PUBLIC;
GRANT CONNECT ON DATABASE skawld TO skawld_app;
GRANT USAGE ON SCHEMA public TO skawld_app;
SQL
