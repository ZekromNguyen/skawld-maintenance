# Local Development

This runbook targets Fedora/Linux with Go and Podman or Docker installed. The
application processes run natively; Compose runs dependencies only.

All credentials below are development-only.

## Prerequisites

- Go `1.25.7` or a compatible newer patch release.
- Podman with `podman compose`, or Docker with `docker compose`.
- `curl` and `make`.

## Start

```bash
cp .env.example .env
set -a
source .env
set +a

podman compose up -d postgres keycloak
go run ./cmd/migrate up
go run ./cmd/api
```

In a second shell with the same environment:

```bash
go run ./cmd/worker
```

Docker users replace `podman compose` with `docker compose`.

Verify:

```bash
curl --fail http://localhost:8080/health/live
curl --fail http://localhost:8080/health/ready
```

Open `http://localhost:8080/auth/login` in a browser and sign in with:

```text
username: phase0.admin
password: phase0-admin-dev
```

The development bootstrap subject may create the first organization. After the
first organization is created, normal authorization comes from memberships.
The bootstrap subject list is an installation/bootstrap mechanism, not a
production IAM model.

## Optional S3 contract-test service

S3Mock is deliberately optional and is only a local adapter-test target:

```bash
podman compose --profile objectstore up -d s3mock
```

It is not the default object-store product for deployment. Production and
on-premise installations must select and qualify an S3 implementation.

## Database roles

- `skawld_owner` runs migrations.
- `skawld_app` is used by both process roles at runtime.
- API and worker use distinct connection pools and configured budgets.
- River uses its own `river` schema and the worker process sets
  `search_path=river,public`.
- Runtime may insert and read audit events but cannot update or delete them.

## SDK workspace

Do not commit `go.work` or a `replace` directive. Once the SDK release gate in
ADR 0002 is satisfied, a developer may create a workspace above both clean
repositories:

```bash
cd ..
go work init ./skawld-maintenance ./skawld-sdk-go
```

CI and release builds must always resolve the pinned SDK tag without the
workspace.

## Checks and shutdown

```bash
make check
make build
podman compose down
```

To remove development data as an explicit destructive action:

```bash
podman compose down --volumes
```
