# Native Package Contract

Skawld Maintenance remains a server product. Linux OCI images are the preferred
production distribution. Native archives support development, integration
testing, demonstrations, and explicitly qualified installations on Windows and
macOS.

## Build

From any platform with Go installed:

```text
go run ./tools/package -version <version> [-commit <sha>] [-built-at <RFC3339>] [-platforms windows,darwin,linux|all] [-clean]
```

The packaging tool uses `CGO_ENABLED=0`, `-trimpath`, and `-buildvcs=false`.
Build metadata is embedded in the binaries and exposed by API liveness.
The default is `-platforms all`. For Windows/macOS backend engineering
archives only, run:

```text
VERSION=<version> make package-backend-desktop
```

Artifacts:

```text
dist/packages/<version>/
├── SHA256SUMS
├── skawld-maintenance_<version>_windows_amd64.zip
├── skawld-maintenance_<version>_windows_arm64.zip
├── skawld-maintenance_<version>_darwin_amd64.tar.gz
├── skawld-maintenance_<version>_darwin_arm64.tar.gz
├── skawld-maintenance_<version>_linux_amd64.tar.gz
└── skawld-maintenance_<version>_linux_arm64.tar.gz
```

Each archive contains:

```text
api[.exe]
worker[.exe]
migrate[.exe]
eval[.exe]
README.txt
.env.example
contracts/openapi.yaml
evaldata/pilot-v1.json
```

The native worker requires Poppler's `pdftotext` executable. Install it
with the platform package manager and either place it on `PATH` or configure
`PDFTOTEXT_BINARY`. The OCI worker image includes this dependency. Object
storage remains an external deployment dependency.

## Release boundary

Current archives are unsigned engineering artifacts:

- macOS binaries are not code-signed or notarized;
- Windows executables are not Authenticode-signed;
- no MSI, PKG, launchd unit, or Windows service wrapper is included;
- dependency services are not bundled into native installers;
- no auto-update mechanism exists.

CI generates a dependency SBOM and tagged provenance, but that does not sign the
Windows/macOS executables. Before external enterprise distribution, define
signing identities, protected CI signing, notarization, support matrix,
upgrade/rollback, and service-management behavior. Do not weaken workstation
security controls to run an unverified artifact.
