SKAWLD MAINTENANCE - NATIVE BACKEND PACKAGE

Contents
--------

api       REST API process
worker    River background-worker process
migrate   PostgreSQL and River migration command
eval      Offline deterministic pilot-safety evaluation command
contracts/openapi.yaml
evaldata/pilot-v1.json
.env.example

Windows binaries use the .exe suffix.

These are backend/operator binaries, not a desktop application. PostgreSQL
with pgvector, an OIDC provider, and S3-compatible object storage remain
external dependencies. The native worker also requires the `pdftotext`
executable from Poppler on PATH (or PDFTOTEXT_BINARY set to its absolute path).
For local development, run server dependencies through compose.yaml in the
source repository.

Required startup order
----------------------

1. Copy .env.example to a private environment file and replace all
   development-only credentials before using a non-local installation.
2. Run:

   migrate up

3. Start api and worker as separate process roles.

Before a pilot demo or release, run:

   eval -dataset evaldata/pilot-v1.json

The command exits non-zero when an evidence, retrieval, workflow-match, or
unsafe-recommendation gate fails. It does not call an external AI provider.

Do not use development credentials outside a local workstation. Production
deployments should normally use the supplied Linux OCI images. Native Windows
and macOS packages are intended for development, integration testing, demos,
and explicitly qualified installations.

No executable path in this package controls PLC, SCADA, DCS, machinery, or
industrial safety systems.
