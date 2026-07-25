SKAWLD MAINTENANCE - NATIVE BACKEND PACKAGE

Contents
--------

api       REST API process
worker    River background-worker process
migrate   PostgreSQL and River migration command
contracts/openapi.yaml
.env.example

Windows binaries use the .exe suffix.

These are backend/operator binaries, not a desktop application. PostgreSQL
with pgvector, an OIDC provider, and later an S3-compatible object store remain
external dependencies. For local development, run those dependencies through
compose.yaml in the source repository.

Required startup order
----------------------

1. Copy .env.example to a private environment file and replace all
   development-only credentials before using a non-local installation.
2. Run:

   migrate up

3. Start api and worker as separate process roles.

Do not use development credentials outside a local workstation. Production
deployments should normally use the supplied Linux OCI images. Native Windows
and macOS packages are intended for development, integration testing, demos,
and explicitly qualified installations.

No executable path in this package controls PLC, SCADA, DCS, machinery, or
industrial safety systems.
