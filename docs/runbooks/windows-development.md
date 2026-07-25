# Windows Development

The Go API and worker can run natively in PowerShell. PostgreSQL/pgvector and
Keycloak run through Docker Desktop using Linux containers.

## Supported architectures

- Windows x64: `windows/amd64`
- Windows ARM64: `windows/arm64`

Windows x64 is the primary Windows development target. ARM64 archives are
provided for compatible Go and Windows environments.

## Prerequisites

- Windows 11 or a supported Windows 10 release.
- Go `1.25.7` or a compatible newer patch release.
- Git for Windows.
- Docker Desktop configured for Linux containers and Docker Compose.
- PowerShell 7 is recommended; Windows PowerShell 5.1 is sufficient for the
  provided script.

## Start from PowerShell

```powershell
Copy-Item .env.example .env
Set-ExecutionPolicy -Scope Process Bypass

.\scripts\skawld.ps1 deps-up
.\scripts\skawld.ps1 migrate
.\scripts\skawld.ps1 api
```

In another PowerShell window:

```powershell
Set-ExecutionPolicy -Scope Process Bypass
.\scripts\skawld.ps1 worker
```

Verify:

```powershell
Invoke-RestMethod http://localhost:8080/health/live
Invoke-RestMethod http://localhost:8080/health/ready
```

## Native packages

Build all platform archives:

```powershell
.\scripts\skawld.ps1 package v0.1.0
Get-Content .\dist\packages\v0.1.0\SHA256SUMS
```

Verify one archive:

```powershell
Get-FileHash -Algorithm SHA256 `
  .\dist\packages\v0.1.0\skawld-maintenance_v0.1.0_windows_amd64.zip
```

The Windows executables are not currently Authenticode-signed and no MSI
installer or Windows service wrapper is provided. These packages are for
development and controlled testing. Signing and service installation must be
designed before enterprise Windows distribution.

## Shutdown

```powershell
.\scripts\skawld.ps1 deps-down
```
