[CmdletBinding()]
param(
    [Parameter(Position = 0)]
    [ValidateSet("deps-up", "deps-down", "migrate", "api", "worker", "check", "package", "help")]
    [string]$Command = "help",

    [Parameter(Position = 1)]
    [ValidatePattern("^[A-Za-z0-9][A-Za-z0-9._-]*$")]
    [string]$Version = "dev"
)

$ErrorActionPreference = "Stop"
$RepositoryRoot = Split-Path -Parent $PSScriptRoot

function Assert-CommandSucceeded {
    param([string]$Description)
    if ($LASTEXITCODE -ne 0) {
        throw "$Description failed with exit code $LASTEXITCODE"
    }
}

function Import-DotEnv {
    $EnvironmentFile = Join-Path $RepositoryRoot ".env"
    if (-not (Test-Path $EnvironmentFile)) {
        throw ".env is missing; run: Copy-Item .env.example .env"
    }
    foreach ($Line in Get-Content $EnvironmentFile) {
        $Trimmed = $Line.Trim()
        if ($Trimmed.Length -eq 0 -or $Trimmed.StartsWith("#")) {
            continue
        }
        $Parts = $Trimmed.Split("=", 2)
        if ($Parts.Count -ne 2) {
            throw "Invalid .env line: $Line"
        }
        $Name = $Parts[0].Trim()
        $Value = $Parts[1].Trim()
        if (($Value.StartsWith('"') -and $Value.EndsWith('"')) -or
            ($Value.StartsWith("'") -and $Value.EndsWith("'"))) {
            $Value = $Value.Substring(1, $Value.Length - 2)
        }
        [Environment]::SetEnvironmentVariable($Name, $Value, "Process")
    }
}

Push-Location $RepositoryRoot
try {
    switch ($Command) {
        "deps-up" {
            docker compose up -d --wait postgres keycloak
            Assert-CommandSucceeded "Docker Compose startup"
        }
        "deps-down" {
            docker compose down
            Assert-CommandSucceeded "Docker Compose shutdown"
        }
        "migrate" {
            Import-DotEnv
            go run ./cmd/migrate up
            Assert-CommandSucceeded "Database migration"
        }
        "api" {
            Import-DotEnv
            go run ./cmd/api
            Assert-CommandSucceeded "API process"
        }
        "worker" {
            Import-DotEnv
            go run ./cmd/worker
            Assert-CommandSucceeded "Worker process"
        }
        "check" {
            go fmt ./...
            Assert-CommandSucceeded "Go formatting"
            go vet ./...
            Assert-CommandSucceeded "Go vet"
            go test ./...
            Assert-CommandSucceeded "Go tests"
            go build ./cmd/api ./cmd/worker ./cmd/migrate
            Assert-CommandSucceeded "Go build"
            docker compose config --quiet
            Assert-CommandSucceeded "Compose validation"
        }
        "package" {
            go run ./tools/package -version $Version -clean
            Assert-CommandSucceeded "Native package build"
        }
        default {
            @"
Usage: .\scripts\skawld.ps1 <command> [version]

Commands:
  deps-up             Start PostgreSQL and Keycloak through Docker Compose
  deps-down           Stop Compose dependencies
  migrate             Apply application and River migrations
  api                 Run the native Go API process
  worker              Run the native Go worker process
  check               Format, vet, test, build, and validate Compose
  package [version]   Build all Windows/macOS/Linux archives and checksums
"@
        }
    }
}
finally {
    Pop-Location
}
