param(
    [string]$Version = "dev",
    [string]$BuildDirectory = "build/windows/x64/runner/Release",
    [string]$OutputDirectory = "dist/desktop"
)

$ErrorActionPreference = "Stop"
$configurationTemplate = Join-Path $PSScriptRoot `
    "../deployments/desktop/skawld-config.example.json"

if (-not (Test-Path -LiteralPath $BuildDirectory -PathType Container)) {
    throw "Windows release directory does not exist: $BuildDirectory"
}
if (-not (Test-Path -LiteralPath $configurationTemplate -PathType Leaf)) {
    throw "Desktop configuration template does not exist: $configurationTemplate"
}

New-Item -ItemType Directory -Force -Path $OutputDirectory | Out-Null
$archive = Join-Path $OutputDirectory "skawld-maintenance-$Version-windows-x64.zip"
$staging = Join-Path ([System.IO.Path]::GetTempPath()) (
    "skawld-maintenance-" + [guid]::NewGuid().ToString("N")
)

try {
    New-Item -ItemType Directory -Force -Path $staging | Out-Null
    Copy-Item -Path (Join-Path $BuildDirectory "*") -Destination $staging -Recurse
    Copy-Item -LiteralPath $configurationTemplate `
        -Destination (Join-Path $staging "skawld-config.example.json")

    if (Test-Path -LiteralPath $archive) {
        Remove-Item -LiteralPath $archive -Force
    }
    Compress-Archive -Path (Join-Path $staging "*") -DestinationPath $archive
} finally {
    if (Test-Path -LiteralPath $staging) {
        Remove-Item -LiteralPath $staging -Recurse -Force
    }
}

$hash = (Get-FileHash -Algorithm SHA256 -LiteralPath $archive).Hash.ToLowerInvariant()
"$hash  $(Split-Path -Leaf $archive)" |
    Set-Content -NoNewline -Encoding ascii -LiteralPath "$archive.sha256"

Write-Host "Created $archive"
