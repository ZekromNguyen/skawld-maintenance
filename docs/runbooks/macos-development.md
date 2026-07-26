# macOS Development

The Go API and worker run natively on macOS. PostgreSQL/pgvector and Keycloak
run as Linux containers through Docker Desktop or another Compose-compatible
runtime.

## Supported architectures

- Apple Silicon: `darwin/arm64`
- Intel Mac: `darwin/amd64`

The pinned PostgreSQL/pgvector and Keycloak container images publish both
`linux/amd64` and `linux/arm64` manifests.

## Prerequisites

- Go `1.25.12` or a compatible newer patch release.
- Docker Desktop with Docker Compose, or a compatible alternative such as
  Colima.
- Git and a POSIX shell.
- Flutter stable, Xcode command-line tools, and CocoaPods when required are
  additionally needed for the technician/workstation desktop application.

With Homebrew:

```bash
brew install go
```

Install Docker Desktop separately, or install and start a compatible container
runtime.

## Start the development environment

```bash
cp .env.example .env
./scripts/skawld.sh deps-up
./scripts/skawld.sh migrate
./scripts/skawld.sh api
```

In another terminal:

```bash
./scripts/skawld.sh worker
```

Verify:

```bash
curl --fail http://localhost:8080/health/live
curl --fail http://localhost:8080/health/ready
```

## Native packages

Build archives for macOS, Windows, and Linux:

```bash
./scripts/skawld.sh package v0.1.0
shasum -a 256 -c dist/packages/v0.1.0/SHA256SUMS
```

The macOS archives are not currently signed or notarized. They are suitable for
developer builds and controlled testing, not general end-user distribution.
Code signing and notarization become a release requirement before distributing
native binaries to enterprise users.

For the Flutter `.app`, DMG, and ZIP rather than the Go backend archives,
follow [Windows and macOS desktop application packages](./desktop-app-packages.md).

## Shutdown

```bash
./scripts/skawld.sh deps-down
```
