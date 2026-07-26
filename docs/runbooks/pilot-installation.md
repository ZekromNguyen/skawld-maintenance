# Single-Server Pilot Installation

This profile is for a controlled maintenance-company/private pilot. It uses a
modular monolith with separate API and worker process roles over one PostgreSQL
database. It does not require Kubernetes, Redis, Kafka, Elasticsearch, a
dedicated vector database, or a public AI provider.

## Required external decisions

Before installation, select and document:

- a supported OIDC provider and web/native clients;
- an S3 API implementation tested against Skawld's narrow object contract;
- TLS termination and DNS;
- encrypted host/storage/backup volumes;
- secret manager or root-owned environment-file procedure;
- retention, RPO/RTO, support hours, and incident contacts;
- whether a private AI/transcription endpoint is enabled. Empty provider
  configuration is valid; unavailable capabilities fail closed.

The included Keycloak and S3Mock profiles are development dependencies, not
pilot defaults.

## Install

1. Provision a supported Linux server with Podman Compose or Docker Compose.
2. Place TLS/reverse proxy in front of the loopback-bound web port.
3. Copy `deployments/pilot/.env.example` outside the repository, replace every
   secret, set mode `0600`, and never commit it.
4. Pin all images to reviewed digests in the customer deployment manifest.
5. Run:

```text
docker compose \
  --env-file /etc/skawld/pilot.env \
  -f deployments/pilot/compose.yaml \
  config

docker compose \
  --env-file /etc/skawld/pilot.env \
  -f deployments/pilot/compose.yaml \
  build

docker compose \
  --env-file /etc/skawld/pilot.env \
  -f deployments/pilot/compose.yaml \
  up -d
```

The one-shot migrator must complete before API/worker start. PostgreSQL is
bound to loopback for backup operations; API and worker are not exposed.

## Acceptance checks

- `/health/live` reports the expected version/commit/build time.
- `/health/ready` succeeds and becomes unavailable when PostgreSQL is stopped.
- OIDC login creates a secure HttpOnly session with the expected tenant/site.
- wrong-site API/search/attachment/workflow requests fail.
- deterministic provider works without internet; unconfigured speech fails
  closed.
- S3 put/head/get/signed upload/download contract tests pass.
- Phase 5 frozen evaluation gates pass.
- backup, empty-target restore, and checksum/count verification are rehearsed.
- supervisor can trace recommendation, report, and workflow evidence.

## Operations

Run API and worker with separate PostgreSQL pools. Initial pilot defaults are
8 API + 8 worker connections under a budget of 20. Lower worker concurrency
before adding queue infrastructure. Add external queue/storage only after load
evidence shows PostgreSQL OLTP contention.

The web container is HTTP-only and loopback-bound by default. Do not expose it
directly to an untrusted network.
