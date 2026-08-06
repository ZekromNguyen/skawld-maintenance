# Local Development

This runbook targets Fedora/Linux with Go and Podman or Docker installed. The
application processes run natively; Compose runs dependencies only.

All credentials below are development-only.

## Prerequisites

- Go `1.25.12` or a compatible newer patch release.
- Podman with `podman compose`, or Docker with `docker compose`.
- `curl`, `make`, and Poppler's `pdftotext` for a native Phase 2 worker.

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

### Faster restarts after a reboot

`compose up` with `--wait` blocks until healthchecks pass, which is the slow
part (Keycloak 26 boots in 30-90s and the docker-compose provider adds
overhead). When the stack is already provisioned and nothing changed, start
the existing containers directly instead of re-running the full compose
up with its health gate:

```bash
make compose-resume        # podman compose start: no pulls, no rebuild, no wait
```

Containers also come back automatically at boot (no manual command at all).
The compose files set `restart: unless-stopped`; for rootless Podman, enable
the user-level restart unit and lingering so they start with the booted user
session:

```bash
loginctl enable-linger "$USER"                 # start user session at boot
systemctl --user enable --now podman-restart.service   # start stopped containers with a restart policy
```

After that, postgres and keycloak (and the s3mock profile if enabled) start
on their own after every reboot. Run `make compose-up` only when the compose
file, realm import, or image tags actually changed.

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

### Role test accounts

Run `make seed` once so every role gets a principal, membership, and approval
authority in the demo organization. Then sign in through the same Keycloak
login with any of these accounts (all passwords match their username pattern):

| Role                   | Username        | Password            | Subject ID |
|------------------------|-----------------|---------------------|------------|
| Administrator          | `phase0.admin`  | `phase0-admin-dev`  | `...0001`  |
| Administrator          | `dev.admin`     | `dev-admin-pw`      | `...0101`  |
| Maintenance Supervisor | `dev.supervisor`| `dev-supervisor-pw` | `...0102`  |
| Senior Technician      | `dev.senior`    | `dev-senior-pw`     | `...0103`  |
| Technician             | `dev.technician`| `dev-technician-pw` | `...0104`  |
| Manager                | `dev.manager`   | `dev-manager-pw`    | `...0105`  |

`skawld-realm.json` is a **development-only** realm: client secret and user
passwords are intentionally plaintext. Never import it into a production
Keycloak. Note: Keycloak's realm importer rejects unknown JSON fields, so do
not add comment fields to the realm file; document dev-only notes here instead.
Recreating the realm after editing the file requires a fresh Keycloak database
(`DROP DATABASE keycloak` in the postgres container, then restart keycloak).

The web console shows a "Sign out" button in the sidebar. Signing out revokes
the local web session cookie and returns to the login screen; Keycloak keeps
its own session, so the next login is a single click.

## Company SSO (Google Workspace / Microsoft Entra)

Skawld brokers sign-in through Keycloak identity providers. The realm file
ships with `google` and `microsoft` providers configured with placeholder
secrets and pinned scopes: Google to one hosted domain, Microsoft to one
Entra **tenant ID** (work and school accounts only, never `/organizations`).
Two layers enforce "only company accounts": the IdP pins above, and the API
`EMAIL_DOMAIN_ALLOWLIST` gate.

### 1. Register the OAuth applications

**Google (Google Cloud Console):**
- Create an OAuth consent screen (Internal) and an OAuth 2.0 Client ID of
  type Web for the project.
- Authorized redirect URI:
  `http://localhost:8081/realms/skawld/broker/google/endpoint`
- Note the client ID and secret.

**Microsoft (Entra / Azure portal):**
- Register an app; under Authentication add a Web platform redirect URI:
  `http://localhost:8081/realms/skawld/broker/microsoft/endpoint`
- Create a client secret; note the **tenant ID** of your directory.

### 2. Configure Keycloak

In the Keycloak admin console (`http://localhost:8081`, `admin` /
`admin_dev_only`), open Realm `skawld` -> Identity providers:
- **Google**: replace `clientId` / `clientSecret` with the real values and
  set `Hosted domain` to your Workspace domain (e.g. `acme-industrial.com`).
- **Microsoft**: replace `clientId` / `clientSecret`, and set `Tenant id`
  to your Entra tenant ID.
- No code changes are required: the placeholders only exist in the import
  file and are overridden in the admin console.

### 3. Grant roles

Product roles come from the Keycloak **`skawld_role`** user attribute, which
the `skawld-web` client maps into the `skawld_role` id_token claim. For each
employee: Users -> select the user -> Attributes -> add `skawld_role` with
one of `Administrator`, `Maintenance Supervisor`, `Senior Technician`,
`Technician`, `Manager`.

### 4. API environment

```bash
EMAIL_DOMAIN_ALLOWLIST=acme-industrial.com   # comma-separated; empty disables
FEDERATED_ORG_ID=<uuid of the demo organization>
```

With the allowlist set, a federated user whose email domain is not listed is
rejected (401). An allowlisted user without a membership is provisioned into
`FEDERATED_ORG_ID` with the role from their `skawld_role` attribute (site-less
membership, idempotent across logins). Bootstrap and seeded accounts are
exempt. When the allowlist is set, `FEDERATED_ORG_ID` must be a valid UUID.

### 5. Applying realm-file changes

Realm import uses `IGNORE_EXISTING`, so identity-provider or mapper changes in
`skawld-realm.json` require a fresh Keycloak database:

```bash
podman stop skawld-maintenance-keycloak-1
podman exec skawld-maintenance-postgres-1 psql -U postgres -c "DROP DATABASE keycloak WITH (FORCE);"
podman exec skawld-maintenance-postgres-1 psql -U postgres -c "CREATE DATABASE keycloak OWNER keycloak;"
podman start skawld-maintenance-keycloak-1
```

## Optional S3 contract-test service

S3Mock is deliberately optional and is only a local adapter-test target:

```bash
podman compose --profile objectstore up -d s3mock
```

It is not the default object-store product for deployment. Production and
on-premise installations must select and qualify an S3 implementation.

## Database roles

- PostgreSQL must have the `vector` extension installed by the image or a DBA
  before migrations run. The Compose bootstrap performs this privileged step;
  `skawld_owner` intentionally is not a superuser.
- `skawld_owner` runs migrations.
- `skawld_app` is used by both process roles at runtime.
- API and worker use distinct connection pools and configured budgets.
- River uses its own `river` schema and the worker process sets
  `search_path=river,public`.
- Runtime may insert and read audit events but cannot update or delete them.

## SDK workspace

Do not commit `go.work` or a `replace` directive. The release build pins SDK
`v0.2.0`; a developer may create a workspace above both clean repositories:

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
